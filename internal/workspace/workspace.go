// Package workspace prepara o repositório de um projeto (ADR-0061): clonar do GitHub,
// usar uma pasta existente ou criar do zero. Tudo acontece dentro de uma raiz
// configurável (WORKSPACES_DIR) e nenhum caminho de fora dela é aceito.
package workspace

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Erros com mensagem segura para o usuário.
var (
	ErrDisabled    = errors.New("a pasta de workspaces não está configurada (WORKSPACES_DIR)")
	ErrOutside     = errors.New("o caminho sai da pasta de workspaces")
	ErrNotRepo     = errors.New("a pasta não é um repositório git")
	ErrNotEmpty    = errors.New("a pasta já existe e não está vazia")
	ErrNotFound    = errors.New("a pasta não existe")
	ErrInvalidPath = errors.New("caminho inválido")
)

// Root é a raiz de workspaces. Dir vazio = recurso desligado.
type Root struct{ Dir string }

// Enabled diz se a raiz foi configurada.
func (r Root) Enabled() bool { return strings.TrimSpace(r.Dir) != "" }

func (r Root) real() (string, error) {
	if !r.Enabled() {
		return "", ErrDisabled
	}
	abs, err := filepath.Abs(r.Dir)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(abs, 0o755); err != nil {
		return "", fmt.Errorf("não consegui preparar a pasta de workspaces: %w", err)
	}
	return filepath.EvalSymlinks(abs)
}

func cleanRel(rel string) (string, error) {
	rel = strings.Trim(strings.TrimSpace(rel), "/")
	if rel == "" {
		return "", nil
	}
	if strings.ContainsAny(rel, "\\\x00") {
		return "", ErrInvalidPath
	}
	for _, seg := range strings.Split(rel, "/") {
		if seg == "" || seg == "." || seg == ".." {
			return "", ErrInvalidPath
		}
	}
	return rel, nil
}

func within(root, p string) bool {
	return p == root || strings.HasPrefix(p, root+string(filepath.Separator))
}

// Resolve devolve o caminho absoluto de rel dentro da raiz, mesmo que ainda não exista.
// Links simbólicos que apontem para fora da raiz são recusados.
func (r Root) Resolve(rel string) (string, error) {
	root, err := r.real()
	if err != nil {
		return "", err
	}
	rel, err = cleanRel(rel)
	if err != nil {
		return "", err
	}
	full := filepath.Join(root, filepath.FromSlash(rel))
	// Resolve o ancestral existente mais profundo e confere se continua dentro da raiz.
	probe, rest := full, ""
	for {
		if _, err := os.Lstat(probe); err == nil {
			break
		}
		parent := filepath.Dir(probe)
		if parent == probe {
			return "", ErrOutside
		}
		rest = filepath.Join(filepath.Base(probe), rest)
		probe = parent
	}
	resolved, err := filepath.EvalSymlinks(probe)
	if err != nil || !within(root, resolved) {
		return "", ErrOutside
	}
	return filepath.Join(resolved, rest), nil
}

// Entry é uma subpasta listada.
type Entry struct {
	Name   string `json:"name"`
	Path   string `json:"path"`
	IsRepo bool   `json:"is_repo"`
	Empty  bool   `json:"empty"`
}

// Listing é o conteúdo de uma pasta.
type Listing struct {
	Enabled bool    `json:"enabled"`
	Path    string  `json:"path"`
	Parent  *string `json:"parent"`
	IsRepo  bool    `json:"is_repo"`
	Entries []Entry `json:"entries"`
}

// Browse lista as subpastas de rel (sem pastas ocultas). Só devolve caminhos relativos.
func (r Root) Browse(rel string) (Listing, error) {
	if !r.Enabled() {
		return Listing{Enabled: false, Entries: []Entry{}}, nil
	}
	abs, err := r.Resolve(rel)
	if err != nil {
		return Listing{}, err
	}
	rel, _ = cleanRel(rel)
	items, err := os.ReadDir(abs)
	if errors.Is(err, os.ErrNotExist) {
		if rel == "" {
			items = nil
		} else {
			return Listing{}, ErrNotFound
		}
	} else if err != nil {
		return Listing{}, err
	}
	out := Listing{Enabled: true, Path: rel, IsRepo: isRepo(abs), Entries: []Entry{}}
	if rel != "" {
		parent := ""
		if i := strings.LastIndex(rel, "/"); i >= 0 {
			parent = rel[:i]
		}
		out.Parent = &parent
	}
	for _, it := range items {
		if !it.IsDir() || strings.HasPrefix(it.Name(), ".") {
			continue
		}
		child := filepath.Join(abs, it.Name())
		// Link simbólico para fora da raiz não aparece.
		if real, err := filepath.EvalSymlinks(child); err != nil || !within(mustReal(r), real) {
			continue
		}
		p := it.Name()
		if rel != "" {
			p = rel + "/" + it.Name()
		}
		out.Entries = append(out.Entries, Entry{Name: it.Name(), Path: p, IsRepo: isRepo(child), Empty: isEmpty(child)})
	}
	sort.Slice(out.Entries, func(i, j int) bool {
		return strings.ToLower(out.Entries[i].Name) < strings.ToLower(out.Entries[j].Name)
	})
	return out, nil
}

func mustReal(r Root) string {
	d, _ := r.real()
	return d
}

func isRepo(dir string) bool {
	st, err := os.Stat(filepath.Join(dir, ".git"))
	return err == nil && (st.IsDir() || st.Mode().IsRegular())
}

func isEmpty(dir string) bool {
	items, err := os.ReadDir(dir)
	return err == nil && len(items) == 0
}

// IsRepo diz se dir é a raiz de um repositório git.
func IsRepo(dir string) bool { return isRepo(dir) }

// IsEmptyOrMissing diz se dir não existe ou é uma pasta vazia.
func IsEmptyOrMissing(dir string) bool {
	if _, err := os.Stat(dir); errors.Is(err, os.ErrNotExist) {
		return true
	}
	return isEmpty(dir)
}
