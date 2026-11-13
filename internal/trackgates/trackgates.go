// Package trackgates liga os resultados por trilha e os testes integrados à publicação da Draft PR
// (AJ08-05). Um Front pode concluir o review cedo, mas falta de resultado no Back, nos checks
// integrados ou no review visual impede a publicação; toda evidência vale só para a revisão a que se
// refere e a consolidação invalida o que ela alterou.
package trackgates

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/ViniciusBernucci/le-fabrique/internal/flow"
	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
	"github.com/ViniciusBernucci/le-fabrique/internal/qualitypolicy"
)

// DB é o que o armazenamento precisa do PostgreSQL.
type DB interface{ persistence.DBTX }

// Artifact referencia uma evidência visual (nunca o conteúdo bruto sem sanitização).
type Artifact struct {
	Kind   string `json:"kind"` // screenshot | log
	Name   string `json:"name"`
	SHA256 string `json:"sha256"`
}

// Store persiste os resultados.
type Store struct{ DB DB }

// RecordTrack grava o resultado da trilha para a revisão (substitui o da mesma revisão).
func (s *Store) RecordTrack(ctx context.Context, featureID string, track flow.Track, revision string, req qualitypolicy.Requirement, ev qualitypolicy.Evidence, artifacts []Artifact) error {
	if revision == "" || (track != flow.Frontend && track != flow.Backend) {
		return fmt.Errorf("%w: trilha e revisão são obrigatórias", persistence.ErrInvalidInput)
	}
	rj, _ := json.Marshal(req)
	ej, _ := json.Marshal(ev)
	if artifacts == nil {
		artifacts = []Artifact{}
	}
	aj, _ := json.Marshal(artifacts)
	_, err := s.DB.Exec(ctx, `INSERT INTO track_results (feature_id, track, revision, requirement, evidence, artifacts) VALUES ($1,$2,$3,$4,$5,$6)
		ON CONFLICT (feature_id, track, revision) DO UPDATE SET requirement = EXCLUDED.requirement, evidence = EXCLUDED.evidence, artifacts = EXCLUDED.artifacts, created_at = now()`,
		featureID, string(track), revision, string(rj), string(ej), string(aj))
	return persistence.MapError(err)
}

// RecordIntegrated grava os testes integrados executados na cabeça consolidada.
func (s *Store) RecordIntegrated(ctx context.Context, featureID, head string, tests qualitypolicy.TestRun) error {
	if head == "" {
		return fmt.Errorf("%w: cabeça obrigatória", persistence.ErrInvalidInput)
	}
	tests.Revision = head
	tj, _ := json.Marshal(tests)
	_, err := s.DB.Exec(ctx, `INSERT INTO integrated_results (feature_id, head_sha, tests) VALUES ($1,$2,$3)
		ON CONFLICT (feature_id, head_sha) DO UPDATE SET tests = EXCLUDED.tests, created_at = now()`, featureID, head, string(tj))
	return persistence.MapError(err)
}

// Input é o pedido de publicação.
type Input struct {
	FeatureID string
	Head      string              // cabeça candidata que será enviada ao GitHub
	Required  []flow.Track        // trilhas obrigatórias (do plano)
	Changed   map[flow.Track]bool // trilhas cujas SHAs a consolidação alterou: o resultado precisa ser da cabeça
}

// Result é a decisão.
type Result struct {
	Allowed  bool     `json:"allowed"`
	Problems []string `json:"problems"`
}

// Gate decide se a cabeça pode ser publicada como Draft PR.
func (s *Store) Gate(ctx context.Context, in Input) (Result, error) {
	res := Result{Problems: []string{}}
	add := func(f string, a ...any) { res.Problems = append(res.Problems, fmt.Sprintf(f, a...)) }
	if in.Head == "" || len(in.Required) == 0 {
		return res, fmt.Errorf("%w: cabeça e trilhas obrigatórias", persistence.ErrInvalidInput)
	}
	req := append([]flow.Track{}, in.Required...)
	sort.Slice(req, func(i, j int) bool { return req[i] < req[j] })
	for _, tr := range req {
		var rev string
		var rj, ej []byte
		err := s.DB.QueryRow(ctx, `SELECT revision, requirement, evidence FROM track_results WHERE feature_id::text=$1 AND track=$2 ORDER BY created_at DESC LIMIT 1`, in.FeatureID, string(tr)).Scan(&rev, &rj, &ej)
		if err != nil {
			add("trilha %s sem resultado de testes e reviews: a publicação aguarda", tr)
			continue
		}
		if in.Changed[tr] && rev != in.Head {
			add("trilha %s: a consolidação alterou os commits e o resultado (%s) não é da cabeça publicada (%s): revalide", tr, short(rev), short(in.Head))
			continue
		}
		var r qualitypolicy.Requirement
		var e qualitypolicy.Evidence
		_ = json.Unmarshal(rj, &r)
		_ = json.Unmarshal(ej, &e)
		for _, p := range qualitypolicy.Verify(r, e, rev) {
			add("trilha %s: %s", tr, p.String())
		}
	}
	var tj []byte
	if err := s.DB.QueryRow(ctx, `SELECT tests FROM integrated_results WHERE feature_id::text=$1 AND head_sha=$2`, in.FeatureID, in.Head).Scan(&tj); err != nil {
		add("os testes integrados ainda não rodaram na cabeça %s", short(in.Head))
	} else {
		var t qualitypolicy.TestRun
		_ = json.Unmarshal(tj, &t)
		switch {
		case len(t.Commands) == 0:
			add("testes integrados sem os comandos executados")
		case !t.Passed:
			add("testes integrados falharam na cabeça %s: %s", short(in.Head), strings.Join(t.Errors, "; "))
		}
	}
	res.Allowed = len(res.Problems) == 0
	return res, nil
}

func short(s string) string {
	if len(s) > 8 {
		return s[:8]
	}
	return s
}
