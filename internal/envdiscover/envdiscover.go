// Package envdiscover propõe a configuração do ambiente de teste/visual de um projeto lendo o
// repositório na revisão examinada (Decisão 29, AJ08-03). A descoberta é leitura estática: nenhum script
// é executado, valores de variáveis nunca entram na proposta (só o nome e uma referência `env:NOME`) e
// cada campo guarda a origem. O que não foi encontrado vira lacuna e o que admite mais de uma leitura
// vira ambiguidade, para o administrador resolver na edição.
package envdiscover

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"io/fs"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/ViniciusBernucci/le-fabrique/internal/projectres"
)

// Field é um valor proposto com a origem. Vazio com Source vazia = não encontrado.
type Field struct {
	Value  string `json:"value"`
	Source string `json:"source"`
}

// Service é um serviço do ambiente.
type Service struct {
	Name      string   `json:"name"`
	Kind      string   `json:"kind"` // frontend | api | database | cache | other
	Image     string   `json:"image,omitempty"`
	Build     string   `json:"build,omitempty"`
	Command   Field    `json:"command"`
	Ports     []int    `json:"ports"`
	Readiness Field    `json:"readiness"` // healthcheck declarado
	DependsOn []string `json:"depends_on"`
	Source    string   `json:"source"`
}

// CredentialRef é uma credencial por referência (nunca o valor).
type CredentialRef struct {
	Name   string `json:"name"`
	Ref    string `json:"ref"` // env:NOME
	Source string `json:"source"`
}

// Proposal é a proposta editável.
type Proposal struct {
	Services      []Service       `json:"services"`
	Install       Field           `json:"install"`
	Start         Field           `json:"start"`
	Build         Field           `json:"build"`
	Test          Field           `json:"test"`
	DataSetup     Field           `json:"data_setup"` // preparo de dados (migrate/seed)
	DataCleanup   Field           `json:"data_cleanup"`
	Credentials   []CredentialRef `json:"credentials"`
	BrowserTarget Field           `json:"browser_target"` // serviço e porta para o navegador MCP (na rede interna)
	Gaps          []string        `json:"gaps"`
	Ambiguities   []string        `json:"ambiguities"`
	Inspected     []string        `json:"inspected"`
}

var composeFiles = []string{"docker-compose.yml", "docker-compose.yaml", "compose.yml", "compose.yaml"}

var (
	keyLine   = regexp.MustCompile(`^(\s*)([A-Za-z_][A-Za-z0-9_-]*):\s*(.*)$`)
	portInt   = regexp.MustCompile(`(\d{2,5})`)
	envName   = regexp.MustCompile(`^[A-Z][A-Z0-9_]{1,63}$`)
	placehold = regexp.MustCompile(`(?i)(changeme|change-me|your[_-]|<.*>|example|xxx+|todo)`)
)

type block struct {
	name  string
	lines []string
}

func serviceBlocks(b []byte) []block {
	var out []block
	in, indent := false, -1
	var cur *block
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
			out = append(out, block{name: strings.TrimSuffix(trim, ":")})
			cur = &out[len(out)-1]
			continue
		}
		if cur != nil && lead > indent {
			cur.lines = append(cur.lines, line)
		}
	}
	return out
}

func classify(name, image string, ports []int) string {
	n := strings.ToLower(name + " " + image)
	switch {
	case strings.Contains(n, "postgres") || strings.Contains(n, "mysql") || strings.Contains(n, "mariadb") || strings.Contains(n, "mongo"):
		return "database"
	case strings.Contains(n, "redis") || strings.Contains(n, "memcache"):
		return "cache"
	case strings.Contains(n, "web") || strings.Contains(n, "front") || strings.Contains(n, "ui") || strings.Contains(n, "vite") || strings.Contains(n, "next"):
		return "frontend"
	case strings.Contains(n, "api") || strings.Contains(n, "backend") || strings.Contains(n, "server") || strings.Contains(n, "worker"):
		return "api"
	}
	for _, p := range ports {
		if p == 3000 || p == 5173 || p == 4200 {
			return "frontend"
		}
	}
	return "other"
}

func parseService(b block, src string) Service {
	s := Service{Name: b.name, Ports: []int{}, DependsOn: []string{}, Source: src + "#services." + b.name}
	section := ""
	for _, l := range b.lines {
		m := keyLine.FindStringSubmatch(l)
		trim := strings.TrimSpace(l)
		if m != nil && !strings.HasPrefix(trim, "- ") {
			key, val := m[2], strings.TrimSpace(m[3])
			lead := len(m[1])
			first := lead <= firstIndent(b.lines)
			if first {
				section = key
				switch key {
				case "image":
					s.Image = strings.Trim(val, `"'`)
				case "build":
					s.Build = strings.Trim(val, `"'`)
				case "command":
					s.Command = Field{Value: strings.Trim(val, `"'`), Source: src + "#services." + b.name + ".command"}
				case "healthcheck":
					s.Readiness = Field{Value: "healthcheck declarado", Source: src + "#services." + b.name + ".healthcheck"}
				}
				continue
			}
			if section == "depends_on" {
				s.DependsOn = append(s.DependsOn, key)
				continue
			}
			if section == "build" && key == "context" {
				s.Build = strings.Trim(val, `"'`)
			}
			if section == "healthcheck" && key == "test" && s.Readiness.Value == "healthcheck declarado" {
				s.Readiness.Value = val
			}
			continue
		}
		if strings.HasPrefix(trim, "- ") {
			item := strings.Trim(strings.TrimPrefix(trim, "- "), `"'`)
			switch section {
			case "ports":
				// "host:container" ou só a porta: a que o navegador usa na rede interna é a do container.
				parts := strings.Split(item, ":")
				if m := portInt.FindString(parts[len(parts)-1]); m != "" {
					if n, err := strconv.Atoi(m); err == nil {
						s.Ports = append(s.Ports, n)
					}
				}
			case "depends_on":
				s.DependsOn = append(s.DependsOn, strings.TrimSuffix(item, ":"))
			}
		}
	}
	sort.Ints(s.Ports)
	sort.Strings(s.DependsOn)
	s.Kind = classify(s.Name, s.Image, s.Ports)
	return s
}

func firstIndent(lines []string) int {
	min := 1 << 30
	for _, l := range lines {
		if strings.TrimSpace(l) == "" {
			continue
		}
		if lead := len(l) - len(strings.TrimLeft(l, " ")); lead < min {
			min = lead
		}
	}
	return min
}

// Discover lê o repositório e monta a proposta. Nada é executado e nenhum valor de variável é copiado.
func Discover(f projectres.Files) Proposal {
	p := Proposal{Services: []Service{}, Credentials: []CredentialRef{}, Gaps: []string{}, Ambiguities: []string{}, Inspected: []string{}}
	read := func(name string) ([]byte, bool) {
		b, err := f.Read(name)
		if errors.Is(err, fs.ErrNotExist) || err != nil || len(b) > projectres.MaxManifestBytes {
			return nil, false
		}
		p.Inspected = append(p.Inspected, name)
		return b, true
	}
	base := projectres.Inspect(f)
	cmds := map[string][]projectres.Suggestion{}
	for _, c := range base.Commands {
		cmds[c.Name] = append(cmds[c.Name], c)
	}
	set := func(dst *Field, name string) {
		cs := cmds[name]
		if len(cs) == 0 {
			return
		}
		*dst = Field{Value: cs[0].Value, Source: cs[0].Source}
		var vs []string
		seen := map[string]bool{}
		for _, c := range cs {
			if !seen[c.Value] {
				seen[c.Value] = true
				vs = append(vs, c.Value+" ("+c.Source+")")
			}
		}
		if len(vs) > 1 {
			p.Ambiguities = append(p.Ambiguities, "mais de um comando para "+name+": "+strings.Join(vs, "; "))
		}
	}
	set(&p.Install, "install")
	set(&p.Build, "build")
	set(&p.Test, "test")
	set(&p.Start, "start")

	// package.json: script de desenvolvimento do front quando houver.
	if b, ok := read("package.json"); ok {
		var pkg struct {
			Scripts map[string]string `json:"scripts"`
		}
		if json.Unmarshal(b, &pkg) == nil {
			if _, has := pkg.Scripts["dev"]; has && p.Start.Value == "" {
				p.Start = Field{Value: "npm run dev", Source: "package.json#scripts.dev"}
			} else if has {
				p.Ambiguities = append(p.Ambiguities, "dois jeitos de subir a aplicação: "+p.Start.Value+" ("+p.Start.Source+") e npm run dev (package.json#scripts.dev)")
			}
		}
	}
	// Makefile: preparo e limpeza de dados.
	if b, ok := read("Makefile"); ok {
		targets := map[string]bool{}
		sc := bufio.NewScanner(bytes.NewReader(b))
		for sc.Scan() {
			if m := regexp.MustCompile(`^([a-zA-Z][a-zA-Z0-9_-]*)\s*:`).FindStringSubmatch(sc.Text()); m != nil {
				targets[m[1]] = true
			}
		}
		for _, t := range []string{"migrate", "seed"} {
			if targets[t] {
				if p.DataSetup.Value == "" {
					p.DataSetup = Field{Value: "make " + t, Source: "Makefile#" + t}
				} else {
					p.DataSetup.Value += " && make " + t
				}
			}
		}
		for _, t := range []string{"clean", "down", "reset"} {
			if targets[t] && p.DataCleanup.Value == "" {
				p.DataCleanup = Field{Value: "make " + t, Source: "Makefile#" + t}
			}
		}
	}
	// Composição.
	var found []string
	for _, name := range composeFiles {
		if b, ok := read(name); ok {
			found = append(found, name)
			for _, blk := range serviceBlocks(b) {
				p.Services = append(p.Services, parseService(blk, name))
			}
		}
	}
	if len(found) > 1 {
		p.Ambiguities = append(p.Ambiguities, "mais de um arquivo de composição: "+strings.Join(found, ", "))
	}
	if len(found) == 0 {
		p.Gaps = append(p.Gaps, "nenhum arquivo de composição: os serviços precisam ser descritos manualmente")
	}
	// Credenciais por referência (nunca o valor).
	if b, ok := read(".env.example"); ok {
		sc := bufio.NewScanner(bytes.NewReader(b))
		for sc.Scan() {
			line := strings.TrimSpace(sc.Text())
			line = strings.TrimPrefix(line, "# ")
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			if i := strings.IndexByte(line, '='); i > 0 {
				n := strings.TrimSpace(strings.TrimPrefix(line[:i], "export "))
				if !envName.MatchString(n) {
					continue
				}
				val := strings.TrimSpace(line[i+1:])
				if val != "" && !placehold.MatchString(val) && isSecretish(n) {
					p.Gaps = append(p.Gaps, "o valor de "+n+" no exemplo parece real: use uma credencial de teste de escopo mínimo")
				}
				p.Credentials = append(p.Credentials, CredentialRef{Name: n, Ref: "env:" + n, Source: ".env.example"})
			}
		}
	}
	// Alvo do navegador: um serviço frontend com porta.
	var fronts []Service
	for _, s := range p.Services {
		if s.Kind == "frontend" {
			fronts = append(fronts, s)
		}
	}
	switch {
	case len(fronts) == 1 && len(fronts[0].Ports) > 0:
		p.BrowserTarget = Field{Value: "http://" + fronts[0].Name + ":" + strconv.Itoa(fronts[0].Ports[0]), Source: fronts[0].Source + ".ports"}
	case len(fronts) > 1:
		p.Ambiguities = append(p.Ambiguities, "mais de um serviço de frontend ("+names(fronts)+"): escolha qual o navegador acessa")
	case len(fronts) == 1:
		p.Gaps = append(p.Gaps, "o serviço de frontend "+fronts[0].Name+" não declara porta")
	default:
		p.Gaps = append(p.Gaps, "nenhum serviço de frontend identificado para o navegador")
	}
	// Lacunas.
	if p.Test.Value == "" {
		p.Gaps = append(p.Gaps, "comando de testes não encontrado")
	}
	if p.Start.Value == "" {
		p.Gaps = append(p.Gaps, "comando para subir a aplicação não encontrado")
	}
	for _, s := range p.Services {
		if s.Readiness.Value == "" && (s.Kind == "frontend" || s.Kind == "api" || s.Kind == "database") {
			p.Gaps = append(p.Gaps, "o serviço "+s.Name+" não declara prontidão (healthcheck)")
		}
	}
	if p.DataSetup.Value == "" {
		p.Gaps = append(p.Gaps, "preparo de dados (migrate/seed) não identificado")
	}
	sort.Strings(p.Gaps)
	sort.Strings(p.Ambiguities)
	return p
}

func names(ss []Service) string {
	var n []string
	for _, s := range ss {
		n = append(n, s.Name)
	}
	return strings.Join(n, ", ")
}

func isSecretish(n string) bool {
	n = strings.ToLower(n)
	for _, k := range []string{"secret", "password", "token", "key"} {
		if strings.Contains(n, k) {
			return true
		}
	}
	return false
}
