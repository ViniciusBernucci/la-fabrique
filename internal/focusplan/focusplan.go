// Package focusplan valida e materializa o plano focalizado de uma correção (AJ05-04): a partir da
// rodada de revisão e da cabeça atual da PR original, só as subtarefas, contratos e verificações
// afetados são gerados. Cada subtarefa aponta para os comentários que trata; referências ainda válidas
// do plano anterior são retidas; o risco é reavaliado sobre os arquivos; e uma mudança material da cabeça
// da PR antes do despacho exige reconciliação.
package focusplan

import (
	"context"
	"errors"
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/ViniciusBernucci/le-fabrique/internal/contract"
	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
	"github.com/ViniciusBernucci/le-fabrique/internal/planning"
	"github.com/ViniciusBernucci/le-fabrique/internal/risk"
	"github.com/ViniciusBernucci/le-fabrique/internal/techplan"
)

// ErrNeedsReconciliation: a cabeça da PR mudou de forma material depois da rodada.
var ErrNeedsReconciliation = errors.New("a cabeça da PR mudou de forma material desde a rodada: reconcilie antes de despachar")

// Round é a rodada que originou a correção.
type Round struct {
	Key     string
	HeadSHA string   // cabeça observada quando a rodada foi fechada
	Refs    []string // comentários da rodada (comment:<id>, line_comment:<id>, review:<id>)
}

// Input é o pedido de plano focalizado.
type Input struct {
	Round             Round
	CurrentHead       string
	ChangedSinceRound []string // arquivos alterados na PR entre a cabeça da rodada e a atual
	Previous          planning.Plan
	Proposed          planning.Plan
	Policy            risk.Policy
}

// Reconcile descreve a divergência de cabeça.
type Reconcile struct {
	Required bool     `json:"required"`
	Files    []string `json:"files,omitempty"`
	From     string   `json:"from"`
	To       string   `json:"to"`
}

// Result é o plano focalizado pronto para materializar.
type Result struct {
	Plan        planning.Plan `json:"plan"`
	Retained    []string      `json:"retained"`    // tarefas do plano anterior ainda válidas
	Invalidated []string      `json:"invalidated"` // tarefas do plano anterior que tocam o que mudou
	Reconcile   Reconcile     `json:"reconcile"`
}

func within(file string, paths []string) bool {
	for _, p := range paths {
		p = strings.TrimSuffix(path.Clean(p), "/")
		if file == p || strings.HasPrefix(file, p+"/") || strings.HasPrefix(p, file+"/") {
			return true
		}
	}
	return false
}

func intersect(changed, paths []string) []string {
	var out []string
	for _, f := range changed {
		if within(f, paths) {
			out = append(out, f)
		}
	}
	sort.Strings(out)
	return out
}

// Prepare valida a proposta e deriva o resultado. Os problemas impedem a materialização.
func Prepare(in Input) (Result, []planning.Problem) {
	var ps []planning.Problem
	add := func(task, f, m string) { ps = append(ps, planning.Problem{Task: task, Field: f, Message: m}) }
	known := map[string]bool{}
	for _, r := range in.Round.Refs {
		known[r] = true
	}
	if len(known) == 0 {
		add("", "round", "a rodada não traz comentários para tratar")
	}
	ps = append(ps, techplan.Validate(in.Proposed)...)
	covered := map[string]bool{}
	adjusted := planning.Normalize(in.Proposed)
	var affectedPaths []string
	for i, t := range adjusted.Tasks {
		if len(t.Refs) == 0 {
			add(t.Key, "refs", "a subtarefa de correção deve apontar para os comentários da rodada que trata")
		}
		for _, r := range t.Refs {
			if !known[r] {
				add(t.Key, "refs", "referência fora da rodada: "+r)
				continue
			}
			covered[r] = true
		}
		// Risco reavaliado sobre os arquivos: só eleva.
		files := make([]risk.File, 0, len(t.Paths))
		for _, p := range t.Paths {
			files = append(files, risk.File{Path: p})
		}
		a := risk.Assess(in.Policy, t.Risk, files)
		adjusted.Tasks[i].Risk = a.Final
		affectedPaths = append(affectedPaths, t.Paths...)
	}
	var uncovered []string
	for r := range known {
		if !covered[r] {
			uncovered = append(uncovered, r)
		}
	}
	sort.Strings(uncovered)
	if len(uncovered) > 0 && len(adjusted.Tasks) > 0 {
		add("", "refs", "comentários da rodada sem tratamento: "+strings.Join(uncovered, ", "))
	}
	res := Result{Plan: adjusted, Retained: []string{}, Invalidated: []string{}}
	// Referências válidas do plano anterior: retidas se não tocam o que mudou na PR e não foram refeitas.
	proposed := map[string]bool{}
	for _, t := range adjusted.Tasks {
		proposed[t.Key] = true
	}
	for _, t := range planning.Normalize(in.Previous).Tasks {
		if proposed[t.Key] {
			continue
		}
		if len(intersect(in.ChangedSinceRound, t.Paths)) > 0 {
			res.Invalidated = append(res.Invalidated, t.Key)
		} else {
			res.Retained = append(res.Retained, t.Key)
		}
	}
	// Cabeça: mudança material = alteração em arquivo que a correção vai tocar.
	res.Reconcile = Reconcile{From: in.Round.HeadSHA, To: in.CurrentHead}
	if in.CurrentHead != "" && in.CurrentHead != in.Round.HeadSHA {
		if files := intersect(in.ChangedSinceRound, affectedPaths); len(files) > 0 {
			res.Reconcile.Required, res.Reconcile.Files = true, files
		}
	}
	return res, ps
}

// DB é o que o serviço precisa do PostgreSQL.
type DB interface {
	persistence.DBTX
	persistence.TxStarter
}

// Service materializa o plano focalizado.
type Service struct {
	DB        DB
	Tech      *techplan.Service
	Contracts *contract.Store
}

// ContractBase fornece os dados determinísticos de cada subtarefa (preenchidos pelo backend).
type ContractBase func(taskKey string) contract.Deterministic

// Accept valida, materializa as subtarefas de correção (classe `correction`) e publica um contrato por
// subtarefa. Se a cabeça mudou de forma material, devolve ErrNeedsReconciliation sem gravar nada.
func (s *Service) Accept(ctx context.Context, projectID, featureID string, in Input, base ContractBase, refs contract.References) (Result, error) {
	res, ps := Prepare(in)
	if len(ps) > 0 {
		return res, &techplan.RejectedError{Problems: ps}
	}
	if res.Reconcile.Required {
		return res, fmt.Errorf("%w (%s)", ErrNeedsReconciliation, strings.Join(res.Reconcile.Files, ", "))
	}
	out, err := s.Tech.Accept(ctx, projectID, featureID, res.Plan)
	if err != nil {
		return res, err
	}
	if _, err := s.DB.Exec(ctx, `UPDATE subtasks SET work_class = 'correction' WHERE feature_id::text = $1 AND plan_version = $2`, featureID, out.PlanVersion); err != nil {
		return res, persistence.MapError(err)
	}
	graph := contract.Graph{}
	for _, t := range res.Plan.Tasks {
		graph[t.Key] = t.DependsOn
	}
	refs.PlanVersion = out.PlanVersion
	for _, t := range res.Plan.Tasks {
		d := base(t.Key)
		d.SubtaskID = t.Key
		var criteria []string
		for _, c := range t.Criteria {
			criteria = append(criteria, c.Text)
		}
		prop := contract.Proposal{Criteria: criteria, Risk: t.Risk, Dependencies: t.DependsOn, SharedResources: t.SharedResources, RelevantFiles: t.Paths}
		if _, _, err := s.Contracts.Publish(ctx, d, prop, t.Risk, refs, graph); err != nil {
			return res, err
		}
	}
	return res, nil
}
