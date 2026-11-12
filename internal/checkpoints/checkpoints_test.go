package checkpoints

import (
	"archive/tar"
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
)

type fakeWS struct {
	tarData []byte
	tarErr  error // erro devolvido por wait() (tar terminou mal)
	git     map[string][2]string
	seeded  []byte
}

func (f *fakeWS) Tar(context.Context) (io.ReadCloser, func() error, error) {
	return io.NopCloser(bytes.NewReader(f.tarData)), func() error { return f.tarErr }, nil
}

func (f *fakeWS) Run(_ context.Context, argv []string) (string, int, error) {
	key := strings.Join(argv[3:], " ")
	if r, ok := f.git[key]; ok {
		code := 0
		if r[1] != "" {
			code = 1
		}
		return r[0], code, nil
	}
	return "", 127, nil
}

func (f *fakeWS) Seed(_ context.Context, r io.Reader) error {
	b, err := io.ReadAll(r)
	f.seeded = b
	return err
}

type entry struct {
	name, body, link string
	typ              byte
	mode             int64
}

func mkTar(es ...entry) []byte {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	for _, e := range es {
		h := &tar.Header{Name: e.name, Typeflag: e.typ, Mode: e.mode, Size: int64(len(e.body)), Linkname: e.link}
		if e.typ != tar.TypeReg {
			h.Size = 0
		}
		_ = tw.WriteHeader(h)
		if e.typ == tar.TypeReg {
			_, _ = tw.Write([]byte(e.body))
		}
	}
	_ = tw.Close()
	return buf.Bytes()
}

func TestValidPath(t *testing.T) {
	for _, ok := range []string{"a", "a/b.txt", ".git/HEAD", "dir com espaço/x"} {
		if !ValidPath(ok) {
			t.Errorf("%q deveria valer", ok)
		}
	}
	for _, bad := range []string{"", "/etc/passwd", "../x", "a/../b", "a//b", "./a", ".", "..", "a\x00b", "a\nb", `a\b`, "a/", strings.Repeat("a", 2000)} {
		if ValidPath(bad) {
			t.Errorf("%q deveria ser recusado", bad)
		}
	}
}

func TestTreeHashIdentityIgnoresOrderAndNonContentMetadata(t *testing.T) {
	a := []FileEntry{{Path: "a", Kind: KindFile, Mode: 0o644, Size: 1, SHA256: "x"}, {Path: "d", Kind: KindDir, Mode: 0o755}, {Path: "l", Kind: KindSymlink, Link: "a", Mode: 0o777}}
	b := []FileEntry{a[2], a[0], a[1]}
	if TreeHash(a) != TreeHash(b) {
		t.Fatal("a ordem não importa")
	}
	c := append([]FileEntry{}, a...)
	c[1].Mode = 0o700 // permissão de diretório não conta
	if TreeHash(a) != TreeHash(c) {
		t.Fatal("permissão de diretório fora da identidade")
	}
	d := append([]FileEntry{}, a...)
	d[0].Mode = 0o755 // permissão de arquivo conta
	if TreeHash(a) == TreeHash(d) {
		t.Fatal("permissão de arquivo é identidade")
	}
	e := append([]FileEntry{}, a...)
	e[2].Link = "outro"
	if TreeHash(a) == TreeHash(e) {
		t.Fatal("alvo do link é identidade")
	}
}

func TestCaptureBuildsManifestWithHashesLinksAndDedup(t *testing.T) {
	ws := &fakeWS{tarData: mkTar(
		entry{name: "./", typ: tar.TypeDir, mode: 0o755},
		entry{name: "./src/", typ: tar.TypeDir, mode: 0o755},
		entry{name: "./src/a.txt", body: "alfa", typ: tar.TypeReg, mode: 0o644},
		entry{name: "./src/b.txt", body: "alfa", typ: tar.TypeReg, mode: 0o755},
		entry{name: "./src/hard", typ: tar.TypeLink, link: "./src/a.txt"},
		entry{name: "./src/sym", typ: tar.TypeSymlink, link: "/etc/passwd"},
		entry{name: "./fifo", typ: tar.TypeFifo, mode: 0o644},
	)}
	snap, err := Capture(t.Context(), ws, Options{})
	if err != nil {
		t.Fatal(err)
	}
	m := snap.Manifest
	if err := m.Validate(); err != nil {
		t.Fatal(err)
	}
	if len(m.Files) != 5 { // src, a, b, hard, sym
		t.Fatalf("%d entradas: %+v", len(m.Files), m.Files)
	}
	if len(snap.Objects) != 1 || m.TotalBytes != 12 {
		t.Fatalf("o mesmo conteúdo vira um objeto: %d objetos, %d bytes", len(snap.Objects), m.TotalBytes)
	}
	by := map[string]FileEntry{}
	for _, f := range m.Files {
		by[f.Path] = f
	}
	if by["src/a.txt"].SHA256 != by["src/hard"].SHA256 || by["src/sym"].Link != "/etc/passwd" || by["src/b.txt"].Mode != 0o755 {
		t.Fatalf("%+v", by)
	}
	if len(m.Skipped) != 1 || m.Skipped[0] != "fifo" || m.Git.Available || m.Git.Note == "" {
		t.Fatalf("pulados/git: %+v %+v", m.Skipped, m.Git)
	}
	// O link para fora fica como link (não é seguido nem lido).
	if _, ok := snap.Objects[sum([]byte("root"))]; ok {
		t.Fatal("conteúdo de fora do workspace não pode entrar")
	}
}

func TestCaptureRefusesTooLargeBadPathsAndBrokenStreams(t *testing.T) {
	big := strings.Repeat("x", MaxFileBytes+1)
	if _, err := Capture(t.Context(), &fakeWS{tarData: mkTar(entry{name: "./big", body: big, typ: tar.TypeReg, mode: 0o644})}, Options{}); !errors.Is(err, ErrTooLarge) {
		t.Fatalf("arquivo grande: %v", err)
	}
	if _, err := Capture(t.Context(), &fakeWS{tarData: mkTar(entry{name: "../fora", body: "x", typ: tar.TypeReg, mode: 0o644})}, Options{}); !errors.Is(err, ErrBadPath) {
		t.Fatalf("caminho: %v", err)
	}
	// Tar cortado no meio: nada de checkpoint parcial.
	full := mkTar(entry{name: "./a", body: strings.Repeat("a", 5000), typ: tar.TypeReg, mode: 0o644})
	if _, err := Capture(t.Context(), &fakeWS{tarData: full[:2000]}, Options{}); err == nil {
		t.Fatal("tar cortado deve falhar")
	}
	// O comando terminou com erro depois de entregar um tar aparentemente válido.
	if _, err := Capture(t.Context(), &fakeWS{tarData: full, tarErr: errors.New("código 1")}, Options{}); err == nil {
		t.Fatal("saída com erro deve falhar")
	}
}

func TestCaptureRecordsGitWhenAvailable(t *testing.T) {
	sha := strings.Repeat("a1", 20)
	ws := &fakeWS{tarData: mkTar(entry{name: "./a", body: "x", typ: tar.TypeReg, mode: 0o644}), git: map[string][2]string{
		"rev-parse --is-inside-work-tree":      {"true\n", ""},
		"rev-parse HEAD":                       {sha + "\n", ""},
		"rev-parse --abbrev-ref HEAD":          {"feature/x\n", ""},
		"diff --binary HEAD":                   {"diff --git a/a b/a\n+x\n", ""},
		"ls-files --others --exclude-standard": {"novo.txt\ndir/outro.txt\n../fora\n", ""},
	}}
	snap, err := Capture(t.Context(), ws, Options{})
	if err != nil {
		t.Fatal(err)
	}
	g := snap.Manifest.Git
	if !g.Available || g.Head != sha || g.Branch != "feature/x" || g.PatchSize == 0 || len(g.Untracked) != 2 {
		t.Fatalf("%+v", g)
	}
	if string(snap.Objects[g.PatchHash]) != "diff --git a/a b/a\n+x\n" {
		t.Fatal("o patch é um objeto do checkpoint")
	}
	// Repositório que não é git ou HEAD estranho.
	ws.git["rev-parse --is-inside-work-tree"] = [2]string{"", "fatal"}
	if s, _ := Capture(t.Context(), ws, Options{}); s.Manifest.Git.Available {
		t.Fatal("fora de repositório")
	}
	ws.git["rev-parse --is-inside-work-tree"] = [2]string{"true", ""}
	ws.git["rev-parse HEAD"] = [2]string{"lixo", ""}
	if s, _ := Capture(t.Context(), ws, Options{}); s.Manifest.Git.Head != "" || s.Manifest.Git.PatchHash != "" {
		t.Fatalf("SHA fora do formato não é registrado: %+v", s.Manifest.Git)
	}
}

func TestManifestValidateCatchesTampering(t *testing.T) {
	snap, err := Capture(t.Context(), &fakeWS{tarData: mkTar(entry{name: "./a", body: "x", typ: tar.TypeReg, mode: 0o644})}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	ok := snap.Manifest
	mutate := map[string]func(m *Manifest){
		"hash do arquivo": func(m *Manifest) {
			m.Files = []FileEntry{{Path: "a", Kind: KindFile, Mode: 0o644, Size: 1, SHA256: strings.Repeat("0", 64)}}
		},
		"tamanho": func(m *Manifest) { m.TotalBytes = 99 },
		"caminho": func(m *Manifest) {
			m.Files = []FileEntry{{Path: "../x", Kind: KindDir}}
			m.TreeHash = TreeHash(m.Files)
		},
		"duplicado": func(m *Manifest) {
			m.Files = append(m.Files, m.Files[0])
			m.TotalBytes = 2
			m.TreeHash = TreeHash(m.Files)
		},
		"tipo": func(m *Manifest) {
			m.Files = []FileEntry{{Path: "a", Kind: "socket"}}
			m.TreeHash = TreeHash(m.Files)
			m.TotalBytes = 0
		},
		"versão": func(m *Manifest) { m.Version = 2 },
	}
	for name, f := range mutate {
		m := ok
		m.Files = append([]FileEntry{}, ok.Files...)
		f(&m)
		if m.Validate() == nil {
			t.Errorf("%s: adulteração não detectada", name)
		}
	}
	if err := ok.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestRestoreTarCarriesEntriesInOrderWithSafeModes(t *testing.T) {
	snap, _ := Capture(t.Context(), &fakeWS{tarData: mkTar(
		entry{name: "./d/", typ: tar.TypeDir, mode: 0o500},
		entry{name: "./d/f", body: "dados", typ: tar.TypeReg, mode: 0o4755}, // setuid não sobrevive
		entry{name: "./d/l", typ: tar.TypeSymlink, link: "f"},
	)}, Options{})
	r := tarOf(snap.Manifest, func(h string) ([]byte, error) { return snap.Objects[h], nil })
	tr := tar.NewReader(r)
	var names []string
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		names = append(names, h.Name)
		if h.Mode&0o7000 != 0 {
			t.Fatalf("bits especiais em %s: %o", h.Name, h.Mode)
		}
		if h.Typeflag == tar.TypeDir && h.Mode != 0o755 {
			t.Fatalf("diretório precisa ser gravável na extração: %o", h.Mode)
		}
	}
	if strings.Join(names, ",") != "d/,d/f,d/l" {
		t.Fatalf("%v", names)
	}
	// Objeto corrompido interrompe o tar com erro (não entrega conteúdo errado).
	bad := tarOf(snap.Manifest, func(string) ([]byte, error) { return nil, ErrCorrupt })
	if _, err := io.ReadAll(bad); !errors.Is(err, ErrCorrupt) {
		t.Fatalf("%v", err)
	}
}
