package checkpoints

import (
	"archive/tar"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/ViniciusBernucci/le-fabrique/internal/execution"
)

// Workspace é o que o checkpoint precisa do ambiente de execução.
type Workspace interface {
	// Tar devolve o tar do workspace e uma função que, chamada DEPOIS de ler tudo,
	// confirma que o comando terminou bem (um tar cortado não vale).
	Tar(ctx context.Context) (io.ReadCloser, func() error, error)
	// Run executa um comando (argv, sem shell) e devolve stdout e o código de saída.
	Run(ctx context.Context, argv []string) (string, int, error)
	// Seed extrai um tar no workspace.
	Seed(ctx context.Context, r io.Reader) error
}

// SandboxWorkspace liga o checkpoint ao sandbox Docker.
type SandboxWorkspace struct{ S *execution.Sandbox }

// Tar implementa Workspace.
func (w SandboxWorkspace) Tar(ctx context.Context) (io.ReadCloser, func() error, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	r, err := w.S.Start(ctx, execution.ExecSpec{Cmd: []string{"tar", "-c", "-C", "/workspace", "."}})
	if err != nil {
		cancel()
		return nil, nil, err
	}
	wait := func() error {
		defer cancel()
		code, werr := r.Wait()
		if werr != nil || code != 0 {
			return fmt.Errorf("tar do workspace terminou com erro (código %d)", code)
		}
		return nil
	}
	return cancelOnClose{r.Stdout, cancel}, wait, nil
}

// cancelOnClose: fechar o leitor antes do fim cancela o comando (e mata a árvore
// de processos), em vez de deixá-lo bloqueado escrevendo num pipe sem leitor.
type cancelOnClose struct {
	io.ReadCloser
	cancel context.CancelFunc
}

func (c cancelOnClose) Close() error {
	c.cancel()
	return c.ReadCloser.Close()
}

// Run implementa Workspace.
func (w SandboxWorkspace) Run(ctx context.Context, argv []string) (string, int, error) {
	res, err := w.S.Exec(ctx, execution.ExecSpec{Cmd: argv, Timeout: time.Minute})
	if err != nil {
		return "", 0, err
	}
	if res.TimedOut {
		return "", -1, errors.New("tempo limite")
	}
	return res.Stdout, res.ExitCode, nil
}

// Seed implementa Workspace.
func (w SandboxWorkspace) Seed(ctx context.Context, r io.Reader) error { return w.S.Seed(ctx, r) }

// Options ajusta a captura.
type Options struct {
	Git string // binário do git dentro do sandbox (padrão "git")
}

// Snapshot é o resultado da captura: manifesto e objetos por hash. Ainda não é
// um checkpoint: só passa a existir quando o Store o grava por inteiro.
type Snapshot struct {
	Manifest Manifest
	Objects  map[string][]byte
}

func cleanName(n string) string {
	n = strings.TrimPrefix(n, "./")
	return strings.TrimSuffix(n, "/")
}

// Capture lê o workspace e monta o manifesto. Acima dos limites, recusa (ErrTooLarge).
func Capture(ctx context.Context, ws Workspace, opt Options) (Snapshot, error) {
	rc, wait, err := ws.Tar(ctx)
	if err != nil {
		return Snapshot{}, fmt.Errorf("ler o workspace: %w", err)
	}
	defer rc.Close()
	snap := Snapshot{Objects: map[string][]byte{}}
	byPath := map[string]FileEntry{}
	var total int64
	tr := tar.NewReader(rc)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			_ = rc.Close() // cancela o comando antes de esperar por ele
			_ = wait()
			return Snapshot{}, fmt.Errorf("ler o tar do workspace: %w", err)
		}
		name := cleanName(hdr.Name)
		if name == "" || name == "." {
			continue
		}
		if !ValidPath(name) {
			return Snapshot{}, fmt.Errorf("%w: %q", ErrBadPath, name)
		}
		if len(byPath) >= MaxFiles {
			return Snapshot{}, ErrTooLarge
		}
		e := FileEntry{Path: name, Mode: hdr.Mode & 0o777}
		switch hdr.Typeflag {
		case tar.TypeDir:
			e.Kind = KindDir
		case tar.TypeSymlink:
			e.Kind, e.Link = KindSymlink, hdr.Linkname
		case tar.TypeReg:
			if hdr.Size > MaxFileBytes || total+hdr.Size > MaxTotalBytes {
				return Snapshot{}, ErrTooLarge
			}
			data, err := io.ReadAll(io.LimitReader(tr, hdr.Size+1))
			if err != nil || int64(len(data)) != hdr.Size {
				return Snapshot{}, fmt.Errorf("ler %q: conteúdo incompleto", name)
			}
			e.Kind, e.Size, e.SHA256 = KindFile, hdr.Size, sum(data)
			snap.Objects[e.SHA256] = data
			total += hdr.Size
		case tar.TypeLink: // link físico: mesmo conteúdo do alvo
			target, ok := byPath[cleanName(hdr.Linkname)]
			if !ok || target.Kind != KindFile {
				snap.Manifest.Skipped = appendCapped(snap.Manifest.Skipped, name)
				continue
			}
			e.Kind, e.Size, e.SHA256 = KindFile, target.Size, target.SHA256
			total += target.Size
			if total > MaxTotalBytes {
				return Snapshot{}, ErrTooLarge
			}
		default: // fifo, dispositivo, socket
			snap.Manifest.Skipped = appendCapped(snap.Manifest.Skipped, name)
			continue
		}
		byPath[name] = e
	}
	if err := wait(); err != nil {
		return Snapshot{}, err
	}
	for _, e := range byPath {
		snap.Manifest.Files = append(snap.Manifest.Files, e)
	}
	sort.Slice(snap.Manifest.Files, func(i, j int) bool { return snap.Manifest.Files[i].Path < snap.Manifest.Files[j].Path })
	snap.Manifest.Version, snap.Manifest.TotalBytes = 1, total
	snap.Manifest.TreeHash = TreeHash(snap.Manifest.Files)
	gitBin := opt.Git
	if gitBin == "" {
		gitBin = "git"
	}
	snap.Manifest.Git = captureGit(ctx, ws, gitBin, snap.Objects)
	return snap, nil
}

func appendCapped(list []string, s string) []string {
	if len(list) >= 50 {
		return list
	}
	return append(list, s)
}

// captureGit registra SHA, ramo, patch e não rastreados quando há git e repositório.
// Qualquer falha vira uma nota: o conteúdo já está na árvore.
func captureGit(ctx context.Context, ws Workspace, bin string, objects map[string][]byte) GitInfo {
	run := func(args ...string) (string, int, error) {
		return ws.Run(ctx, append([]string{bin, "-C", "/workspace"}, args...))
	}
	out, code, err := run("rev-parse", "--is-inside-work-tree")
	if err != nil || code == 127 || code == 126 {
		return GitInfo{Note: "git indisponível no ambiente"}
	}
	if code != 0 || strings.TrimSpace(out) != "true" {
		return GitInfo{Note: "o workspace não é um repositório git"}
	}
	g := GitInfo{Available: true}
	if o, c, e := run("rev-parse", "HEAD"); e == nil && c == 0 {
		g.Head = strings.TrimSpace(o)
		if len(g.Head) != 40 && len(g.Head) != 64 {
			g.Head, g.Note = "", "SHA de HEAD em formato inesperado"
		}
	} else {
		g.Note = "repositório sem commits"
	}
	if o, c, e := run("rev-parse", "--abbrev-ref", "HEAD"); e == nil && c == 0 {
		g.Branch = strings.TrimSpace(o)
	}
	if g.Head != "" {
		if o, c, e := run("diff", "--binary", "HEAD"); e == nil && c == 0 && o != "" {
			if len(o) > MaxPatchBytes {
				g.Note = strings.TrimSpace(g.Note + " patch grande demais para guardar (a árvore já o contém)")
			} else {
				b := []byte(o)
				g.PatchHash, g.PatchSize = sum(b), len(b)
				objects[g.PatchHash] = b
			}
		}
	}
	if o, c, e := run("ls-files", "--others", "--exclude-standard"); e == nil && c == 0 {
		for _, l := range strings.Split(strings.TrimSpace(o), "\n") {
			if l = strings.TrimSpace(l); l != "" && ValidPath(path.Clean(l)) && len(g.Untracked) < 1000 {
				g.Untracked = append(g.Untracked, l)
			}
		}
	}
	return g
}

// tarOf monta o tar de restauração a partir do manifesto; load devolve o conteúdo
// (já verificado) de cada objeto.
func tarOf(m Manifest, load func(hash string) ([]byte, error)) io.Reader {
	pr, pw := io.Pipe()
	go func() {
		tw := tar.NewWriter(pw)
		fail := func(err error) { _ = pw.CloseWithError(err) }
		for _, e := range m.Files { // ordenado: o diretório vem antes do conteúdo
			hdr := &tar.Header{Name: e.Path, ModTime: time.Unix(0, 0), Format: tar.FormatPAX}
			switch e.Kind {
			case KindDir:
				hdr.Typeflag, hdr.Mode = tar.TypeDir, 0o755
				hdr.Name += "/"
				if err := tw.WriteHeader(hdr); err != nil {
					fail(err)
					return
				}
			case KindSymlink:
				hdr.Typeflag, hdr.Linkname, hdr.Mode = tar.TypeSymlink, e.Link, 0o777
				if err := tw.WriteHeader(hdr); err != nil {
					fail(err)
					return
				}
			case KindFile:
				data, err := load(e.SHA256)
				if err != nil {
					fail(err)
					return
				}
				hdr.Typeflag, hdr.Mode, hdr.Size = tar.TypeReg, e.Mode&0o777, int64(len(data))
				if err := tw.WriteHeader(hdr); err != nil {
					fail(err)
					return
				}
				if _, err := io.Copy(tw, bytes.NewReader(data)); err != nil {
					fail(err)
					return
				}
			}
		}
		if err := tw.Close(); err != nil {
			fail(err)
			return
		}
		_ = pw.Close()
	}()
	return pr
}
