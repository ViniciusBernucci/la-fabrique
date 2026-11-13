package flow

import (
	"fmt"
	"sort"
)

// Track é uma trilha de subtarefas.
type Track string

// Trilhas.
const (
	Frontend Track = "FRONTEND"
	Backend  Track = "BACKEND"
)

// Stage é a etapa obrigatória de uma trilha.
type Stage int

// Etapas, da mais atrasada à mais adiantada.
const (
	StageNone Stage = iota
	StageImplementing
	StageImplemented
	StageReviewed
)

// Evidence reúne o que a Sentinela conferiu para a transição pedida.
type Evidence struct {
	// PRD-DOC → PRD-DOC-CONCLUIDO
	PRDComplete     bool
	InitialContract bool // contrato inicial publicado e vinculado à revisão
	// REPROVADO_PR → CORRECAO_PR
	HumanRelease      bool
	LinkedToFeaturePR bool
	// PRD-DOC-CONCLUIDO / CORRECAO_PR → PLANEJAMENTO
	ProjectActive     bool
	WorkflowPublished bool
	InputsValid       bool
	PlanningSlot      bool
	FocusedPlanning   bool // exigido quando vem de correção
	// PLANEJAMENTO → IMPLEMENTACAO
	PlanValid        bool
	SubtasksMaterial bool
	// por trilha obrigatória
	Tracks map[Track]Stage
	// IMPLEMENTACAO → REVIEW agregado e REVIEW → DRAFT_PR
	GatesVerified  bool // testes, evidências e gates exigidos
	ReviewsPassed  bool
	CommitsMerged  bool // commits consolidados
	IntegratedHead bool // verificações integradas válidas para a cabeça publicada
	GitHubConfirms bool
	// DRAFT_PR → CONCLUIDO
	MergeConfirmed bool
}

// Problem é um motivo de recusa.
type Problem struct{ Code, Message string }

func (p Problem) String() string { return p.Code + ": " + p.Message }

// Rejection reúne os motivos de uma transição recusada.
type Rejection struct {
	From, To string
	Problems []Problem
}

func (r *Rejection) Error() string {
	s := fmt.Sprintf("transição %s → %s recusada", r.From, r.To)
	for _, p := range r.Problems {
		s += "; " + p.String()
	}
	return s
}

// Check aplica a guarda da transição. Devolve nil se válida, ou *Rejection.
func Check(from, to string, ev Evidence) error {
	f, ok1 := Normalize(from)
	t, ok2 := Normalize(to)
	if !ok1 || !ok2 {
		return &Rejection{from, to, []Problem{{"unknown_status", "status fora do catálogo"}}}
	}
	if !Allowed(f.Code, t.Code) {
		return &Rejection{f.Code, t.Code, []Problem{{"not_allowed", "a transição não existe na tabela"}}}
	}
	var ps []Problem
	need := func(ok bool, code, msg string) {
		if !ok {
			ps = append(ps, Problem{code, msg})
		}
	}
	switch {
	case f.Code == PRDDoc && t.Code == PRDDocConcluido:
		need(ev.PRDComplete, "prd_incomplete", "PRD externo incompleto ou inconsistente")
		need(ev.InitialContract, "no_initial_contract", "contrato inicial não publicado")
	case f.Code == ReprovadoPR && t.Code == CorrecaoPR:
		need(ev.HumanRelease, "no_human_release", "falta liberação humana na origem")
		need(ev.LinkedToFeaturePR, "unlinked", "sem vínculo inequívoco à feature e à PR")
	case t.Code == Planejamento:
		need(ev.ProjectActive, "project_inactive", "projeto inativo")
		need(ev.WorkflowPublished, "no_workflow", "workflow não publicado")
		need(ev.InputsValid, "invalid_inputs", "insumos inválidos")
		need(ev.PlanningSlot, "no_planning_slot", "sem vaga de planejamento")
		if f.Code == CorrecaoPR {
			need(ev.FocusedPlanning, "not_focused", "correção exige planejamento focalizado")
		}
	case t.Code == Implementacao:
		need(ev.PlanValid, "invalid_plan", "plano técnico inválido")
		need(ev.SubtasksMaterial, "no_subtasks", "subtarefas obrigatórias não materializadas")
	case t.Code == Review:
		for _, tr := range sortedTracks(ev.Tracks) {
			need(ev.Tracks[tr] >= StageImplemented, "track_implementing", "trilha "+string(tr)+" ainda implementando")
		}
		need(len(ev.Tracks) > 0, "no_tracks", "nenhuma trilha obrigatória")
		need(ev.GatesVerified, "gates_pending", "testes, evidências ou gates exigidos pendentes")
	case t.Code == DraftPR:
		for _, tr := range sortedTracks(ev.Tracks) {
			need(ev.Tracks[tr] >= StageReviewed, "track_unreviewed", "trilha "+string(tr)+" sem review aprovado")
		}
		need(ev.ReviewsPassed, "reviews_pending", "revisões ou checks exigidos não aprovados")
		need(ev.CommitsMerged, "commits_not_consolidated", "commits não consolidados")
		need(ev.IntegratedHead, "stale_head", "verificações integradas não valem para a cabeça publicada")
		need(ev.GitHubConfirms, "no_github_confirmation", "GitHub não confirmou a criação/atualização da PR")
	case t.Code == Concluido:
		need(ev.MergeConfirmed, "no_merge", "merge não confirmado na PR vinculada")
	}
	if len(ps) > 0 {
		return &Rejection{f.Code, t.Code, ps}
	}
	return nil
}

func sortedTracks(m map[Track]Stage) []Track {
	out := make([]Track, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// Aggregate deriva o status agregado da Issue principal pela etapa obrigatória mais atrasada,
// independentemente do avanço individual das trilhas. planned indica plano materializado.
func Aggregate(planned bool, tracks map[Track]Stage) (string, Track) {
	if !planned || len(tracks) == 0 {
		return Planejamento, ""
	}
	min, who := StageReviewed+1, Track("")
	for _, tr := range sortedTracks(tracks) {
		if tracks[tr] < min {
			min, who = tracks[tr], tr
		}
	}
	switch {
	case min < StageImplemented:
		return Implementacao, who
	default:
		// todas as implementações prontas: o agregado está em REVIEW; com todas revisadas, pronto p/ Draft PR.
		return Review, who
	}
}
