// Package correction conduz uma correção liberada de ponta a ponta (AJ09-03): só começa depois da
// liberação humana na origem, planeja de forma focalizada, assume a próxima vaga compatível com prioridade
// (sem interromper trabalho ativo), executa em worktrees temporárias, consolida e publica os commits
// validados na PR ORIGINAL, avançando a Issue pela Sentinela. Cada passo deixa evidência ligando rodada,
// Issue, contratos, commits e cabeça publicada.
package correction

import (
	"context"
	"errors"
	"fmt"

	"github.com/ViniciusBernucci/le-fabrique/internal/consolidate"
	"github.com/ViniciusBernucci/le-fabrique/internal/contract"
	"github.com/ViniciusBernucci/le-fabrique/internal/flow"
	"github.com/ViniciusBernucci/le-fabrique/internal/focusplan"
	"github.com/ViniciusBernucci/le-fabrique/internal/publication"
	"github.com/ViniciusBernucci/le-fabrique/internal/sentinel"
	"github.com/ViniciusBernucci/le-fabrique/internal/trackadmit"
	"github.com/ViniciusBernucci/le-fabrique/internal/worktrees"
)

// ErrAwaitingRelease: a correção ainda não foi liberada por um humano na origem.
var ErrAwaitingRelease = errors.New("a correção aguarda a liberação humana na origem (REPROVADO_PR): nada é despachado")

// Dispatcher reúne os componentes.
type Dispatcher struct {
	Sentinel *sentinel.Coordinator
	Admit    *trackadmit.Service
	Focus    *focusplan.Service
	Wt       *worktrees.Manager
	Cons     *consolidate.Consolidator
	Pub      *publication.Service
}

// PlanRequest pede o planejamento focalizado.
type PlanRequest struct {
	ProjectID, FeatureID, IssueKey string
	Input                          focusplan.Input
	Base                           focusplan.ContractBase
	Refs                           contract.References
}

// Plan confere a liberação humana, planeja só o escopo afetado e leva a Issue a IMPLEMENTACAO.
func (d *Dispatcher) Plan(ctx context.Context, r PlanRequest) (focusplan.Result, error) {
	iss, err := d.Sentinel.Get(ctx, r.ProjectID, r.IssueKey)
	if err != nil {
		return focusplan.Result{}, err
	}
	if iss.Status != flow.CorrecaoPR {
		return focusplan.Result{}, fmt.Errorf("%w (estado atual: %s)", ErrAwaitingRelease, iss.Status)
	}
	res, err := d.Focus.Accept(ctx, r.ProjectID, r.FeatureID, r.Input, r.Base, r.Refs)
	if err != nil {
		return res, err
	}
	step := func(to string, ev flow.Evidence) error {
		out, err := d.Sentinel.Transition(ctx, sentinel.Request{ProjectID: r.ProjectID, IssueKey: r.IssueKey, To: to, Actor: sentinel.ActorSentinel,
			Observer: "correction", ObservationKey: "correction:" + r.Input.Round.Key + ":" + to, Evidence: ev})
		if err != nil {
			return err
		}
		if out.Decision != sentinel.DecisionApplied && !out.Duplicate {
			return fmt.Errorf("transição para %s recusada: %v", to, out.Reasons)
		}
		return nil
	}
	if err := step(flow.Planejamento, flow.Evidence{ProjectActive: true, WorkflowPublished: true, InputsValid: true, PlanningSlot: true, FocusedPlanning: true}); err != nil {
		return res, err
	}
	return res, step(flow.Implementacao, flow.Evidence{PlanValid: true, SubtasksMaterial: true})
}

// Dispatched é um trabalho admitido com a worktree provisionada.
type Dispatched struct {
	trackadmit.Started
	Attempt worktrees.Attempt
}

// Dispatch admite o que couber (a correção tem prioridade, o trabalho ativo segue) e provisiona uma
// worktree temporária para cada implementação admitida a partir da cabeça atual da PR.
func (d *Dispatcher) Dispatch(ctx context.Context, projectID string, lim sentinel.Limits, headSHA string, attemptNo int, pathsOf func(featureID, taskKey string) []string) ([]Dispatched, []trackadmit.Waiting, error) {
	started, waiting, err := d.Admit.Tick(ctx, projectID, lim)
	if err != nil {
		return nil, waiting, err
	}
	var out []Dispatched
	for _, s := range started {
		if s.Stage != trackadmit.StageImplementation {
			out = append(out, Dispatched{Started: s})
			continue
		}
		a, err := d.Wt.Provision(ctx, worktrees.Spec{ProjectID: projectID, FeatureID: s.FeatureID, IssueKey: s.TaskKey, Track: s.Track, Kind: "correction", Attempt: attemptNo,
			BaseSHA: headSHA, Paths: pathsOf(s.FeatureID, s.TaskKey)})
		if err != nil {
			return out, waiting, err
		}
		_ = d.Wt.SetWriter(ctx, a.ID, true)
		out = append(out, Dispatched{Started: s, Attempt: a})
	}
	return out, waiting, nil
}

// FinishRequest encerra a correção: consolida as tentativas e publica na PR original.
type FinishRequest struct {
	ProjectID, FeatureID, IssueKey string
	RoundKey                       string
	DeliveryBranch                 string
	Attempts                       []consolidate.Attempt
	Publish                        publication.Correction
}

// Finish consolida as tentativas na branch de entrega, publica na PR original (gates conferidos pela
// publicação) e leva a Issue a DRAFT_PR. Conflito ou bloqueio nada publicam e preservam o trabalho.
func (d *Dispatcher) Finish(ctx context.Context, r FinishRequest) (publication.Result, error) {
	if _, err := d.Cons.Run(ctx, consolidate.Request{ProjectID: r.ProjectID, FeatureID: r.FeatureID, DeliveryBranch: r.DeliveryBranch, Attempts: r.Attempts}); err != nil {
		return publication.Result{}, err
	}
	res, err := d.Pub.PublishCorrection(ctx, r.Publish)
	if err != nil {
		return res, err
	}
	tracks := map[flow.Track]flow.Stage{}
	for _, tr := range r.Publish.Required {
		tracks[tr] = flow.StageReviewed
	}
	for _, s := range []struct {
		to string
		ev flow.Evidence
	}{
		{flow.Review, flow.Evidence{Tracks: tracks, GatesVerified: true}},
		{flow.DraftPR, flow.Evidence{Tracks: tracks, ReviewsPassed: true, CommitsMerged: true, IntegratedHead: true, GitHubConfirms: true}},
	} {
		out, err := d.Sentinel.Transition(ctx, sentinel.Request{ProjectID: r.ProjectID, IssueKey: r.IssueKey, To: s.to, Actor: sentinel.ActorSentinel,
			Observer: "correction", ObservationKey: "correction:" + r.RoundKey + ":" + s.to + ":" + res.Head, Evidence: s.ev})
		if err != nil {
			return res, err
		}
		if out.Decision != sentinel.DecisionApplied && !out.Duplicate {
			return res, fmt.Errorf("transição para %s recusada: %v", s.to, out.Reasons)
		}
	}
	return res, nil
}
