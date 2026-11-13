package checkpoints

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"sort"
	"strings"
)

// Limites de um checkpoint. Acima deles a captura é RECUSADA (nunca parcial).
const (
	MaxFileBytes  = 16 << 20
	MaxTotalBytes = 128 << 20
	MaxFiles      = 20000
	MaxPatchBytes = 4<<20 - 1 // abaixo do limite de saída do sandbox (4 MiB): assim um patch truncado é detectável
)

// Erros.
var (
	ErrTooLarge     = errors.New("o workspace excede os limites do checkpoint")
	ErrCorrupt      = errors.New("checkpoint corrompido: um hash não confere")
	ErrMismatch     = errors.New("o workspace restaurado difere do checkpoint")
	ErrNotQuiescent = errors.New("o sandbox ainda tem processos vivos: o checkpoint seria inconsistente")
	ErrNotEmpty     = errors.New("o workspace de destino não está vazio")
	ErrBadPath      = errors.New("caminho inválido no checkpoint")
)

// Tipos de entrada.
const (
	KindFile    = "file"
	KindSymlink = "symlink"
	KindDir     = "dir"
)

// FileEntry é uma entrada do workspace. Mtime, dono e bits especiais não fazem
// parte da identidade.
type FileEntry struct {
	Path   string `json:"path"`
	Kind   string `json:"kind"`
	Mode   int64  `json:"mode"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256,omitempty"` // arquivo
	Link   string `json:"link,omitempty"`   // alvo do link (guardado como está)
}

// GitInfo é o que o Git do workspace informou. Available=false quando não há git
// ou repositório (o conteúdo continua capturado pela árvore).
type GitInfo struct {
	Available bool     `json:"available"`
	Head      string   `json:"head,omitempty"`
	Branch    string   `json:"branch,omitempty"`
	PatchHash string   `json:"patch_hash,omitempty"` // objeto com `git diff --binary HEAD`
	PatchSize int      `json:"patch_size,omitempty"`
	Untracked []string `json:"untracked,omitempty"`
	Note      string   `json:"note,omitempty"`
}

// Manifest descreve o checkpoint do workspace.
type Manifest struct {
	Version    int         `json:"version"`
	Files      []FileEntry `json:"files"`
	TreeHash   string      `json:"tree_hash"`
	TotalBytes int64       `json:"total_bytes"`
	Skipped    []string    `json:"skipped,omitempty"` // fifos, dispositivos etc. (não capturados)
	Git        GitInfo     `json:"git"`
}

// ValidPath aceita só caminhos relativos limpos, sem "..", sem bytes de controle.
func ValidPath(p string) bool {
	if p == "" || len(p) > 1024 || strings.HasPrefix(p, "/") || strings.ContainsAny(p, "\x00\n\r\\") {
		return false
	}
	c := path.Clean(p)
	return c == p && c != "." && c != ".." && !strings.HasPrefix(c, "../")
}

func sum(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

// TreeHash é o hash do conjunto: caminhos ordenados, tipo, modo (permissões, só de arquivo),
// tamanho, hash do conteúdo e alvo do link.
func TreeHash(files []FileEntry) string {
	es := append([]FileEntry{}, files...)
	sort.Slice(es, func(i, j int) bool { return es[i].Path < es[j].Path })
	h := sha256.New()
	for _, e := range es {
		mode := e.Mode & 0o777
		if e.Kind != KindFile {
			mode = 0 // permissões de diretório e de link não fazem parte da identidade
		}
		fmt.Fprintf(h, "%s\x00%s\x00%o\x00%d\x00%s\x00%s\n", e.Path, e.Kind, mode, e.Size, e.SHA256, e.Link)
	}
	return hex.EncodeToString(h.Sum(nil))
}

// canonicalJSON serializa com chaves ordenadas (maps do Go já saem ordenados).
func canonicalJSON(v any) []byte {
	b, _ := json.Marshal(v)
	var generic any
	if json.Unmarshal(b, &generic) != nil {
		return b
	}
	out, _ := json.Marshal(generic)
	return out
}

// Hash do manifesto e do estado.
func (m Manifest) Hash() string { return sum(canonicalJSON(m)) }

func stateHash(state json.RawMessage) string {
	if len(state) == 0 {
		state = json.RawMessage(`{}`)
	}
	var generic any
	if json.Unmarshal(state, &generic) != nil {
		return sum(state)
	}
	return sum(canonicalJSON(generic))
}

// Validate confere a coerência interna do manifesto (usado ao carregar e ao restaurar).
func (m Manifest) Validate() error {
	if m.Version != 1 || len(m.Files) > MaxFiles {
		return fmt.Errorf("%w: versão ou tamanho do manifesto", ErrCorrupt)
	}
	seen := map[string]bool{}
	var total int64
	for _, e := range m.Files {
		if !ValidPath(e.Path) || seen[e.Path] {
			return fmt.Errorf("%w: %q", ErrBadPath, e.Path)
		}
		seen[e.Path] = true
		switch e.Kind {
		case KindFile:
			if len(e.SHA256) != 64 || e.Size < 0 || e.Size > MaxFileBytes {
				return fmt.Errorf("%w: arquivo %q", ErrCorrupt, e.Path)
			}
			total += e.Size
		case KindSymlink:
			if e.Link == "" || len(e.Link) > 4096 || strings.ContainsAny(e.Link, "\x00") {
				return fmt.Errorf("%w: link %q", ErrCorrupt, e.Path)
			}
		case KindDir:
		default:
			return fmt.Errorf("%w: tipo de %q", ErrCorrupt, e.Path)
		}
	}
	if total != m.TotalBytes || m.TreeHash != TreeHash(m.Files) {
		return fmt.Errorf("%w: totais ou hash da árvore", ErrCorrupt)
	}
	return nil
}
