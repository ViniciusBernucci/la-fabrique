package releases

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/ViniciusBernucci/le-fabrique/internal/agents"
	"github.com/ViniciusBernucci/le-fabrique/internal/models"
	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
	"github.com/ViniciusBernucci/le-fabrique/internal/security"
	"github.com/ViniciusBernucci/le-fabrique/internal/skills"
	"github.com/ViniciusBernucci/le-fabrique/internal/tools"
	"github.com/ViniciusBernucci/le-fabrique/internal/workflows"
)

// Erros.
var (
	ErrNotPublished = errors.New("o workflow não tem versão publicada: rascunho não executa")
	ErrRevalidation = errors.New("a versão publicada não passou na revalidação em tempo de execução")
)

// Problem é um problema encontrado ao publicar ou revalidar.
type Problem struct {
	Node    string `json:"node,omitempty"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

// PublishError reúne os problemas que impediram a publicação.
type PublishError struct{ Problems []Problem }

func (e *PublishError) Error() string {
	return fmt.Sprintf("o workflow tem %d problema(s) e não pode ser publicado", len(e.Problems))
}
func (e *PublishError) Unwrap() error { return persistence.ErrInvalidInput }

// RevalidationError reúne os problemas da revalidação.
type RevalidationError struct{ Problems []Problem }

func (e *RevalidationError) Error() string {
	return fmt.Sprintf("%v: %d problema(s)", ErrRevalidation, len(e.Problems))
}
func (e *RevalidationError) Unwrap() error { return ErrRevalidation }

// DB é o que o pacote precisa do PostgreSQL.
type DB interface {
	persistence.DBTX
	persistence.TxStarter
}

// SkillRef e ToolRef congelam o que o nó usa.
type SkillRef struct {
	Name        string `json:"name"`
	ID          string `json:"id"`
	Version     int    `json:"version"`
	ContentHash string `json:"content_hash"`
}

// ToolRef congela uma ferramenta usada por um agente.
type ToolRef struct {
	Name    string `json:"name"`
	Kind    string `json:"kind"`
	Version int    `json:"version"`
}

// NodeSnapshot é a configuração efetiva congelada de um nó (sem segredos: a conta
// aparece só como referência).
type NodeSnapshot struct {
	NodeID     string            `json:"node_id"`
	Type       string            `json:"type"`
	Agent      *agents.Effective `json:"agent,omitempty"`
	ModelCheck *models.Verdict   `json:"model_check,omitempty"`
	Skill      *SkillRef         `json:"skill,omitempty"`
	Tools      []ToolRef         `json:"tools,omitempty"`
}

// Version é uma versão publicada.
type Version struct {
	ID           string            `json:"id"`
	WorkflowID   string            `json:"workflow_id"`
	Number       int               `json:"number"`
	DraftVersion int               `json:"draft_version"`
	Name         string            `json:"name"`
	Kind         string            `json:"kind"`
	Description  string            `json:"description"`
	Graph        workflows.Graph   `json:"graph"`
	FlatGraph    workflows.Graph   `json:"flat_graph"`
	Subflows     map[string]string `json:"subflows"` // workflow_id → versão fixada
	Effective    []NodeSnapshot    `json:"effective"`
	ContentHash  string            `json:"content_hash"`
	PublishedAt  time.Time         `json:"published_at"`
	Unchanged    bool              `json:"unchanged,omitempty"`
}

const verCols = `id, workflow_id, number, draft_version, name, kind, description, graph, flat_graph, subflows, effective, content_hash, published_at`

func scanVersion(row interface{ Scan(...any) error }) (Version, error) {
	var v Version
	var graph, flat, sub, eff []byte
	if err := row.Scan(&v.ID, &v.WorkflowID, &v.Number, &v.DraftVersion, &v.Name, &v.Kind, &v.Description, &graph, &flat, &sub, &eff, &v.ContentHash, &v.PublishedAt); err != nil {
		return v, persistence.MapError(err)
	}
	if json.Unmarshal(graph, &v.Graph) != nil || json.Unmarshal(flat, &v.FlatGraph) != nil || json.Unmarshal(sub, &v.Subflows) != nil || json.Unmarshal(eff, &v.Effective) != nil {
		return v, errors.New("versão publicada ilegível")
	}
	return v, nil
}

// Get lê uma versão publicada.
func Get(ctx context.Context, db persistence.DBTX, id string) (Version, error) {
	return scanVersion(db.QueryRow(ctx, `SELECT `+verCols+` FROM workflow_versions WHERE id = $1`, id))
}

// List devolve as versões de um workflow, da mais nova para a mais antiga (sem os grafos).
func List(ctx context.Context, db persistence.DBTX, workflowID string) ([]Version, error) {
	rows, err := db.Query(ctx, `SELECT `+verCols+` FROM workflow_versions WHERE workflow_id = $1 ORDER BY number DESC`, workflowID)
	if err != nil {
		return nil, persistence.MapError(err)
	}
	defer rows.Close()
	out := []Version{}
	for rows.Next() {
		v, err := scanVersion(rows)
		if err != nil {
			return nil, err
		}
		v.Graph, v.FlatGraph, v.Effective = workflows.Graph{}, workflows.Graph{}, nil
		out = append(out, v)
	}
	return out, persistence.MapError(rows.Err())
}

// Published devolve a versão publicada vigente do workflow (ErrNotPublished se não houver).
func Published(ctx context.Context, db persistence.DBTX, workflowID string) (Version, error) {
	var vid *string
	err := db.QueryRow(ctx, `SELECT published_version_id::text FROM workflows WHERE id = $1`, workflowID).Scan(&vid)
	if err != nil {
		return Version{}, persistence.MapError(err)
	}
	if vid == nil {
		return Version{}, ErrNotPublished
	}
	return Get(ctx, db, *vid)
}

// Publisher publica workflows.
type Publisher struct {
	DB       DB
	Redactor *security.Redactor
}

func canonicalHash(v any) string {
	b, _ := json.Marshal(v)
	var generic any
	_ = json.Unmarshal(b, &generic)
	c, _ := json.Marshal(generic)
	h := sha256.Sum256(c)
	return hex.EncodeToString(h[:])
}

// snapshot resolve e valida a configuração efetiva de cada nó do grafo expandido.
func snapshot(ctx context.Context, db persistence.DBTX, flat workflows.Graph) ([]NodeSnapshot, []Problem) {
	var snaps []NodeSnapshot
	var problems []Problem
	reg := tools.NewRegistry(func(c context.Context, name string) (tools.Tool, error) { return tools.Get(c, db, name) })
	for _, n := range flat.Nodes {
		switch n.Type {
		case workflows.TAgent:
			eff, err := agents.EffectiveForNode(ctx, db, n)
			if err != nil {
				code := "agent_invalid"
				switch {
				case errors.Is(err, agents.ErrAgentNotFound):
					code = "agent_not_found"
				case errors.Is(err, agents.ErrConnection):
					code = "connection_incompatible"
				case errors.Is(err, agents.ErrOverride):
					code = "invalid_override"
				}
				problems = append(problems, Problem{Node: n.ID, Code: code, Message: err.Error()})
				continue
			}
			sn := NodeSnapshot{NodeID: n.ID, Type: n.Type, Agent: &eff}
			req := models.Requirement{Provider: eff.Provider, Model: eff.Model, NeedsTools: len(eff.Tools) > 0}
			if eff.ConnectionID != nil {
				req.ConnectionID = *eff.ConnectionID
			}
			verdict, err := models.Check(ctx, db, req)
			if err != nil {
				problems = append(problems, Problem{Node: n.ID, Code: "model_check_failed", Message: err.Error()})
				continue
			}
			sn.ModelCheck = &verdict
			if !verdict.OK {
				problems = append(problems, Problem{Node: n.ID, Code: "model_incompatible", Message: strings.Join(verdict.Problems, " ")})
			}
			for _, name := range eff.Tools {
				if t, err := reg.Get(ctx, name); err != nil {
					problems = append(problems, Problem{Node: n.ID, Code: "tool_not_found", Message: fmt.Sprintf("ferramenta %q não existe no catálogo", name)})
				} else {
					sn.Tools = append(sn.Tools, ToolRef{Name: t.Name, Kind: t.Kind, Version: t.Version})
				}
			}
			snaps = append(snaps, sn)
		case workflows.TSkill:
			name, _ := n.Props["skill"].(string)
			sk, err := skills.GetByName(ctx, db, strings.TrimSpace(name))
			if err != nil {
				problems = append(problems, Problem{Node: n.ID, Code: "skill_not_found", Message: fmt.Sprintf("skill %q não existe no catálogo", name)})
				continue
			}
			snaps = append(snaps, NodeSnapshot{NodeID: n.ID, Type: n.Type, Skill: &SkillRef{Name: sk.Name, ID: sk.ID, Version: sk.Version, ContentHash: sk.ContentHash}})
		}
	}
	sort.Slice(snaps, func(i, j int) bool { return snaps[i].NodeID < snaps[j].NodeID })
	return snaps, problems
}

// Publish publica a versão salva do rascunho (expectedDraftVersion precisa ser a
// atual). Idempotente: publicar de novo o mesmo conteúdo devolve a versão vigente
// (Unchanged). Subfluxos precisam estar publicados e ficam fixados por versão.
func (p *Publisher) Publish(ctx context.Context, workflowID string, expectedDraftVersion int, adminID string) (Version, error) {
	var out Version
	err := persistence.WithTx(ctx, p.DB, func(tx pgx.Tx) error {
		var locked string
		if err := tx.QueryRow(ctx, `SELECT id::text FROM workflows WHERE id = $1 FOR UPDATE`, workflowID).Scan(&locked); err != nil {
			return persistence.MapError(err)
		}
		wf, err := workflows.Get(ctx, tx, workflowID)
		if err != nil {
			return err
		}
		if wf.Version != expectedDraftVersion {
			return persistence.ErrVersionConflict
		}
		var problems []Problem
		for _, is := range workflows.Validate(wf.Graph) {
			problems = append(problems, Problem{Node: is.Node, Code: is.Code, Message: is.Message})
		}
		if len(problems) > 0 {
			return &PublishError{Problems: problems}
		}
		pins := map[string]string{}
		resolve := func(id string) (workflows.Graph, error) {
			sub, err := workflows.Get(ctx, tx, id)
			if err != nil {
				return workflows.Graph{}, errors.New("o subfluxo não existe")
			}
			if sub.Kind != workflows.KindSubflow {
				return workflows.Graph{}, errors.New("o workflow referenciado não é um subfluxo")
			}
			ver, err := Published(ctx, tx, id)
			if err != nil {
				return workflows.Graph{}, errors.New("o subfluxo não está publicado (publique-o antes)")
			}
			pins[id] = ver.ID
			return ver.Graph, nil
		}
		flat, err := workflows.Flatten(wf.Graph, wf.ID, resolve)
		if err != nil {
			code := "subflow_invalid"
			switch {
			case errors.Is(err, workflows.ErrSubflowRecursive):
				code = "subflow_recursive"
			case errors.Is(err, workflows.ErrSubflowDepth):
				code = "subflow_depth"
			}
			return &PublishError{Problems: []Problem{{Code: code, Message: err.Error()}}}
		}
		for _, is := range workflows.ValidateFlat(flat) {
			problems = append(problems, Problem{Node: is.Node, Code: is.Code, Message: is.Message})
		}
		snaps, sp := snapshot(ctx, tx, flat)
		problems = append(problems, sp...)
		if len(problems) > 0 {
			return &PublishError{Problems: problems}
		}
		hash := canonicalHash(map[string]any{"flat": flat, "effective": snaps, "subflows": pins})
		if cur, err := Published(ctx, tx, workflowID); err == nil && cur.ContentHash == hash {
			out = cur
			out.Unchanged = true
			return nil
		}
		effJSON, _ := json.Marshal(snaps)
		if p.Redactor != nil {
			if clean := p.Redactor.Redact(string(effJSON)); json.Valid([]byte(clean)) {
				effJSON = []byte(clean)
			}
		}
		graphJSON, _ := json.Marshal(wf.Graph)
		flatJSON, _ := json.Marshal(flat)
		pinsJSON, _ := json.Marshal(pins)
		var admin any
		if adminID != "" {
			admin = adminID
		}
		var number int
		if err := tx.QueryRow(ctx, `SELECT COALESCE(max(number), 0) + 1 FROM workflow_versions WHERE workflow_id = $1`, workflowID).Scan(&number); err != nil {
			return persistence.MapError(err)
		}
		out, err = scanVersion(tx.QueryRow(ctx, `INSERT INTO workflow_versions (workflow_id, number, draft_version, name, kind, description, graph, flat_graph, subflows, effective, content_hash, published_by)
			VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb, $8::jsonb, $9::jsonb, $10::jsonb, $11, $12::uuid) RETURNING `+verCols,
			workflowID, number, wf.Version, wf.Name, wf.Kind, wf.Description, string(graphJSON), string(flatJSON), string(pinsJSON), string(effJSON), hash, admin))
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `UPDATE workflows SET published_version_id = $2 WHERE id = $1`, workflowID, out.ID)
		return persistence.MapError(err)
	})
	return out, err
}

// Unpublish tira a versão vigente de circulação (novas execuções não nascem).
// As versões e as execuções que já usam uma versão não mudam.
func (p *Publisher) Unpublish(ctx context.Context, workflowID string) error {
	tag, err := p.DB.Exec(ctx, `UPDATE workflows SET published_version_id = NULL WHERE id = $1`, workflowID)
	if err != nil {
		return persistence.MapError(err)
	}
	if tag.RowsAffected() == 0 {
		return persistence.ErrNotFound
	}
	return nil
}

// Revalidate confere em TEMPO DE EXECUÇÃO se a versão congelada ainda pode rodar:
// o agente existe, a conta efetiva está ativa e compatível (revogação vale), o
// modelo está ativo e compatível, e as ferramentas e skills existem. NÃO compara
// com o conteúdo atual do catálogo: o que foi publicado é o que roda.
func Revalidate(ctx context.Context, db persistence.DBTX, v Version) []Problem {
	var problems []Problem
	reg := tools.NewRegistry(func(c context.Context, name string) (tools.Tool, error) { return tools.Get(c, db, name) })
	for _, sn := range v.Effective {
		if sn.Agent != nil {
			a := sn.Agent
			if _, err := agents.Get(ctx, db, a.AgentID); err != nil {
				problems = append(problems, Problem{Node: sn.NodeID, Code: "agent_removed", Message: "o agente deixou de existir no catálogo"})
				continue
			}
			if a.ConnectionID != nil {
				if err := agents.CheckConnection(ctx, db, a.Provider, *a.ConnectionID); err != nil {
					code := "connection_incompatible"
					if strings.Contains(err.Error(), "revogada") {
						code = "connection_revoked"
					}
					problems = append(problems, Problem{Node: sn.NodeID, Code: code, Message: err.Error()})
				}
			}
			req := models.Requirement{Provider: a.Provider, Model: a.Model, NeedsTools: len(a.Tools) > 0}
			if a.ConnectionID != nil {
				req.ConnectionID = *a.ConnectionID
			}
			if verdict, err := models.Check(ctx, db, req); err != nil {
				problems = append(problems, Problem{Node: sn.NodeID, Code: "model_check_failed", Message: err.Error()})
			} else if !verdict.OK {
				problems = append(problems, Problem{Node: sn.NodeID, Code: "model_incompatible", Message: strings.Join(verdict.Problems, " ")})
			}
			for _, t := range sn.Tools {
				if _, err := reg.Get(ctx, t.Name); err != nil {
					problems = append(problems, Problem{Node: sn.NodeID, Code: "tool_removed", Message: fmt.Sprintf("a ferramenta %q deixou de existir", t.Name)})
				}
			}
		}
		if sn.Skill != nil {
			if _, err := skills.Get(ctx, db, sn.Skill.ID); err != nil {
				problems = append(problems, Problem{Node: sn.NodeID, Code: "skill_removed", Message: fmt.Sprintf("a skill %q deixou de existir", sn.Skill.Name)})
			}
		}
	}
	return problems
}

// StartExecution cria a execução de uma feature a partir da versão PUBLICADA do
// workflow, depois de revalidá-la. Rascunho (sem versão publicada) não executa. A
// execução fica presa à versão: publicar outra depois não a altera.
func StartExecution(ctx context.Context, db DB, projectID, featureID, workflowID string) (persistence.Execution, Version, error) {
	var ex persistence.Execution
	var ver Version
	err := persistence.WithTx(ctx, db, func(tx pgx.Tx) error {
		var err error
		if ver, err = Published(ctx, tx, workflowID); err != nil {
			return err
		}
		if ver.Kind != workflows.KindFeature {
			return fmt.Errorf("%w: só workflows do tipo feature têm execução", persistence.ErrInvalidInput)
		}
		if problems := Revalidate(ctx, tx, ver); len(problems) > 0 {
			return &RevalidationError{Problems: problems}
		}
		if ex, err = persistence.CreateExecution(ctx, tx, projectID, featureID); err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `UPDATE executions SET workflow_version_id = $2 WHERE id = $1`, ex.ID, ver.ID)
		return persistence.MapError(err)
	})
	return ex, ver, err
}

// ExecutionVersion devolve a versão publicada à qual a execução está presa.
func ExecutionVersion(ctx context.Context, db persistence.DBTX, executionID string) (Version, error) {
	var vid *string
	if err := db.QueryRow(ctx, `SELECT workflow_version_id::text FROM executions WHERE id = $1`, executionID).Scan(&vid); err != nil {
		return Version{}, persistence.MapError(err)
	}
	if vid == nil {
		return Version{}, ErrNotPublished
	}
	return Get(ctx, db, *vid)
}
