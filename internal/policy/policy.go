package policy

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"regexp"
	"sort"
	"strings"
)

// Efeitos da decisão, do mais permissivo ao mais restritivo.
type Effect string

const (
	Allow   Effect = "allow"   // baixo risco: segue automaticamente
	Approve Effect = "approve" // sensível: exige aprovação humana
	Deny    Effect = "deny"    // proibido: nunca executa
)

func severity(e Effect) int {
	switch e {
	case Allow:
		return 1
	case Approve:
		return 2
	}
	return 3
}

// Risk é a classificação associada.
type Risk string

const (
	RiskLow       Risk = "low"
	RiskSensitive Risk = "sensitive"
	RiskForbidden Risk = "forbidden"
)

// Ferramentas conhecidas. Qualquer outra é negada.
const (
	ToolScript    = "script"
	ToolCLI       = "cli"
	ToolAPI       = "api"
	ToolMCP       = "mcp"
	ToolConnector = "connector"
	ToolGit       = "git"
	ToolFS        = "fs" // leitura e escrita no workspace do sandbox (ferramentas nativas)
)

// Action é o que se quer executar, com os parâmetros reais.
type Action struct {
	Project   string         `json:"project,omitempty"`
	Tool      string         `json:"tool"`
	Operation string         `json:"operation"`
	Resource  string         `json:"resource,omitempty"`
	Target    string         `json:"target,omitempty"` // URL ou host de destino
	Params    map[string]any `json:"params,omitempty"`
}

// ParamMatch condiciona uma regra a um parâmetro.
type ParamMatch struct {
	Key    string   `json:"key"`
	Equals any      `json:"equals,omitempty"`
	OneOf  []string `json:"one_of,omitempty"`
	Regex  string   `json:"regex,omitempty"`
	Absent bool     `json:"absent,omitempty"`
	re     *regexp.Regexp
}

// Rule é uma regra declarativa. Campos vazios casam com qualquer valor; globs
// usam "*" para qualquer sequência.
type Rule struct {
	ID        string       `json:"id"`
	Effect    Effect       `json:"effect"`
	Reason    string       `json:"reason"`
	Tool      string       `json:"tool,omitempty"`
	Operation string       `json:"operation,omitempty"`
	Resource  string       `json:"resource,omitempty"`
	Target    string       `json:"target,omitempty"` // glob de host
	Params    []ParamMatch `json:"params,omitempty"`
}

// Policy reúne as regras. O efeito final é o MAIS RESTRITIVO entre as regras que
// casam (deny > approve > allow), independente da ordem; sem nenhuma regra
// vale Default (padrão: negar).
type Policy struct {
	Rules   []Rule `json:"rules"`
	Default Effect `json:"default"`
}

// Decision é o resultado da avaliação.
type Decision struct {
	Effect     Effect `json:"effect"`
	Risk       Risk   `json:"risk"`
	Rule       string `json:"rule"`
	Reason     string `json:"reason"`
	ParamsHash string `json:"params_hash"` // SHA-256 dos parâmetros canônicos
}

func riskOf(e Effect) Risk {
	switch e {
	case Allow:
		return RiskLow
	case Approve:
		return RiskSensitive
	}
	return RiskForbidden
}

// Compile valida as regras (efeitos e expressões) e prepara os casadores.
func (p *Policy) Compile() error {
	if p.Default == "" {
		p.Default = Deny
	}
	if p.Default != Allow && p.Default != Approve && p.Default != Deny {
		return fmt.Errorf("efeito padrão inválido %q", p.Default)
	}
	seen := map[string]bool{}
	for i := range p.Rules {
		r := &p.Rules[i]
		if r.ID == "" || seen[r.ID] {
			return fmt.Errorf("regra %d: ID vazio ou repetido", i)
		}
		seen[r.ID] = true
		if r.Effect != Allow && r.Effect != Approve && r.Effect != Deny {
			return fmt.Errorf("regra %s: efeito inválido", r.ID)
		}
		for j := range r.Params {
			if r.Params[j].Key == "" {
				return fmt.Errorf("regra %s: parâmetro sem chave", r.ID)
			}
			if r.Params[j].Regex != "" {
				re, err := regexp.Compile(r.Params[j].Regex)
				if err != nil {
					return fmt.Errorf("regra %s: expressão inválida: %v", r.ID, err)
				}
				r.Params[j].re = re
			}
		}
	}
	return nil
}

func glob(pattern, s string) bool {
	if pattern == "" {
		return true
	}
	parts := strings.Split(pattern, "*")
	if len(parts) == 1 {
		return pattern == s
	}
	if !strings.HasPrefix(s, parts[0]) {
		return false
	}
	s = s[len(parts[0]):]
	for _, mid := range parts[1 : len(parts)-1] {
		i := strings.Index(s, mid)
		if i < 0 {
			return false
		}
		s = s[i+len(mid):]
	}
	return strings.HasSuffix(s, parts[len(parts)-1])
}

func (pm ParamMatch) matches(params map[string]any) bool {
	v, present := params[pm.Key]
	if pm.Absent {
		return !present
	}
	if !present {
		return false
	}
	text := fmt.Sprint(v)
	switch {
	case pm.Equals != nil:
		return fmt.Sprint(pm.Equals) == text
	case len(pm.OneOf) > 0:
		for _, o := range pm.OneOf {
			if o == text {
				return true
			}
		}
		return false
	case pm.re != nil:
		return pm.re.MatchString(text)
	}
	return true
}

func (r Rule) matches(a Action, host string) bool {
	if r.Tool != "" && r.Tool != a.Tool {
		return false
	}
	if !glob(r.Operation, a.Operation) || !glob(r.Resource, a.Resource) || !glob(r.Target, host) {
		return false
	}
	for _, pm := range r.Params {
		if !pm.matches(a.Params) {
			return false
		}
	}
	return true
}

// HostOf extrai o host (minúsculo, sem porta) de uma URL ou host:porta.
func HostOf(target string) string {
	target = strings.TrimSpace(target)
	if target == "" {
		return ""
	}
	if !strings.Contains(target, "://") {
		target = "//" + target
	}
	u, err := url.Parse(target)
	if err != nil {
		return strings.ToLower(target)
	}
	return strings.ToLower(u.Hostname())
}

// privateHost informa se o destino é interno: loopback, rede privada, link-local,
// metadados de nuvem ou nomes internos. Só olha o texto do host: a imposição na
// camada de rede (resolução de DNS e redirecionamentos) é de quem faz a chamada.
func privateHost(host string) bool {
	if host == "" {
		return false
	}
	switch {
	case host == "localhost", strings.HasSuffix(host, ".localhost"), strings.HasSuffix(host, ".local"),
		strings.HasSuffix(host, ".internal"), strings.HasSuffix(host, ".lan"), host == "metadata", host == "host.docker.internal":
		return true
	}
	if ip := net.ParseIP(strings.Trim(host, "[]")); ip != nil {
		return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() ||
			ip.IsUnspecified() || ip.IsMulticast() || ip.Equal(net.ParseIP("169.254.169.254"))
	}
	// Formas numéricas alternativas (decimal, octal, hexadecimal) de endereços IP.
	if regexp.MustCompile(`^(0x[0-9a-f]+|[0-9]+)$`).MatchString(host) {
		return true
	}
	return false
}

// Canonical devolve os parâmetros em JSON canônico (chaves ordenadas) e o hash.
func Canonical(params map[string]any) ([]byte, string) {
	raw := canon(params)
	sum := sha256.Sum256(raw)
	return raw, hex.EncodeToString(sum[:])
}

func canon(v any) []byte {
	switch x := v.(type) {
	case nil:
		return []byte("null")
	case map[string]any:
		keys := make([]string, 0, len(x))
		for k := range x {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		var b strings.Builder
		b.WriteByte('{')
		for i, k := range keys {
			if i > 0 {
				b.WriteByte(',')
			}
			kb, _ := json.Marshal(k)
			b.Write(kb)
			b.WriteByte(':')
			b.Write(canon(x[k]))
		}
		b.WriteByte('}')
		return []byte(b.String())
	case []any:
		var b strings.Builder
		b.WriteByte('[')
		for i, e := range x {
			if i > 0 {
				b.WriteByte(',')
			}
			b.Write(canon(e))
		}
		b.WriteByte(']')
		return []byte(b.String())
	case []string:
		out := make([]any, len(x))
		for i, e := range x {
			out[i] = e
		}
		return canon(out)
	case int:
		return canon(float64(x))
	case int64:
		return canon(float64(x))
	}
	b, _ := json.Marshal(v)
	return b
}

// Evaluate decide sobre a ação. Proibições embutidas (destino interno e
// operações destrutivas óbvias) valem mesmo que uma regra permita.
func (p *Policy) Evaluate(a Action) Decision {
	_, hash := Canonical(a.Params)
	d := p.decide(a)
	d.ParamsHash = hash
	d.Risk = riskOf(d.Effect)
	return d
}

func (p *Policy) decide(a Action) Decision {
	deny := func(rule, why string) Decision { return Decision{Effect: Deny, Rule: rule, Reason: why} }
	switch a.Tool {
	case ToolScript, ToolCLI, ToolAPI, ToolMCP, ToolConnector, ToolGit, ToolFS:
	default:
		return deny("builtin.unknown_tool", "Ferramenta desconhecida: nada fora da lista é executado.")
	}
	if strings.TrimSpace(a.Operation) == "" {
		return deny("builtin.no_operation", "Ação sem operação.")
	}
	host := HostOf(a.Target)
	if host == "" {
		if u, ok := a.Params["url"].(string); ok {
			host = HostOf(u)
		}
	}
	if privateHost(host) {
		return deny("builtin.internal_target", "Destino interno, de loopback ou de metadados: nunca permitido.")
	}
	if a.Tool == ToolScript {
		return p.decideScript(a)
	}
	return p.combine(a, host)
}

func (p *Policy) combine(a Action, host string) Decision {
	best := Decision{Effect: p.Default, Rule: "default", Reason: "Fora da lista permitida: negado por padrão."}
	if p.Default == Allow {
		best.Reason = "Sem regra: permitido por padrão."
	}
	matched := false
	for _, r := range p.Rules {
		if !r.matches(a, host) {
			continue
		}
		if !matched || severity(r.Effect) > severity(best.Effect) {
			best = Decision{Effect: r.Effect, Rule: r.ID, Reason: r.Reason}
		}
		matched = true
	}
	return best
}

var segmentSplit = regexp.MustCompile(`\r?\n|&&|\|\||;|\|`)
var unparsable = regexp.MustCompile("`|\\$\\(|<\\(|>\\(|\\$\\{|<<")

// decideScript avalia cada comando simples do script e fica com o mais restritivo.
// Substituição de comando e construções que não dá para analisar viram aprovação.
func (p *Policy) decideScript(a Action) Decision {
	cmd, _ := a.Params["command"].(string)
	if strings.TrimSpace(cmd) == "" {
		return Decision{Effect: Deny, Rule: "builtin.empty_script", Reason: "Script sem comando."}
	}
	var best *Decision
	consider := func(d Decision) {
		if best == nil || severity(d.Effect) > severity(best.Effect) {
			c := d
			best = &c
		}
	}
	if unparsable.MatchString(cmd) {
		consider(Decision{Effect: Approve, Rule: "builtin.unparsable_script", Reason: "O script usa substituição de comando ou construções que a política não consegue analisar."})
	}
	for _, seg := range segmentSplit.Split(cmd, -1) {
		seg = strings.TrimSpace(seg)
		if seg == "" {
			continue
		}
		sub := a
		sub.Params = map[string]any{"command": seg}
		consider(p.combine(sub, ""))
	}
	if best == nil {
		return Decision{Effect: Deny, Rule: "builtin.empty_script", Reason: "Script sem comando."}
	}
	return *best
}

// ExplicitDeny informa se alguma regra de PROIBIÇÃO casa com a ação (ignora o
// padrão de negar). Serve a varreduras estáticas, como a de scripts de pacotes
// importados, onde só interessa o que é comprovadamente proibido.
func (p *Policy) ExplicitDeny(a Action) (Decision, bool) {
	hit := func(act Action) (Decision, bool) {
		for _, r := range p.Rules {
			if r.Effect == Deny && r.matches(act, HostOf(act.Target)) {
				return Decision{Effect: Deny, Risk: RiskForbidden, Rule: r.ID, Reason: r.Reason}, true
			}
		}
		return Decision{}, false
	}
	if a.Tool != ToolScript {
		return hit(a)
	}
	cmd, _ := a.Params["command"].(string)
	for _, seg := range segmentSplit.Split(cmd, -1) {
		seg = strings.TrimSpace(seg)
		if seg == "" {
			continue
		}
		sub := a
		sub.Params = map[string]any{"command": seg}
		if d, ok := hit(sub); ok {
			return d, true
		}
	}
	return Decision{}, false
}
