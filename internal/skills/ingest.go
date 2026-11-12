package skills

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"strings"

	"github.com/ViniciusBernucci/le-fabrique/internal/policy"
)

// Package é uma skill validada, pronta para ser guardada. Nada foi executado.
type Package struct {
	Manifest      Manifest
	Body          string
	Files         []File
	NeedsApproval bool   // algum comando é sensível: exigirá aprovação ao rodar
	Hash          string // SHA-256 do conteúdo (arquivos, manifesto e instruções)
}

func hashOf(m Manifest, body string, files []File) string {
	h := sha256.New()
	sorted := append([]File{}, files...)
	sortFiles(sorted)
	for _, f := range sorted {
		sum := sha256.Sum256(f.Content)
		fmt.Fprintf(h, "%s\x00%x\n", f.Path, sum)
	}
	raw, _ := json.Marshal(m)
	h.Write(raw)
	h.Write([]byte{0})
	h.Write([]byte(body))
	return hex.EncodeToString(h.Sum(nil))
}

func find(files []File, name string) (File, bool) {
	for _, f := range files {
		if f.Path == name {
			return f, true
		}
	}
	return File{}, false
}

func commentOrBlank(line string) bool {
	line = strings.TrimSpace(line)
	return line == "" || strings.HasPrefix(line, "#")
}

// Ingest valida o pacote: manifesto, coerência com o executor, recursos
// existentes e scripts contra a política. Não executa nada.
func Ingest(files []File, pol *policy.Policy) (Package, error) {
	var pkg Package
	skillMD, hasMD := find(files, "SKILL.md")
	manifestFile, hasJSON := find(files, "skill.json")

	var m Manifest
	body := ""
	switch {
	case hasJSON:
		var err error
		if m, err = ParseManifestJSON(manifestFile.Content); err != nil {
			return pkg, asImport(err, "skill.json")
		}
		if hasMD {
			_, b, err := ParseFrontmatter(string(skillMD.Content))
			if err != nil {
				return pkg, asImport(err, "SKILL.md")
			}
			body = b
		}
	case hasMD:
		fm, b, err := ParseFrontmatter(string(skillMD.Content))
		if err != nil {
			return pkg, asImport(err, "SKILL.md")
		}
		body = b
		m = Manifest{Name: fm["name"], Description: fm["description"], Executor: fm["executor"]}
		if m.Executor == "" {
			m.Executor = ExecAI
		}
		for _, p := range strings.Split(fm["permissions"], ",") {
			if p = strings.TrimSpace(p); p != "" {
				m.Permissions = append(m.Permissions, p)
			}
		}
	default:
		return pkg, imp("no_manifest", "", "O pacote precisa de um SKILL.md ou de um skill.json na raiz.")
	}
	if err := m.Validate(body); err != nil {
		return pkg, asImport(err, "")
	}
	for _, r := range m.Resources {
		clean, err := CleanPath(r)
		if err != nil {
			return pkg, imp("bad_path", clipPath(r), "Recurso com caminho inválido no manifesto.")
		}
		if _, ok := find(files, clean); !ok {
			return pkg, imp("resource_missing", clean, "O manifesto cita um recurso que não está no pacote.")
		}
	}

	// Comandos declarados: tudo o que a política negar (inclusive por não estar na
	// lista) recusa a importação; sensível é aceito e marcado.
	var declared []string
	declared = append(declared, m.Commands...)
	for _, s := range m.Steps {
		if s.Type == StepScript {
			declared = append(declared, s.Command)
		}
	}
	for _, c := range declared {
		d := pol.Evaluate(policy.Action{Tool: policy.ToolScript, Operation: "run", Params: map[string]any{"command": c}})
		switch d.Effect {
		case policy.Deny:
			return pkg, imp("script_forbidden", "skill.json", fmt.Sprintf("O comando %q está fora da política: %s", clipPath(c), d.Reason))
		case policy.Approve:
			pkg.NeedsApproval = true
		}
	}
	// Arquivos de script do pacote: só o comprovadamente proibido é recusado.
	for _, f := range files {
		if !strings.HasPrefix(f.Path, "scripts/") || path.Ext(f.Path) == ".md" {
			continue
		}
		for _, line := range strings.Split(string(f.Content), "\n") {
			if commentOrBlank(line) {
				continue
			}
			if d, bad := pol.ExplicitDeny(policy.Action{Tool: policy.ToolScript, Operation: "run", Params: map[string]any{"command": line}}); bad {
				return pkg, imp("script_forbidden", f.Path, fmt.Sprintf("O script contém um comando proibido: %s", d.Reason))
			}
		}
	}

	pkg.Manifest, pkg.Body, pkg.Files = m, body, files
	pkg.Hash = hashOf(m, body, files)
	return pkg, nil
}

func asImport(err error, path string) error {
	var ve *ValidationError
	if errors.As(err, &ve) {
		return imp("invalid_manifest", path, ve.Field+": "+ve.Message)
	}
	return err
}

// FromManual monta um Package a partir de um cadastro manual (sem arquivos).
func FromManual(m Manifest, body string, pol *policy.Policy) (Package, error) {
	files := []File{}
	if strings.TrimSpace(body) != "" {
		files = append(files, File{Path: "SKILL.md", Content: []byte(body)})
	}
	raw, _ := json.Marshal(m)
	files = append(files, File{Path: "skill.json", Content: raw})
	return Ingest(files, pol)
}
