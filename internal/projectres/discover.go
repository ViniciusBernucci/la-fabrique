package projectres

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"io/fs"
	"regexp"
	"sort"
	"strings"
)

// MaxManifestBytes limita cada arquivo inspecionado.
const MaxManifestBytes = 256 << 10

// Files lê arquivos do repositório por caminho relativo fixo. Quem implementa
// não pode seguir links para fora do repositório; devolve fs.ErrNotExist se não houver.
type Files interface {
	Read(path string) ([]byte, error)
}

// MapFiles é um repositório em memória (manifestos enviados pelo chamador).
type MapFiles map[string]string

// Read implementa Files.
func (m MapFiles) Read(p string) ([]byte, error) {
	v, ok := m[p]
	if !ok {
		return nil, fs.ErrNotExist
	}
	return []byte(v), nil
}

// Suggestion é um valor sugerido com a origem.
type Suggestion struct {
	Name   string `json:"name"` // install | build | test | lint | start
	Value  string `json:"value"`
	Source string `json:"source"` // ex.: package.json#scripts.test
}

// Discovery é o resultado da inspeção.
type Discovery struct {
	Stack       []string     `json:"stack"`
	Commands    []Suggestion `json:"commands"`
	Services    []string     `json:"services"`
	SecretNames []string     `json:"secret_names"` // só nomes, de .env.example
	Inspected   []string     `json:"inspected"`
	Skipped     []string     `json:"skipped"`
}

// Arquivos inspecionados, em ordem fixa. Nada além desta lista é lido.
var manifests = []string{"package.json", "pnpm-lock.yaml", "yarn.lock", "package-lock.json", "go.mod", "Makefile", "pyproject.toml",
	"requirements.txt", "Cargo.toml", "pom.xml", "Gemfile", "docker-compose.yml", "docker-compose.yaml", "compose.yml", "compose.yaml", ".env.example"}

var (
	secretName = regexp.MustCompile(`^[A-Z][A-Z0-9_]{1,63}$`)
	makeTarget = regexp.MustCompile(`^([a-zA-Z][a-zA-Z0-9_-]*)\s*:`)
	svcName    = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]{0,62}$`)
)

// Inspect inspeciona os manifestos conhecidos. Conteúdo é só dado: nunca executado.
func Inspect(f Files) Discovery {
	d := Discovery{Stack: []string{}, Commands: []Suggestion{}, Services: []string{}, SecretNames: []string{}, Inspected: []string{}, Skipped: []string{}}
	got := map[string][]byte{}
	for _, name := range manifests {
		b, err := f.Read(name)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			d.Skipped = append(d.Skipped, name+": não foi possível ler")
			continue
		}
		if len(b) > MaxManifestBytes {
			d.Skipped = append(d.Skipped, name+": maior que o limite de inspeção")
			continue
		}
		got[name] = b
		d.Inspected = append(d.Inspected, name)
	}
	add := func(name, value, source string) {
		d.Commands = append(d.Commands, Suggestion{Name: name, Value: value, Source: source})
	}
	stack := map[string]bool{}

	if b, ok := got["package.json"]; ok {
		stack["node"] = true
		pm := "npm"
		switch {
		case got["pnpm-lock.yaml"] != nil:
			pm = "pnpm"
		case got["yarn.lock"] != nil:
			pm = "yarn"
		}
		var pkg struct {
			Scripts map[string]string `json:"scripts"`
		}
		if json.Unmarshal(b, &pkg) != nil {
			d.Skipped = append(d.Skipped, "package.json: JSON inválido")
		} else {
			add("install", pm+" install", "package.json")
			for _, s := range []string{"build", "test", "lint", "start"} {
				if _, ok := pkg.Scripts[s]; ok {
					cmd := pm + " run " + s
					if s == "test" && pm == "npm" || s == "start" && pm == "npm" {
						cmd = "npm " + s
					}
					add(s, cmd, "package.json#scripts."+s)
				}
			}
		}
	}
	if _, ok := got["go.mod"]; ok {
		stack["go"] = true
		add("build", "go build ./...", "go.mod")
		add("test", "go test ./...", "go.mod")
		add("lint", "go vet ./...", "go.mod")
	}
	if b, ok := got["Makefile"]; ok {
		stack["make"] = true
		targets := map[string]bool{}
		sc := bufio.NewScanner(bytes.NewReader(b))
		for sc.Scan() {
			if m := makeTarget.FindStringSubmatch(sc.Text()); m != nil {
				targets[m[1]] = true
			}
		}
		for _, t := range []string{"install", "build", "test", "lint"} {
			if targets[t] {
				add(t, "make "+t, "Makefile#"+t)
			}
		}
		if targets["up"] {
			add("start", "make up", "Makefile#up")
		}
	}
	if b, ok := got["pyproject.toml"]; ok {
		stack["python"] = true
		if bytes.Contains(b, []byte("pytest")) {
			add("test", "pytest", "pyproject.toml")
		}
	}
	if b, ok := got["requirements.txt"]; ok {
		stack["python"] = true
		add("install", "pip install -r requirements.txt", "requirements.txt")
		if bytes.Contains(b, []byte("pytest")) {
			add("test", "pytest", "requirements.txt")
		}
	}
	if _, ok := got["Cargo.toml"]; ok {
		stack["rust"] = true
		add("build", "cargo build", "Cargo.toml")
		add("test", "cargo test", "Cargo.toml")
	}
	if _, ok := got["pom.xml"]; ok {
		stack["java"] = true
		add("build", "mvn -B package", "pom.xml")
		add("test", "mvn -B test", "pom.xml")
	}
	if _, ok := got["Gemfile"]; ok {
		stack["ruby"] = true
		add("install", "bundle install", "Gemfile")
	}
	for _, name := range []string{"docker-compose.yml", "docker-compose.yaml", "compose.yml", "compose.yaml"} {
		b, ok := got[name]
		if !ok {
			continue
		}
		stack["docker"] = true
		d.Services = append(d.Services, composeServices(b)...)
		add("start", "docker compose up", name)
		break
	}
	if b, ok := got[".env.example"]; ok {
		sc := bufio.NewScanner(bytes.NewReader(b))
		for sc.Scan() {
			line := strings.TrimSpace(sc.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			if i := strings.IndexByte(line, '='); i > 0 {
				if n := strings.TrimSpace(strings.TrimPrefix(line[:i], "export ")); secretName.MatchString(n) {
					d.SecretNames = append(d.SecretNames, n) // só o nome; o valor nunca é guardado
				}
			}
		}
	}
	for s := range stack {
		d.Stack = append(d.Stack, s)
	}
	sort.Strings(d.Stack)
	sort.Strings(d.Services)
	sort.Strings(d.SecretNames)
	d.Services = dedupe(d.Services)
	d.SecretNames = dedupe(d.SecretNames)
	return d
}

func dedupe(in []string) []string {
	out := []string{}
	for i, s := range in {
		if i == 0 || s != in[i-1] {
			out = append(out, s)
		}
	}
	return out
}

// composeServices lê os nomes sob "services:" por indentação, sem interpretar YAML.
func composeServices(b []byte) []string {
	var out []string
	in, indent := false, -1
	sc := bufio.NewScanner(bytes.NewReader(b))
	for sc.Scan() {
		line := sc.Text()
		trim := strings.TrimSpace(line)
		if trim == "" || strings.HasPrefix(trim, "#") {
			continue
		}
		lead := len(line) - len(strings.TrimLeft(line, " "))
		if !in {
			if lead == 0 && strings.HasPrefix(trim, "services:") {
				in = true
			}
			continue
		}
		if lead == 0 {
			break
		}
		if indent < 0 {
			indent = lead
		}
		if lead == indent && strings.HasSuffix(trim, ":") {
			if n := strings.TrimSuffix(trim, ":"); svcName.MatchString(n) {
				out = append(out, n)
			}
		}
	}
	return out
}
