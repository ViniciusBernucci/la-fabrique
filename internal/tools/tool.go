package tools

import (
	"fmt"
	"path"
	"regexp"
	"strings"
)

// Tipos de campo.
const (
	TypeString = "string"
	TypeInt    = "int"
	TypeBool   = "bool"
	TypePath   = "path" // caminho confinado ao workspace
	TypeEnum   = "enum"
	TypeList   = "list" // lista de textos (só na saída)
)

// Tipos de ferramenta.
const (
	KindNative      = "native"
	KindScript      = "script"
	KindIntegration = "integration"
)

// Permissões (escopos) que uma ferramenta pode exigir.
const (
	PermRead  = "read"
	PermWrite = "write"
	PermExec  = "exec"
	PermGit   = "git"
	PermAPI   = "api"
)

var knownPerms = map[string]bool{PermRead: true, PermWrite: true, PermExec: true, PermGit: true, PermAPI: true}

// Field descreve um parâmetro ou um campo de saída.
type Field struct {
	Name        string   `json:"name"`
	Type        string   `json:"type"`
	Required    bool     `json:"required"`
	Description string   `json:"description,omitempty"`
	Max         int      `json:"max,omitempty"` // texto: tamanho máximo; inteiro: valor máximo
	Min         int      `json:"min,omitempty"` // inteiro: valor mínimo
	Options     []string `json:"options,omitempty"`
	Pattern     string   `json:"pattern,omitempty"`
	re          *regexp.Regexp
}

// Limits são os limites de uma chamada.
type Limits struct {
	TimeoutSeconds int `json:"timeout_seconds"`
	MaxOutputBytes int `json:"max_output_bytes"`
}

// Tool é uma ferramenta do catálogo.
type Tool struct {
	Name        string   `json:"name"`
	Kind        string   `json:"kind"`
	Description string   `json:"description"`
	Input       []Field  `json:"input"`
	Output      []Field  `json:"output"`
	Permissions []string `json:"permissions"`
	Limits      Limits   `json:"limits"`
	Command     string   `json:"command,omitempty"`   // script: roda com os parâmetros em PARAM_<nome>
	Operation   string   `json:"operation,omitempty"` // integration: operação do conector
	Requires    []string `json:"requires,omitempty"`  // binários exigidos na imagem (ex.: git)
	Version     int      `json:"version,omitempty"`
}

// ArgError é um argumento inválido com o campo e o motivo.
type ArgError struct{ Field, Message string }

func (e *ArgError) Error() string { return e.Field + ": " + e.Message }

var (
	namePattern  = regexp.MustCompile(`^[a-z][a-z0-9_]{1,47}$`)
	fieldPattern = regexp.MustCompile(`^[a-z][a-z0-9_]{0,31}$`)
)

// Compile valida a definição da ferramenta e prepara os padrões.
func (t *Tool) Compile() error {
	if !namePattern.MatchString(t.Name) {
		return &ArgError{"name", "O nome deve usar minúsculas, números e sublinhado (2 a 48)."}
	}
	switch t.Kind {
	case KindNative, KindScript, KindIntegration:
	default:
		return &ArgError{"kind", "O tipo deve ser native, script ou integration."}
	}
	if strings.TrimSpace(t.Description) == "" || len(t.Description) > 500 {
		return &ArgError{"description", "A descrição deve ter entre 1 e 500 caracteres."}
	}
	for _, p := range t.Permissions {
		if !knownPerms[p] {
			return &ArgError{"permissions", fmt.Sprintf("Permissão desconhecida: %q.", p)}
		}
	}
	if len(t.Permissions) == 0 {
		return &ArgError{"permissions", "Declare ao menos uma permissão."}
	}
	if t.Limits.TimeoutSeconds < 1 || t.Limits.TimeoutSeconds > 3600 || t.Limits.MaxOutputBytes < 1 || t.Limits.MaxOutputBytes > 4<<20 {
		return &ArgError{"limits", "Limites inválidos (tempo 1–3600 s; saída 1 B–4 MiB)."}
	}
	if t.Kind == KindScript && (strings.TrimSpace(t.Command) == "" || len(t.Command) > 2000) {
		return &ArgError{"command", "Uma ferramenta de script precisa de um comando."}
	}
	if t.Kind == KindIntegration && !fieldPattern.MatchString(t.Operation) {
		return &ArgError{"operation", "Uma ferramenta de integração precisa de uma operação válida."}
	}
	if err := compileFields("input", t.Input); err != nil {
		return err
	}
	return compileFields("output", t.Output)
}

func compileFields(where string, fs []Field) error {
	seen := map[string]bool{}
	for i := range fs {
		f := &fs[i]
		if !fieldPattern.MatchString(f.Name) || seen[f.Name] {
			return &ArgError{where, fmt.Sprintf("Campo %q inválido ou repetido.", f.Name)}
		}
		seen[f.Name] = true
		switch f.Type {
		case TypeString, TypeInt, TypeBool, TypePath:
		case TypeEnum:
			if len(f.Options) == 0 {
				return &ArgError{where, fmt.Sprintf("O campo %q precisa de opções.", f.Name)}
			}
		case TypeList:
			if where == "input" {
				return &ArgError{where, "Listas só existem na saída."}
			}
		default:
			return &ArgError{where, fmt.Sprintf("Tipo desconhecido no campo %q.", f.Name)}
		}
		if f.Pattern != "" {
			re, err := regexp.Compile(f.Pattern)
			if err != nil {
				return &ArgError{where, fmt.Sprintf("Padrão inválido no campo %q.", f.Name)}
			}
			f.re = re
		}
	}
	return nil
}

const workspace = "/workspace"

// SafePath confina um caminho ao workspace (lexicalmente): relativos são
// resolvidos a partir de /workspace; ".." que escape, NUL e caminhos fora são
// recusados. Links simbólicos dentro do sandbox são tratados pelo T11-02.
func SafePath(p string) (string, error) {
	p = strings.TrimSpace(p)
	if p == "" || len(p) > 300 || strings.ContainsRune(p, 0) || strings.Contains(p, `\`) {
		return "", fmt.Errorf("caminho inválido")
	}
	if !strings.HasPrefix(p, "/") {
		p = workspace + "/" + p
	}
	c := path.Clean(p)
	if c != workspace && !strings.HasPrefix(c, workspace+"/") {
		return "", fmt.Errorf("o caminho sai do workspace")
	}
	// ".." resolvido pelo Clean já não escapa; recusa também o que só funcionaria por ".." explícito
	// que mudaria de diretório de forma ambígua.
	if strings.Contains(p, "/../") || strings.HasSuffix(p, "/..") {
		return "", fmt.Errorf("travessia de diretório não é permitida")
	}
	return c, nil
}

// Validate confere os argumentos contra o esquema e devolve a cópia normalizada.
// Argumento desconhecido, ausente, de tipo errado ou fora dos limites falha ANTES
// de qualquer efeito.
func (t Tool) Validate(args map[string]any) (map[string]any, error) {
	out := map[string]any{}
	known := map[string]bool{}
	for _, f := range t.Input {
		known[f.Name] = true
	}
	for k := range args {
		if !known[k] {
			return nil, &ArgError{k, "Parâmetro desconhecido para esta ferramenta."}
		}
	}
	for _, f := range t.Input {
		v, present := args[f.Name]
		if !present || v == nil {
			if f.Required {
				return nil, &ArgError{f.Name, "Parâmetro obrigatório."}
			}
			continue
		}
		n, err := f.normalize(v)
		if err != nil {
			return nil, &ArgError{f.Name, err.Error()}
		}
		out[f.Name] = n
	}
	return out, nil
}

func (f Field) normalize(v any) (any, error) {
	switch f.Type {
	case TypeString, TypePath, TypeEnum:
		s, ok := v.(string)
		if !ok {
			return nil, fmt.Errorf("deve ser texto")
		}
		max := f.Max
		if max == 0 {
			max = 100000
		}
		if len(s) > max {
			return nil, fmt.Errorf("texto acima do limite de %d bytes", max)
		}
		if f.Type == TypePath {
			c, err := SafePath(s)
			if err != nil {
				return nil, err
			}
			return c, nil
		}
		if f.Type == TypeEnum {
			for _, o := range f.Options {
				if o == s {
					return s, nil
				}
			}
			return nil, fmt.Errorf("deve ser um de: %s", strings.Join(f.Options, ", "))
		}
		if f.re != nil && !f.re.MatchString(s) {
			return nil, fmt.Errorf("formato inválido")
		}
		return s, nil
	case TypeInt:
		n, ok := v.(float64)
		if i, isInt := v.(int); isInt {
			n, ok = float64(i), true
		}
		if !ok || n != float64(int64(n)) {
			return nil, fmt.Errorf("deve ser um inteiro")
		}
		if int(n) < f.Min || (f.Max != 0 && int(n) > f.Max) {
			return nil, fmt.Errorf("fora da faixa permitida (%d a %d)", f.Min, f.Max)
		}
		return int(n), nil
	case TypeBool:
		b, ok := v.(bool)
		if !ok {
			return nil, fmt.Errorf("deve ser verdadeiro ou falso")
		}
		return b, nil
	}
	return nil, fmt.Errorf("tipo desconhecido")
}

// TypedOutput confere a saída contra o esquema: campos ausentes ou de tipo
// errado são erro; campos fora do esquema são descartados.
func (t Tool) TypedOutput(raw map[string]any) (map[string]any, error) {
	out := map[string]any{}
	for _, f := range t.Output {
		v, ok := raw[f.Name]
		if !ok {
			if f.Required {
				return nil, fmt.Errorf("a saída não tem o campo %q", f.Name)
			}
			continue
		}
		switch f.Type {
		case TypeString, TypePath, TypeEnum:
			if _, ok := v.(string); !ok {
				return nil, fmt.Errorf("o campo %q da saída deve ser texto", f.Name)
			}
		case TypeInt:
			switch x := v.(type) {
			case int:
			case int64:
				v = int(x)
			case float64:
				v = int(x)
			default:
				return nil, fmt.Errorf("o campo %q da saída deve ser inteiro", f.Name)
			}
		case TypeBool:
			if _, ok := v.(bool); !ok {
				return nil, fmt.Errorf("o campo %q da saída deve ser booleano", f.Name)
			}
		case TypeList:
			if _, ok := v.([]string); !ok {
				return nil, fmt.Errorf("o campo %q da saída deve ser uma lista de textos", f.Name)
			}
		}
		out[f.Name] = v
	}
	return out, nil
}
