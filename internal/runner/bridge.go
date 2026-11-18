package runner

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ViniciusBernucci/le-fabrique/internal/flow"
	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
	"github.com/ViniciusBernucci/le-fabrique/internal/prcycle"
	"github.com/ViniciusBernucci/le-fabrique/internal/projects"
	"github.com/ViniciusBernucci/le-fabrique/internal/sentinel"
	"github.com/ViniciusBernucci/le-fabrique/internal/workflows"
)

// IssueBridge liga a execução de uma Issue da origem à Sentinela (ADR-0069): conforme o fluxo avança, pede as
// transições de status do catálogo (PLANEJAMENTO → IMPLEMENTACAO → REVIEW → DRAFT_PR) com a EVIDÊNCIA que a execução
// realmente produziu, e, quando a Draft PR é aberta de verdade, registra-a como gerenciada para o ciclo da PR (rodadas
// de revisão viram Issues de correção; o merge conclui). As guardas são as do pacote flow: se faltar evidência a
// transição é recusada e o motivo fica registrado na execução; a ponte nunca falha o fluxo. A projeção do status
// para o Jira/ClickUp é feita pela rotina de projeção da Sentinela (intenção na outbox).
type IssueBridge struct {
	DB    DB
	Coord *sentinel.Coordinator

	mu    sync.Mutex
	tried map[string]bool
}

func (b *IssueBridge) once(key string) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.tried == nil {
		b.tried = map[string]bool{}
	}
	if b.tried[key] {
		return false
	}
	b.tried[key] = true
	return true
}

type visits struct {
	ok  map[string]bool // nó percorrido por uma porta que não é de erro
	bad map[string]bool // nó que saiu por erro
}

func visitsOf(st workflows.RunState) visits {
	v := visits{ok: map[string]bool{}, bad: map[string]bool{}}
	for _, s := range st.Trace {
		switch s.Port {
		case "error":
			v.bad[s.Node] = true
		case "":
			// fim, espera e decisão não "completam" um nó de trabalho
		default:
			v.ok[s.Node] = true
		}
	}
	return v
}

func varsOf(st workflows.RunState, node string) map[string]any {
	m, _ := st.Vars[workflows.VarKey(node)].(map[string]any)
	return m
}

func num(m map[string]any, k string) int {
	switch v := m[k].(type) {
	case float64:
		return int(v)
	case int:
		return v
	}
	return 0
}

// Progress é o gancho de runner.Options.Progress.
func (b *IssueBridge) Progress(ctx context.Context, run Run, st workflows.RunState) {
	if b == nil || run.IssueKey == "" || run.Kind == KindTest {
		return
	}
	v := visitsOf(st)
	// 1) PLANEJAMENTO: o fluxo chegou ao planejamento.
	if v.ok["pronto_planejar"] || v.ok[workflows.GuardPlanning] {
		b.step(ctx, run, st, flow.Planejamento)
	}
	// 2) IMPLEMENTACAO: o Tech Lead entregou um plano aceito e as subtarefas foram materializadas.
	if v.ok[workflows.GuardPlanning] {
		b.step(ctx, run, st, flow.Implementacao)
	}
	// 3) REVIEW: implementações prontas e checks integrados aprovados na cabeça consolidada.
	if v.ok[workflows.GuardIntegrated] {
		b.step(ctx, run, st, flow.Review)
	}
	// 4) DRAFT_PR: a Draft PR foi aberta de verdade (não simulada) e o GitHub a confirmou.
	if v.ok[workflows.GuardDraftPR] {
		b.step(ctx, run, st, flow.DraftPR)
	}
}

func (b *IssueBridge) emit(ctx context.Context, run Run, f map[string]any) {
	f["step"] = "issue.transition"
	f["at"] = time.Now().UTC().Format(time.RFC3339Nano)
	_ = Emit(context.WithoutCancel(ctx), b.DB, run.ProjectID, run.ExecutionID, EventDiag, f)
}

func (b *IssueBridge) step(ctx context.Context, run Run, st workflows.RunState, to string) {
	if !b.once(run.ExecutionID + ":" + to) {
		return
	}
	ctx = context.WithoutCancel(ctx)
	iss, err := b.Coord.Get(ctx, run.ProjectID, run.IssueKey)
	if err != nil {
		b.emit(ctx, run, map[string]any{"to": to, "ok": false, "erro": "a Issue " + run.IssueKey + " não está rastreada pela Sentinela: " + err.Error()})
		return
	}
	if iss.FeatureID == "" {
		_, _ = b.DB.Exec(ctx, `UPDATE flow_issues SET feature_id = (SELECT feature_id FROM executions WHERE id = $1) WHERE id = $2 AND feature_id IS NULL`, run.ExecutionID, iss.ID)
	}
	ev, why := b.evidence(ctx, run, st, iss.Status, to)
	if why != "" {
		b.emit(ctx, run, map[string]any{"to": to, "ok": false, "decision": "não pedida", "erro": why})
		return
	}
	out, err := b.Coord.Transition(ctx, sentinel.Request{
		ProjectID: run.ProjectID, IssueKey: run.IssueKey, To: to, Actor: sentinel.ActorSentinel, Observer: "fabrica",
		ObservationKey: "run:" + run.ExecutionID + ":" + to, ObservedAt: time.Now().UTC(), ExpectedVersion: iss.Version, Evidence: ev,
	})
	f := map[string]any{"to": to, "from": iss.Status, "decision": out.Decision, "duplicate": out.Duplicate, "ok": err == nil && (out.Decision == sentinel.DecisionApplied || out.Duplicate)}
	if err != nil {
		f["erro"] = err.Error()
	}
	if len(out.Reasons) > 0 {
		r := ""
		for _, p := range out.Reasons {
			r += p.String() + "; "
		}
		f["motivos"] = r
	}
	b.emit(ctx, run, f)
	if to == flow.DraftPR && f["ok"] == true {
		b.linkPR(ctx, run, st)
	}
}

// evidence monta a evidência da transição a partir do que a execução fez. why != "" indica que ainda não dá para pedir.
func (b *IssueBridge) evidence(ctx context.Context, run Run, st workflows.RunState, from, to string) (flow.Evidence, string) {
	v := visitsOf(st)
	plan := varsOf(st, workflows.GuardPlanning)
	front, back := num(plan, "subtarefas_frontend"), num(plan, "subtarefas_backend")
	tracks := map[flow.Track]flow.Stage{}
	reviewed := func(impl, review string, n int, tr flow.Track) {
		if n == 0 {
			return
		}
		switch {
		case v.ok[review]:
			tracks[tr] = flow.StageReviewed
		case v.ok[impl]:
			tracks[tr] = flow.StageImplemented
		default:
			tracks[tr] = flow.StageImplementing
		}
	}
	reviewed(workflows.GuardImplFront, workflows.GuardReviewFront, front, flow.Frontend)
	reviewed(workflows.GuardImplBack, workflows.GuardReviewBack, back, flow.Backend)

	ev := flow.Evidence{}
	switch to {
	case flow.Planejamento:
		p, err := projects.Get(ctx, b.DB, run.ProjectID)
		if err != nil {
			return ev, err.Error()
		}
		ev.ProjectActive = p.State == "active"
		ev.WorkflowPublished, ev.InputsValid, ev.PlanningSlot = true, true, true // a execução só nasce de versão publicada e revalidada
		ev.FocusedPlanning = from == flow.CorrecaoPR
	case flow.Implementacao:
		ev.PlanValid = num(plan, "plano_versao") > 0
		ev.SubtasksMaterial = num(plan, "subtarefas") > 0
		if !ev.PlanValid {
			return ev, "o Tech Lead não entregou um plano aceito (sem plano_versao nas saídas do planejamento)"
		}
	case flow.Review:
		ev.Tracks, ev.GatesVerified = tracks, v.ok[workflows.GuardIntegrated]
	case flow.DraftPR:
		pr := varsOf(st, workflows.GuardDraftPR)
		real := num(pr, "pr_numero") > 0 && pr["simulado"] != true
		if !real {
			return ev, "a Draft PR não foi aberta no GitHub (simulada ou envio desligado): nada a confirmar"
		}
		ev.Tracks = tracks
		ev.ReviewsPassed = len(v.bad) == 0
		ev.CommitsMerged = v.ok[workflows.GuardConsolidate]
		ev.IntegratedHead = v.ok[workflows.GuardIntegrated]
		ev.GitHubConfirms = true
	}
	return ev, ""
}

func (b *IssueBridge) linkPR(ctx context.Context, run Run, st workflows.RunState) {
	pr := varsOf(st, workflows.GuardDraftPR)
	n := num(pr, "pr_numero")
	branch, _ := pr["branch"].(string)
	var featureID string
	_ = b.DB.QueryRow(ctx, `SELECT feature_id::text FROM executions WHERE id = $1`, run.ExecutionID).Scan(&featureID)
	err := (&prcycle.Service{DB: b.DB}).Link(ctx, run.ProjectID, featureID, n, run.IssueKey, branch)
	f := map[string]any{"step": "pr.linked", "pr": n, "issue": run.IssueKey, "ok": err == nil}
	if err != nil {
		f["erro"] = persistence.MapError(err).Error()
	}
	f["detail"] = fmt.Sprintf("PR #%d registrada como gerenciada: o ciclo da PR acompanha as revisões e o merge", n)
	_ = Emit(ctx, b.DB, run.ProjectID, run.ExecutionID, EventDiag, f)
}
