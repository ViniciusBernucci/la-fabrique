package factory

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/ViniciusBernucci/le-fabrique/internal/agents"
	"github.com/ViniciusBernucci/le-fabrique/internal/execution"
	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
	"github.com/ViniciusBernucci/le-fabrique/internal/planning"
	"github.com/ViniciusBernucci/le-fabrique/internal/runner"
	"github.com/ViniciusBernucci/le-fabrique/internal/techplan"
	"github.com/ViniciusBernucci/le-fabrique/internal/workflows"
)

// Canal da fábrica: pasta, dentro do workspace do agente, onde ele deixa entregas estruturadas (o plano do Tech
// Lead). Fica fora do patch: nunca entra no commit.
const (
	factoryDir   = ".fabrica"
	planFile     = factoryDir + "/plano.json"
	plannerSkill = "lf-tech-lead"
)

// isPlanner diz se o agente é um planejador (o Tech Lead): é quem produz o plano que vira subtarefas.
func isPlanner(eff agents.Effective) bool {
	for _, s := range eff.Skills {
		if s == plannerSkill {
			return true
		}
	}
	return false
}

// plannerFormat é o formato de saída que a fábrica exige do Tech Lead (a skill manda usar o schema do workflow).
const plannerFormat = `## FORMATO DE SAÍDA OBRIGATÓRIO DO PLANO
A fábrica só consome o plano como arquivo estruturado. Além do resumo em texto na sua resposta, GRAVE o arquivo ` + "`" + planFile + "`" + ` (JSON válido, sem comentários) neste formato:

{
  "summary": "resumo do plano em 1 a 3 frases",
  "tasks": [
    {
      "key": "backend-saude",
      "title": "título curto",
      "description": "o que fazer, com o escopo e o que fica de fora",
      "track": "BACKEND",
      "risk": "low",
      "order": 1,
      "depends_on": [],
      "paths": ["caminho/relativo/do/arquivo"],
      "criteria": [
        {"text": "descrição verificável do critério", "check": "file", "path": "caminho/relativo/do/arquivo"}
      ]
    }
  ]
}

Regras (a fábrica recusa o plano que as violar e a etapa falha):
- "key": minúsculas, números, "-" ou "_", até 40 caracteres, única. "depends_on" lista chaves existentes, sem ciclos.
- "track": "FRONTEND" para interface (páginas, componentes, estilos; pastas apps/web, frontend, arquivos .tsx, .css, .html) ou "BACKEND" para o resto. Divida o trabalho misto em uma tarefa FRONTEND e outra BACKEND; só marque "indivisible": true (com a trilha predominante) se for impossível dividir.
- "risk": "low", "medium" ou "high".
- "paths": caminhos relativos ao repositório (sem "/" inicial, sem "..") dos arquivos que a tarefa cria ou altera.
- cada tarefa precisa de ao menos um critério com "check" igual a: "file" (exige "path"), "command" (exige "command", uma linha), "test" (exige "target") ou "review" (exige "evidence" observável, mínimo de 15 caracteres). O texto do critério tem no mínimo 8 caracteres.
- no máximo 50 tarefas; planeje só o necessário para a tarefa da fábrica.
Não crie nenhum outro arquivo. O arquivo ` + "`" + planFile + "`" + ` é lido pela fábrica e não entra no commit.
`

// trackOfNode deduz a trilha de um nó de agente pelo texto do nó e do agente ("" = vê todas as trilhas).
func trackOfNode(n workflows.Node, eff agents.Effective) string {
	text := strings.ToUpper(propStr(n, "prompt") + " " + eff.AgentName + " " + n.ID)
	front, back := strings.Contains(text, "FRONTEND"), strings.Contains(text, "BACKEND")
	switch {
	case front && !back:
		return "FRONTEND"
	case back && !front:
		return "BACKEND"
	}
	return ""
}

// planContext monta, para um agente que NÃO é o planejador, o plano técnico vigente (as subtarefas da sua trilha) e
// o que as etapas anteriores relataram. Vazio se ainda não há nada.
func (e *Executor) planContext(ctx context.Context, info runner.Info, n workflows.Node, eff agents.Effective, vars map[string]any) string {
	var b strings.Builder
	var featureID string
	_ = e.DB.QueryRow(ctx, `SELECT feature_id::text FROM executions WHERE id = $1`, info.Run.ExecutionID).Scan(&featureID)
	if featureID != "" {
		st, err := (&planning.Store{DB: e.DB}).Current(ctx, info.Run.ProjectID, featureID)
		if err == nil && len(st.Plan.Tasks) > 0 {
			track := trackOfNode(n, eff)
			fmt.Fprintf(&b, "## Plano técnico aprovado (versão %d)\n%s\n", st.Version, strings.TrimSpace(st.Plan.Summary))
			if track != "" {
				fmt.Fprintf(&b, "Você atua na trilha %s: execute APENAS as subtarefas dela, abaixo.\n", track)
			}
			shown := 0
			for _, t := range st.Plan.Tasks {
				if track != "" && t.Track != track {
					continue
				}
				shown++
				fmt.Fprintf(&b, "\n### Subtarefa `%s` — %s (trilha %s, risco %s)\n", t.Key, t.Title, t.Track, t.Risk)
				if d := strings.TrimSpace(t.Description); d != "" {
					fmt.Fprintf(&b, "%s\n", d)
				}
				if len(t.Paths) > 0 {
					fmt.Fprintf(&b, "Arquivos previstos: %s\n", strings.Join(t.Paths, ", "))
				}
				if len(t.DependsOn) > 0 {
					fmt.Fprintf(&b, "Depende de: %s\n", strings.Join(t.DependsOn, ", "))
				}
				b.WriteString("Critérios de aceite:\n")
				for _, c := range t.Criteria {
					fmt.Fprintf(&b, "- %s [%s]\n", c.Text, criterionTarget(c))
				}
			}
			if shown == 0 {
				b.WriteString("\n(o plano não tem subtarefas para a sua trilha: apenas relate isso)\n")
			}
			b.WriteString("\n")
		}
	}
	// o que as etapas anteriores de agente relataram (resumo final de cada uma), em ordem estável
	type prev struct{ node, text string }
	var prevs []prev
	for k, v := range vars {
		m, ok := v.(map[string]any)
		if !ok || k == workflows.VarKey(n.ID) {
			continue
		}
		if text, _ := m["resumo"].(string); strings.TrimSpace(text) != "" {
			prevs = append(prevs, prev{k, text})
		}
	}
	sort.Slice(prevs, func(i, j int) bool { return prevs[i].node < prevs[j].node })
	if len(prevs) > 0 {
		b.WriteString("## O que as etapas anteriores relataram\n")
		for _, p := range prevs {
			fmt.Fprintf(&b, "### %s\n%s\n\n", p.node, tail(p.text, 2500))
		}
	}
	return b.String()
}

func criterionTarget(c planning.Criterion) string {
	switch c.Check {
	case planning.CheckFile:
		return "arquivo " + c.Path
	case planning.CheckCommand:
		return "comando " + c.Command
	case planning.CheckTest:
		return "teste " + c.Target
	}
	return "revisão: " + c.Evidence
}

// harvestPlan lê o plano que o Tech Lead gravou, valida (techplan), guarda a versão e materializa as subtarefas.
// Devolve as saídas do nó ou o motivo, em texto, da recusa.
func (e *Executor) harvestPlan(ctx context.Context, info runner.Info, n workflows.Node, sb *execution.Sandbox) (map[string]any, error) {
	r, err := sb.Exec(ctx, execution.ExecSpec{Cmd: []string{"sh", "-c", "cat " + planFile}})
	if err != nil {
		return nil, err
	}
	if r.ExitCode != 0 || strings.TrimSpace(r.Stdout) == "" {
		return nil, fmt.Errorf("o Tech Lead não gravou o plano em %s (a fábrica só consome o plano como arquivo estruturado)", planFile)
	}
	var plan planning.Plan
	dec := json.NewDecoder(bytes.NewReader([]byte(r.Stdout)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&plan); err != nil {
		return nil, fmt.Errorf("o plano em %s não é um JSON no formato exigido: %v", planFile, err)
	}
	var featureID string
	if err := e.DB.QueryRow(ctx, `SELECT feature_id::text FROM executions WHERE id = $1`, info.Run.ExecutionID).Scan(&featureID); err != nil {
		return nil, persistence.MapError(err)
	}
	svc := &techplan.Service{DB: e.DB, Plans: &planning.Store{DB: e.DB}}
	res, err := svc.Accept(ctx, info.Run.ProjectID, featureID, plan)
	var rej *techplan.RejectedError
	if errors.As(err, &rej) {
		msgs := make([]string, 0, len(rej.Problems))
		for _, p := range rej.Problems {
			msgs = append(msgs, p.String())
		}
		e.diag(info, n.ID, "plan.rejected", map[string]any{"problems": strings.Join(msgs, " | ")})
		return nil, fmt.Errorf("plano recusado pela fábrica: %s", tail(strings.Join(msgs, "; "), 700))
	}
	if err != nil {
		return nil, err
	}
	front, back := 0, 0
	for _, s := range res.Subtasks {
		if s.Track == "FRONTEND" {
			front++
		} else {
			back++
		}
	}
	e.diag(info, n.ID, "plan.accepted", map[string]any{"version": res.PlanVersion, "unchanged": res.Unchanged, "subtarefas": len(res.Subtasks), "frontend": front, "backend": back})
	info.Emit(runner.EventNode, map[string]any{"node": n.ID, "kind": "note",
		"text": fmt.Sprintf("plano v%d aceito: %d subtarefa(s) (%d frontend, %d backend) materializada(s) na fábrica", res.PlanVersion, len(res.Subtasks), front, back)})
	return map[string]any{"plano_versao": res.PlanVersion, "subtarefas": len(res.Subtasks), "subtarefas_frontend": front, "subtarefas_backend": back, "plano_resumo": plan.Summary}, nil
}
