package contract

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// SchemaVersion é a versão do schema; leitura histórica continua suportada.
const SchemaVersion = 1

// Origens dos campos (proveniência).
const (
	FromBackend  = "backend"
	FromTechLead = "tech_lead"
	FromSentinel = "sentinel"
)

// Limites.
const (
	maxList    = 100
	maxText    = 500
	maxPathLen = 300
)

// RepoIdentity identifica o repositório.
type RepoIdentity struct {
	Owner string `json:"owner"`
	Name  string `json:"name"`
}

// Versions fixa as versões usadas.
type Versions struct {
	Workflow string `json:"workflow"`
	Agent    string `json:"agent"`
	Skill    string `json:"skill"`
}

// Limits são os limites efetivos.
type Limits struct {
	TimeoutSeconds int `json:"timeout_seconds"`
	MaxRounds      int `json:"max_rounds"`
	BudgetCents    int `json:"budget_cents"`
}

// Deterministic é o que só o backend preenche.
type Deterministic struct {
	ProjectID        string       `json:"project_id"`
	IssueID          string       `json:"issue_id"`
	SubtaskID        string       `json:"subtask_id,omitempty"`
	ExecutionID      string       `json:"execution_id,omitempty"`
	Phase            string       `json:"phase"`
	Status           string       `json:"status"`
	OfficialPriority *int         `json:"official_priority,omitempty"`
	Repo             RepoIdentity `json:"repo"`
	Branch           string       `json:"branch"`
	BaseSHA          string       `json:"base_sha"`
	Worktree         string       `json:"worktree"`
	Versions         Versions     `json:"versions"`
	Limits           Limits       `json:"limits"`
	Permissions      []string     `json:"permissions"`
}

// Proposal é o que o Tech Lead pode propor.
type Proposal struct {
	Criteria        []string `json:"criteria"`
	Risk            string   `json:"risk"`
	Dependencies    []string `json:"dependencies"`
	SharedResources []string `json:"shared_resources"`
	RelevantFiles   []string `json:"relevant_files"`
}

// References são os vínculos do cabeçalho do despacho.
type References struct {
	RequirementsRevision string `json:"requirements_revision,omitempty"`
	PlanVersion          int    `json:"plan_version,omitempty"`
	CheckpointID         string `json:"checkpoint_id,omitempty"`
}

// Contract é o corpo imutável (sem hash; o hash cobre exatamente estes campos).
type Contract struct {
	SchemaVersion int               `json:"schema_version"`
	Version       int               `json:"version"`
	ParentHash    string            `json:"parent_hash"`
	Deterministic Deterministic     `json:"deterministic"`
	Proposal      Proposal          `json:"proposal"`
	Provenance    map[string]string `json:"provenance"`
	References    References        `json:"references"`
}

// Problem é um defeito da proposta ou do contrato.
type Problem struct{ Field, Message string }

func (p Problem) String() string { return p.Field + ": " + p.Message }

// ErrRejected indica proposta não publicada.
var ErrRejected = errors.New("contrato rejeitado: nada foi publicado")

// RejectedError carrega os motivos.
type RejectedError struct{ Problems []Problem }

func (e *RejectedError) Error() string {
	ps := make([]string, 0, len(e.Problems))
	for _, p := range e.Problems {
		ps = append(ps, p.String())
	}
	return ErrRejected.Error() + ": " + strings.Join(ps, "; ")
}
func (e *RejectedError) Unwrap() error { return ErrRejected }

// ParseProposal decodifica a proposta do Tech Lead recusando qualquer campo fora do schema:
// tentar enviar permissions, project_id, worktree etc. é rejeitado.
func ParseProposal(raw []byte) (Proposal, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	var p Proposal
	if err := dec.Decode(&p); err != nil {
		return Proposal{}, &RejectedError{[]Problem{{"proposal", "fora do schema: " + err.Error()}}}
	}
	if dec.More() {
		return Proposal{}, &RejectedError{[]Problem{{"proposal", "conteúdo extra após o objeto"}}}
	}
	return p, nil
}

var shaRe = regexp.MustCompile(`^[0-9a-f]{40,64}$`)

func relOK(p string) bool {
	if p == "" || len(p) > maxPathLen || strings.HasPrefix(p, "/") || strings.ContainsAny(p, "\x00\n\r\\") {
		return false
	}
	for _, seg := range strings.Split(p, "/") {
		if seg == ".." || seg == "" || seg == "." {
			return false
		}
	}
	return true
}

// Graph mapeia subtarefa → dependências já conhecidas (para detectar ciclo).
type Graph map[string][]string

func (g Graph) cycle(start string, deps []string) bool {
	visited := map[string]bool{}
	var walk func(n string) bool
	walk = func(n string) bool {
		if n == start {
			return true
		}
		if visited[n] {
			return false
		}
		visited[n] = true
		for _, d := range g[n] {
			if walk(d) {
				return true
			}
		}
		return false
	}
	for _, d := range deps {
		if walk(d) {
			return true
		}
	}
	return false
}

// Validate confere a proposta e o dado determinístico. Pura.
func Validate(d Deterministic, p Proposal, g Graph) []Problem {
	var ps []Problem
	add := func(f, m string) { ps = append(ps, Problem{f, m}) }
	required := map[string]string{"project_id": d.ProjectID, "issue_id": d.IssueID, "phase": d.Phase, "status": d.Status,
		"repo.owner": d.Repo.Owner, "repo.name": d.Repo.Name}
	if d.SubtaskID != "" { // o contrato inicial (PRD) não exige o que só o planejamento descobre
		required["branch"], required["worktree"] = d.Branch, d.Worktree
		if !shaRe.MatchString(d.BaseSHA) {
			add("base_sha", "SHA base inválida")
		}
	}
	keys := make([]string, 0, len(required))
	for f := range required {
		keys = append(keys, f)
	}
	sort.Strings(keys)
	for _, f := range keys {
		if strings.TrimSpace(required[f]) == "" {
			add(f, "obrigatório (preenchido pelo backend)")
		}
	}
	for _, seg := range strings.Split(d.Worktree, "/") {
		if seg == ".." {
			add("worktree", "caminho inválido")
			break
		}
	}
	if d.Limits.TimeoutSeconds <= 0 {
		add("limits.timeout_seconds", "deve ser positivo")
	}
	if len(p.Criteria) == 0 {
		add("criteria", "informe ao menos um critério técnico")
	}
	for i, c := range p.Criteria {
		if len(strings.TrimSpace(c)) < 8 || len(c) > maxText {
			add(fmt.Sprintf("criteria[%d]", i), "critério vazio, curto demais ou longo demais")
		}
	}
	switch p.Risk {
	case "low", "medium", "high":
	default:
		add("risk", "use low, medium ou high")
	}
	if len(p.Dependencies) > maxList || len(p.SharedResources) > maxList || len(p.RelevantFiles) > maxList {
		add("proposal", "listas longas demais")
	}
	for _, f := range p.RelevantFiles {
		if !relOK(f) {
			add("relevant_files", "caminho fora do escopo do worktree: "+f)
		}
	}
	for _, r := range p.SharedResources {
		if strings.TrimSpace(r) == "" || len(r) > maxText {
			add("shared_resources", "recurso inválido")
		}
	}
	if d.SubtaskID != "" {
		for _, dep := range p.Dependencies {
			if dep == d.SubtaskID {
				add("dependencies", "dependência de si mesma")
			}
		}
		if g.cycle(d.SubtaskID, p.Dependencies) {
			add("dependencies", "dependência cíclica")
		}
	}
	return ps
}

// Canonical serializa de forma determinística (campos em ordem fixa, mapas ordenados).
func (c Contract) Canonical() []byte {
	b, _ := json.Marshal(c)
	return b
}

// Hash é o SHA-256 do corpo canônico.
func (c Contract) Hash() string {
	h := sha256.Sum256(c.Canonical())
	return hex.EncodeToString(h[:])
}

// provenance monta a proveniência por campo.
func provenance() map[string]string {
	m := map[string]string{}
	for _, f := range []string{"project_id", "issue_id", "subtask_id", "execution_id", "phase", "status", "official_priority", "repo",
		"branch", "base_sha", "worktree", "versions", "limits", "permissions"} {
		m[f] = FromBackend
	}
	for _, f := range []string{"criteria", "dependencies", "shared_resources", "relevant_files"} {
		m[f] = FromTechLead
	}
	m["risk"] = FromSentinel // proposta do Tech Lead com piso aplicado pela Sentinela
	return m
}

// Build monta o contrato (sem versão/parent; a Store os define). Aplica o piso de risco.
func Build(d Deterministic, p Proposal, riskFloor string, refs References) Contract {
	p.Risk = maxRisk(p.Risk, riskFloor)
	norm := func(in []string) []string {
		out := append([]string{}, in...)
		sort.Strings(out)
		return out
	}
	p.Dependencies, p.SharedResources, p.RelevantFiles = norm(p.Dependencies), norm(p.SharedResources), norm(p.RelevantFiles)
	d.Permissions = norm(d.Permissions)
	return Contract{SchemaVersion: SchemaVersion, Deterministic: d, Proposal: p, Provenance: provenance(), References: refs}
}

var riskRank = map[string]int{"low": 1, "medium": 2, "high": 3}

func maxRisk(a, b string) string {
	if riskRank[b] > riskRank[a] {
		return b
	}
	return a
}
