package projectres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/ViniciusBernucci/le-fabrique/internal/agents"
	"github.com/ViniciusBernucci/le-fabrique/internal/models"
	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
	"github.com/ViniciusBernucci/le-fabrique/internal/projects"
	"github.com/ViniciusBernucci/le-fabrique/internal/releases"
	"github.com/ViniciusBernucci/le-fabrique/internal/selection"
	"github.com/ViniciusBernucci/le-fabrique/internal/skills"
	"github.com/ViniciusBernucci/le-fabrique/internal/stdflow"
	"github.com/ViniciusBernucci/le-fabrique/internal/tools"
	"github.com/ViniciusBernucci/le-fabrique/internal/workflows"
)

// Tipos de recurso associável.
const (
	KindWorkflow = "workflow"
	KindAgent    = "agent"
	KindSkill    = "skill"
	KindTool     = "tool"
	KindModel    = "model"
)

var kinds = map[string]bool{KindWorkflow: true, KindAgent: true, KindSkill: true, KindTool: true, KindModel: true}

// ResourceInput é uma associação escolhida.
type ResourceInput struct {
	Kind     string         `json:"kind"`
	Ref      string         `json:"ref"`
	Override map[string]any `json:"override"`
}

// Resource é uma associação gravada, com o nome do recurso global.
type Resource struct {
	Kind      string            `json:"kind"`
	Ref       string            `json:"ref"`
	Name      string            `json:"name"`
	Override  map[string]any    `json:"override"`
	Effective *agents.Effective `json:"effective,omitempty"` // agente: base global + sobrescrita local
}

// nameOf confere que o recurso global existe e devolve o nome.
func nameOf(ctx context.Context, db persistence.DBTX, kind, ref string) (string, error) {
	var name string
	var err error
	switch kind {
	case KindWorkflow:
		var w workflows.Workflow
		w, err = workflows.Get(ctx, db, ref)
		name = w.Name
	case KindAgent:
		var a agents.Agent
		a, err = agents.Get(ctx, db, ref)
		name = a.Name
	case KindSkill:
		var s skills.Skill
		s, err = skills.Get(ctx, db, ref)
		name = s.Name
	case KindTool:
		var t tools.Tool
		t, err = tools.Get(ctx, db, ref)
		name = t.Name
	case KindModel:
		var m models.Model
		m, err = models.Get(ctx, db, ref)
		name = m.Name
	}
	if err != nil {
		if errors.Is(err, persistence.ErrNotFound) || errors.Is(err, persistence.ErrInvalidInput) {
			return "", &FieldError{"ref", fmt.Sprintf("o recurso %s %q não existe no catálogo global", kind, ref)}
		}
		return "", err
	}
	return name, nil
}

// overrideNode transforma a sobrescrita local num nó, para reutilizar as regras do agente.
func overrideNode(o map[string]any) workflows.Node {
	return workflows.Node{Type: workflows.TAgent, Props: o}
}

// SetResources substitui, de forma atômica, as associações do projeto. Cada
// recurso precisa existir no catálogo global; só agentes aceitam sobrescrita
// (modelo, conexão, instruções extras, ferramentas reduzidas, tempo limite),
// validada pelas mesmas regras do nó. O recurso global nunca é alterado.
func SetResources(ctx context.Context, db DB, projectID string, in []ResourceInput) ([]Resource, error) {
	seen := map[string]bool{}
	type row struct {
		in   ResourceInput
		json []byte
	}
	var rows []row
	for _, r := range in {
		r.Ref = strings.TrimSpace(r.Ref)
		if !kinds[r.Kind] {
			return nil, &FieldError{"kind", fmt.Sprintf("tipo de recurso inválido: %q", r.Kind)}
		}
		if r.Ref == "" {
			return nil, &FieldError{"ref", "informe o recurso"}
		}
		key := r.Kind + "\x00" + r.Ref
		if seen[key] {
			return nil, &FieldError{"ref", fmt.Sprintf("o recurso %s %q foi informado duas vezes", r.Kind, r.Ref)}
		}
		seen[key] = true
		if _, err := nameOf(ctx, db, r.Kind, r.Ref); err != nil {
			return nil, err
		}
		if len(r.Override) > 0 {
			if r.Kind != KindAgent {
				return nil, &FieldError{"override", "só agentes aceitam sobrescrita local"}
			}
			base, err := agents.Get(ctx, db, r.Ref)
			if err != nil {
				return nil, err
			}
			if _, err := agents.Resolve(base, overrideNode(r.Override)); err != nil {
				return nil, &FieldError{"override", err.Error()}
			}
		}
		if r.Override == nil {
			r.Override = map[string]any{}
		}
		b, _ := json.Marshal(r.Override)
		rows = append(rows, row{r, b})
	}
	err := persistence.WithTx(ctx, db, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 5))`, projectID); err != nil {
			return persistence.MapError(err)
		}
		if _, err := tx.Exec(ctx, `DELETE FROM project_resources WHERE project_id = $1`, projectID); err != nil {
			return persistence.MapError(err)
		}
		for _, r := range rows {
			if _, err := tx.Exec(ctx, `INSERT INTO project_resources (project_id, kind, ref, override) VALUES ($1, $2, $3, $4::jsonb)`,
				projectID, r.in.Kind, r.in.Ref, string(r.json)); err != nil {
				return persistence.MapError(err)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return ListResources(ctx, db, projectID)
}

// ListResources devolve as associações, com a configuração efetiva dos agentes.
func ListResources(ctx context.Context, db persistence.DBTX, projectID string) ([]Resource, error) {
	rows, err := db.Query(ctx, `SELECT kind, ref, override FROM project_resources WHERE project_id = $1 ORDER BY kind, ref`, projectID)
	if err != nil {
		return nil, persistence.MapError(err)
	}
	var out []Resource
	for rows.Next() {
		var r Resource
		var o []byte
		if err := rows.Scan(&r.Kind, &r.Ref, &o); err != nil {
			rows.Close()
			return nil, persistence.MapError(err)
		}
		r.Override = map[string]any{}
		_ = json.Unmarshal(o, &r.Override)
		out = append(out, r)
	}
	rows.Close()
	if out == nil {
		out = []Resource{}
	}
	for i := range out {
		name, err := nameOf(ctx, db, out[i].Kind, out[i].Ref)
		if err != nil {
			out[i].Name = "(removido do catálogo)"
			continue
		}
		out[i].Name = name
		if out[i].Kind == KindAgent {
			if base, err := agents.Get(ctx, db, out[i].Ref); err == nil {
				if eff, err := agents.Resolve(base, overrideNode(out[i].Override)); err == nil {
					out[i].Effective = &eff
				}
			}
		}
	}
	return out, nil
}

func init() { projects.AddValidator("project-resources", Validate) }

const area = "resources"

// Validate acrescenta ao relatório do projeto os itens de recursos associados.
func Validate(ctx context.Context, db persistence.DBTX, p projects.Project) ([]projects.Check, error) {
	res, err := ListResources(ctx, db, p.ID)
	if err != nil {
		return nil, err
	}
	var wfProblems, agentProblems, secretProblems []string

	// Workflows: associados e alvos da seleção precisam estar publicados e válidos.
	wfIDs := map[string]bool{}
	for _, r := range res {
		if r.Kind == KindWorkflow {
			wfIDs[r.Ref] = true
		}
	}
	cfg, err := selection.Load(ctx, db, p.ID)
	if err != nil {
		return nil, err
	}
	if cfg.Default != nil {
		wfIDs[cfg.Default.WorkflowID] = true
		if cfg.Default.SubflowID != "" {
			wfIDs[cfg.Default.SubflowID] = true
		}
	}
	for _, r := range cfg.Rules {
		wfIDs[r.WorkflowID] = true
		if r.SubflowID != "" {
			wfIDs[r.SubflowID] = true
		}
	}
	ids := make([]string, 0, len(wfIDs))
	for id := range wfIDs {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		wfProblems = append(wfProblems, workflowProblems(ctx, db, id)...)
	}

	for _, r := range res {
		switch r.Kind {
		case KindAgent:
			agentProblems = append(agentProblems, agentProblemsOf(ctx, db, r)...)
		case KindModel:
			if m, err := models.Get(ctx, db, r.Ref); err == nil && m.State != "active" {
				agentProblems = append(agentProblems, fmt.Sprintf("o modelo %q está desativado", m.Name))
			}
		}
	}
	prof, err := GetProfile(ctx, db, p.ID)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(prof.SecretRefs))
	for n := range prof.SecretRefs {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		m := refPattern.FindStringSubmatch(prof.SecretRefs[n])
		if m == nil {
			secretProblems = append(secretProblems, fmt.Sprintf("a referência de %s é inválida", n))
			continue
		}
		var state string
		if err := db.QueryRow(ctx, `SELECT state FROM connections WHERE id = $1`, m[1]).Scan(&state); err != nil || state != "active" {
			secretProblems = append(secretProblems, fmt.Sprintf("a conexão do segredo %s não existe ou está revogada", n))
		}
	}
	mk := func(id, label string, problems []string, action string) projects.Check {
		c := projects.Check{ID: id, Area: area, Label: label, Status: projects.CheckOK}
		if len(problems) > 0 {
			c.Status, c.Action = projects.CheckPending, strings.Join(problems, "; ")+". "+action
		}
		return c
	}
	return []projects.Check{
		mk("resources.workflows", "Workflows publicados e válidos", wfProblems, "Publique o workflow no editor ou retire-o da seleção."),
		mk("resources.agents", "Agentes, modelos e contas compatíveis", agentProblems, "Ajuste a conta ou o modelo do agente (ou da sobrescrita do projeto)."),
		mk("resources.secrets", "Referências de segredo", secretProblems, "Escolha uma conexão ativa para a variável no perfil técnico."),
	}, nil
}

func workflowProblems(ctx context.Context, db persistence.DBTX, id string) []string {
	w, err := workflows.Get(ctx, db, id)
	if err != nil {
		return []string{fmt.Sprintf("o workflow %s não existe mais", id)}
	}
	ver, err := releases.Published(ctx, db, id)
	if errors.Is(err, releases.ErrNotPublished) {
		return []string{fmt.Sprintf("o workflow %q ainda é rascunho (sem versão publicada)", w.Name)}
	}
	if err != nil {
		return []string{fmt.Sprintf("não foi possível ler a versão publicada de %q", w.Name)}
	}
	var out []string
	for _, pr := range releases.Revalidate(ctx, db, ver) {
		out = append(out, fmt.Sprintf("o workflow %q deixou de ser executável: %s", w.Name, pr.Message))
	}
	return out
}

func agentProblemsOf(ctx context.Context, db persistence.DBTX, r Resource) []string {
	base, err := agents.Get(ctx, db, r.Ref)
	if err != nil {
		return []string{fmt.Sprintf("o agente %s não existe mais", r.Ref)}
	}
	eff, err := agents.Resolve(base, overrideNode(r.Override))
	if err != nil {
		return []string{fmt.Sprintf("agente %q: %v", base.Name, err)}
	}
	var out []string
	req := models.Requirement{Provider: eff.Provider, Model: eff.Model, NeedsTools: len(eff.Tools) > 0}
	if eff.ConnectionID != nil {
		req.ConnectionID = *eff.ConnectionID
		var state string
		if err := db.QueryRow(ctx, `SELECT state FROM connections WHERE id = $1`, *eff.ConnectionID).Scan(&state); err != nil || state != "active" {
			out = append(out, fmt.Sprintf("agente %q: a conta está revogada ou não existe", base.Name))
		}
	}
	v, err := models.Check(ctx, db, req)
	switch {
	case err != nil:
		out = append(out, fmt.Sprintf("agente %q: não foi possível verificar o modelo", base.Name))
	case !v.OK:
		out = append(out, fmt.Sprintf("agente %q: conta ou modelo incompatível (%s)", base.Name, strings.Join(v.Problems, " ")))
	}
	for _, f := range eff.Fallback {
		var state string
		if err := db.QueryRow(ctx, `SELECT state FROM connections WHERE id = $1`, f.ConnectionID).Scan(&state); err != nil || state != "active" {
			out = append(out, fmt.Sprintf("agente %q: uma conta da cadeia de fallback está revogada ou não existe", base.Name))
		}
	}
	return out
}

// Readiness diz se o projeto pode iniciar uma execução agora.
type Readiness struct {
	Ready    bool              `json:"ready"`
	Problems []string          `json:"problems"`
	Target   *selection.Target `json:"target,omitempty"`
}

// Ready exige projeto ativo, recursos válidos e um fluxo padrão PUBLICADO.
func Ready(ctx context.Context, db persistence.DBTX, projectID string) (Readiness, error) {
	p, err := projects.Get(ctx, db, projectID)
	if err != nil {
		return Readiness{}, err
	}
	out := Readiness{Problems: []string{}}
	if p.State != projects.StateActive {
		out.Problems = append(out.Problems, fmt.Sprintf("o projeto está %q: só projeto ativo inicia execução", p.State))
	}
	checks, err := Validate(ctx, db, p)
	if err != nil {
		return Readiness{}, err
	}
	for _, c := range checks {
		if c.Status != projects.CheckOK {
			out.Problems = append(out.Problems, c.Action)
		}
	}
	cfg, err := selection.Load(ctx, db, projectID)
	if err != nil {
		return Readiness{}, err
	}
	if cfg.Default == nil {
		out.Problems = append(out.Problems, "nenhum fluxo padrão definido na seleção do projeto")
	} else {
		out.Target = cfg.Default
	}
	// Workflow estrutural do projeto (adequação): rascunho incompleto ou sem versão publicada não executa.
	wfID, err := stdflow.Of(ctx, db, projectID)
	if err != nil {
		return Readiness{}, err
	}
	if wfID != "" {
		ver, perr := releases.Published(ctx, db, wfID)
		switch {
		case errors.Is(perr, persistence.ErrNotFound) || errors.Is(perr, releases.ErrNotPublished):
			out.Problems = append(out.Problems, "o workflow do projeto não está publicado")
			if wf, gerr := workflows.Get(ctx, db, wfID); gerr == nil {
				for _, is := range workflows.Validate(wf.Graph) {
					out.Problems = append(out.Problems, fmt.Sprintf("nó %s: %s", is.Node, is.Message))
				}
			}
		case perr != nil:
			return Readiness{}, perr
		default:
			for _, pr := range releases.Revalidate(ctx, db, ver) {
				out.Problems = append(out.Problems, fmt.Sprintf("nó %s: %s", pr.Node, pr.Message))
			}
		}
	}
	out.Ready = len(out.Problems) == 0
	return out, nil
}
