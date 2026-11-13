// Package prdgate é o gate do PRD (AJ04-02): a Sentinela confere os campos obrigatórios do projeto,
// o conteúdo acessível, a consistência dos requisitos e as referências. Incompleto: expõe pendências
// e a Issue permanece em PRD-DOC. Válido: publica o contrato inicial (uma vez por revisão) ANTES da
// transição para PRD-DOC-CONCLUIDO, que libera o Tech Lead.
package prdgate

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ViniciusBernucci/le-fabrique/internal/contract"
	"github.com/ViniciusBernucci/le-fabrique/internal/flow"
	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
	"github.com/ViniciusBernucci/le-fabrique/internal/sentinel"
)

// PRD é o documento de requisitos lido da origem.
type PRD struct {
	Key         string
	Revision    string // revisão observada na origem (ex.: versão do Jira/ClickUp)
	Title       string
	Description string
	Fields      map[string]string // campos do projeto (ex.: "Critérios de aceite")
	Criteria    []string
	References  []string
	Accessible  bool // o conteúdo pôde ser lido por inteiro
}

// Config é a configuração do projeto para o gate.
type Config struct {
	ProjectID      string
	RequiredFields []string
	Repo           contract.RepoIdentity
	Versions       contract.Versions
	Limits         contract.Limits
	Permissions    []string
	Priority       *int
	DefaultRisk    string
}

// Pending é uma pendência do PRD.
type Pending struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// Evaluate confere o PRD. Pura.
func Evaluate(cfg Config, p PRD) []Pending {
	var out []Pending
	add := func(f, m string) { out = append(out, Pending{f, m}) }
	if !p.Accessible {
		add("content", "conteúdo do PRD inacessível ou truncado")
	}
	if strings.TrimSpace(p.Title) == "" {
		add("title", "título obrigatório")
	}
	if len(strings.TrimSpace(p.Description)) < 20 {
		add("description", "descrição ausente ou curta demais")
	}
	for _, f := range cfg.RequiredFields {
		if strings.TrimSpace(p.Fields[f]) == "" {
			add("field:"+f, "campo obrigatório do projeto não preenchido")
		}
	}
	if len(p.Criteria) == 0 {
		add("criteria", "critérios de aceite ausentes")
	}
	seen := map[string]bool{}
	for i, c := range p.Criteria {
		k := strings.ToLower(strings.TrimSpace(c))
		if k == "" {
			add(fmt.Sprintf("criteria[%d]", i), "critério vazio")
		} else if seen[k] {
			add(fmt.Sprintf("criteria[%d]", i), "critério duplicado")
		}
		seen[k] = true
	}
	for i, r := range p.References {
		if strings.TrimSpace(r) == "" || strings.ContainsAny(r, " \t\n") {
			add(fmt.Sprintf("references[%d]", i), "referência inválida")
		}
	}
	if strings.TrimSpace(p.Revision) == "" {
		add("revision", "revisão da origem desconhecida")
	}
	return out
}

// Gate liga o gate à Sentinela.
type Gate struct {
	DB        sentinel.DB
	Contracts *contract.Store
	Sentinel  *sentinel.Coordinator
}

// Result é o resultado do gate.
type Result struct {
	Complete   bool               `json:"complete"`
	Pending    []Pending          `json:"pending,omitempty"`
	Contract   *contract.Snapshot `json:"contract,omitempty"`
	Transition string             `json:"transition,omitempty"` // decisão da Sentinela
	Repeated   bool               `json:"repeated"`
}

// Run avalia o PRD da Issue. Idempotente por (Issue, revisão): repetir não duplica contrato nem
// transição. A Issue precisa estar rastreada pela Sentinela.
func (g *Gate) Run(ctx context.Context, cfg Config, p PRD) (Result, error) {
	pend := Evaluate(cfg, p)
	pj, _ := json.Marshal(pend)
	if len(pend) > 0 {
		if _, err := g.DB.Exec(ctx, `INSERT INTO prd_gate_results (project_id, issue_key, revision, complete, problems) VALUES ($1,$2,$3,false,$4)
			ON CONFLICT (project_id, issue_key, revision) DO UPDATE SET problems=$4, evaluated_at=now() WHERE prd_gate_results.complete=false`,
			cfg.ProjectID, p.Key, p.Revision, pj); err != nil {
			return Result{}, persistence.MapError(err)
		}
		return Result{Complete: false, Pending: pend}, nil
	}
	risk := cfg.DefaultRisk
	if risk == "" {
		risk = "medium"
	}
	d := contract.Deterministic{ProjectID: cfg.ProjectID, IssueID: p.Key, Phase: flow.PRDDoc, Status: flow.PRDDocConcluido,
		OfficialPriority: cfg.Priority, Repo: cfg.Repo, Versions: cfg.Versions, Limits: cfg.Limits, Permissions: cfg.Permissions}
	snap, unchanged, err := g.Contracts.Publish(ctx, d, contract.Proposal{Criteria: p.Criteria, Risk: risk}, "", contract.References{RequirementsRevision: p.Revision}, nil)
	if err != nil {
		return Result{}, err
	}
	if _, err := g.DB.Exec(ctx, `INSERT INTO prd_gate_results (project_id, issue_key, revision, complete, problems, contract_id) VALUES ($1,$2,$3,true,'[]',$4)
		ON CONFLICT (project_id, issue_key, revision) DO UPDATE SET complete=true, problems='[]', contract_id=$4, evaluated_at=now()`,
		cfg.ProjectID, p.Key, p.Revision, snap.ID); err != nil {
		return Result{}, persistence.MapError(err)
	}
	// A publicação do contrato precede a transição que libera o Tech Lead.
	out, err := g.Sentinel.Transition(ctx, sentinel.Request{ProjectID: cfg.ProjectID, IssueKey: p.Key, To: flow.PRDDocConcluido, Actor: sentinel.ActorSentinel,
		Observer: "prd-gate", ObservationKey: "prd:" + p.Revision, RawStatus: flow.PRDDocConcluido,
		Evidence: flow.Evidence{PRDComplete: true, InitialContract: true}})
	if err != nil {
		return Result{}, err
	}
	return Result{Complete: true, Contract: &snap, Transition: out.Decision, Repeated: unchanged && out.Duplicate}, nil
}
