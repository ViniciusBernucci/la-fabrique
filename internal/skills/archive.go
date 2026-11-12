package skills

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"errors"
	"io"
	"os"
	"regexp"
	"sort"
	"strings"
)

// Limites de um pacote.
const (
	MaxFiles    = 200
	MaxFileSize = 1 << 20
	MaxTotal    = 5 << 20
	MaxArchive  = 8 << 20
)

// File é um arquivo do pacote (já com caminho validado).
type File struct {
	Path    string
	Content []byte
}

// ImportError é uma recusa de importação, com código estável e o caminho envolvido.
type ImportError struct {
	Code    string
	Path    string
	Message string
}

func (e *ImportError) Error() string {
	if e.Path != "" {
		return e.Message + " (" + e.Path + ")"
	}
	return e.Message
}

func imp(code, path, msg string) error { return &ImportError{Code: code, Path: path, Message: msg} }

var drive = regexp.MustCompile(`^[A-Za-z]:`)

// CleanPath valida um caminho relativo de arquivo do pacote: nada de absoluto,
// "..", ".", barras invertidas, NUL, diretório .git ou caminhos vazios.
func CleanPath(p string) (string, error) {
	if p == "" || len(p) > 300 || strings.ContainsRune(p, 0) || strings.Contains(p, `\`) ||
		strings.HasPrefix(p, "/") || drive.MatchString(p) {
		return "", imp("bad_path", clipPath(p), "Caminho inválido no pacote.")
	}
	p = strings.TrimSuffix(p, "/")
	for _, el := range strings.Split(p, "/") {
		if el == "" || el == "." || el == ".." || el == ".git" {
			return "", imp("bad_path", clipPath(p), "Caminho inválido no pacote (travessia de diretório ou elemento proibido).")
		}
	}
	return p, nil
}

func clipPath(p string) string {
	p = strings.Map(func(r rune) rune {
		if r < 32 {
			return '?'
		}
		return r
	}, p)
	if len(p) > 80 {
		return p[:80] + "…"
	}
	return p
}

type collector struct {
	files []File
	seen  map[string]bool
	total int
}

func (c *collector) add(path string, content []byte) error {
	clean, err := CleanPath(path)
	if err != nil {
		return err
	}
	key := strings.ToLower(clean)
	if c.seen[key] {
		return imp("duplicate_path", clean, "Há dois arquivos com o mesmo caminho no pacote.")
	}
	c.seen[key] = true
	if len(c.files) >= MaxFiles {
		return imp("too_many_files", "", "O pacote tem arquivos demais.")
	}
	c.total += len(content)
	if c.total > MaxTotal {
		return imp("package_too_large", "", "O pacote descompactado é grande demais.")
	}
	c.files = append(c.files, File{Path: clean, Content: content})
	return nil
}

func readLimited(r io.Reader, path string) ([]byte, error) {
	b, err := io.ReadAll(io.LimitReader(r, MaxFileSize+1))
	if err != nil {
		return nil, imp("bad_archive", path, "Falha ao ler o arquivo do pacote.")
	}
	if len(b) > MaxFileSize {
		return nil, imp("file_too_large", path, "Um arquivo do pacote é grande demais.")
	}
	return b, nil
}

// ReadArchive extrai um pacote "tar.gz" ou "zip" para a memória, sem tocar o
// disco. Recusa links simbólicos, links físicos, dispositivos e demais entradas
// especiais; confere o tamanho real dos dados (não o declarado) e os limites.
func ReadArchive(r io.Reader, format string) ([]File, error) {
	raw, err := io.ReadAll(io.LimitReader(r, MaxArchive+1))
	if err != nil {
		return nil, imp("bad_archive", "", "Falha ao ler o pacote.")
	}
	if len(raw) > MaxArchive {
		return nil, imp("package_too_large", "", "O arquivo do pacote é grande demais.")
	}
	c := &collector{seen: map[string]bool{}}
	switch format {
	case "tar.gz":
		gz, err := gzip.NewReader(bytes.NewReader(raw))
		if err != nil {
			return nil, imp("bad_archive", "", "O pacote não é um tar.gz válido.")
		}
		tr := tar.NewReader(gz)
		for {
			h, err := tr.Next()
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				return nil, imp("bad_archive", "", "O pacote tar está corrompido.")
			}
			switch h.Typeflag {
			case tar.TypeDir, tar.TypeXGlobalHeader:
				continue
			case tar.TypeReg:
			default:
				return nil, imp("bad_entry", clipPath(h.Name), "O pacote contém um link, dispositivo ou entrada especial: recusado.")
			}
			if h.Size > MaxFileSize {
				return nil, imp("file_too_large", clipPath(h.Name), "Um arquivo do pacote é grande demais.")
			}
			body, err := readLimited(tr, clipPath(h.Name))
			if err != nil {
				return nil, err
			}
			if err := c.add(h.Name, body); err != nil {
				return nil, err
			}
		}
	case "zip":
		zr, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
		if err != nil {
			return nil, imp("bad_archive", "", "O pacote não é um zip válido.")
		}
		for _, f := range zr.File {
			mode := f.Mode()
			if f.FileInfo().IsDir() {
				continue
			}
			if mode&os.ModeSymlink != 0 || !mode.IsRegular() {
				return nil, imp("bad_entry", clipPath(f.Name), "O pacote contém um link ou entrada especial: recusado.")
			}
			if f.UncompressedSize64 > MaxFileSize {
				return nil, imp("file_too_large", clipPath(f.Name), "Um arquivo do pacote é grande demais.")
			}
			rc, err := f.Open()
			if err != nil {
				return nil, imp("bad_archive", clipPath(f.Name), "Falha ao abrir o arquivo do pacote.")
			}
			body, err := readLimited(rc, clipPath(f.Name))
			_ = rc.Close()
			if err != nil {
				return nil, err
			}
			if err := c.add(f.Name, body); err != nil {
				return nil, err
			}
		}
	default:
		return nil, imp("bad_format", "", "Formato de pacote não suportado (use tar.gz ou zip).")
	}
	return stripRoot(c.files), nil
}

// stripRoot remove o diretório raiz único que exportações do GitHub costumam ter.
func stripRoot(files []File) []File {
	if len(files) == 0 {
		return files
	}
	root, _, ok := strings.Cut(files[0].Path, "/")
	if !ok {
		return files
	}
	for _, f := range files {
		if r, _, ok := strings.Cut(f.Path, "/"); !ok || r != root {
			return files
		}
	}
	out := make([]File, len(files))
	for i, f := range files {
		out[i] = File{Path: strings.TrimPrefix(f.Path, root+"/"), Content: f.Content}
	}
	return out
}

func sortFiles(files []File) {
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
}
