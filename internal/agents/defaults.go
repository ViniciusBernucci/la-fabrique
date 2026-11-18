package agents

import (
	"context"
	"errors"

	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
	"github.com/ViniciusBernucci/le-fabrique/internal/workflows"
)

const commonRules = `

Regras do ofício na La fabrique:
- Trabalhe em português e preserve a grafia dos identificadores recebidos no contrato de execução.
- Use o cliente oficial e a worktree atribuída; um único writer por workspace.
- Não faça merge, push, deploy, mudança de plano ou gasto, nem acesse credenciais sem autorização explícita.
- Nunca grave segredos em arquivos, logs, prompts ou testes.
- Relate resultados reais (PASS, FAIL ou NOT_RUN); a fala do modelo e o código de saída não provam sucesso.`

// Ferramentas nativas dos agentes padrão: só leitura e busca (planejar, documentar) ou também edição e execução de
// comandos no terminal isolado (implementar e revisar). O terminal não tem rota para a internet fora do proxy de saída.
var (
	toolsRead = []string{"read_file", "list_dir", "search", "git_status", "git_diff"}
	toolsWork = []string{"read_file", "write_file", "edit_file", "list_dir", "search", "run_command", "git_status", "git_diff"}
)

// defaultAgents é o catálogo inicial de agentes (ADR-0068): um por papel do fluxo padrão, cada um já
// com a skill padrão correspondente. Provedor Claude, modelo e conta vazios (usam o padrão da conta
// conectada); o administrador troca o que quiser na tela de Agentes.
var defaultAgents = []Input{
	{
		Name: workflows.AgentPRD, Description: "Entrevista requisitos e produz PRD, especificação técnica e tarefas rastreáveis.",
		Provider: ProviderClaude, Skills: []string{"escrever-prd-e-specs"}, Tools: toolsRead, TimeoutMinutes: 60,
		Instructions: "Você é o redator de PRD e especificações da La fabrique. Use a skill escrever-prd-e-specs para transformar uma ideia ou mudança em PRD, especificação técnica, contratos, critérios de aceite e testes rastreáveis." + commonRules,
	},
	{
		Name: workflows.AgentPlanner, Description: "Planeja a feature ou a correção: subtarefas, critérios verificáveis, dependências, riscos e testes.",
		Provider: ProviderClaude, Skills: []string{"lf-tech-lead"}, Tools: toolsRead, TimeoutMinutes: 60,
		Instructions: "Você é o Tech Lead da La fabrique. Use a skill lf-tech-lead para propor o plano técnico a partir do PRD validado ou da correção liberada, no formato de saída que o workflow fornecer. Proponha; a Sentinela valida e materializa as subtarefas." + commonRules,
	},
	{
		Name: workflows.AgentImplFront, Description: "Implementa as subtarefas FRONTEND e entrega código, testes reais e evidências.",
		Provider: ProviderClaude, Skills: []string{"lf-implementar-tarefa"}, Tools: toolsWork, TimeoutMinutes: 120,
		Instructions: "Você implementa as subtarefas FRONTEND despachadas pela La fabrique. Use a skill lf-implementar-tarefa: entregue código, testes reais e evidências à Sentinela, que controla o estado da Issue, a revisão e os efeitos externos. Siga o design estabelecido do projeto." + commonRules,
	},
	{
		Name: workflows.AgentImplBack, Description: "Implementa as subtarefas BACKEND e entrega código, testes reais e evidências.",
		Provider: ProviderClaude, Skills: []string{"lf-implementar-tarefa"}, Tools: toolsWork, TimeoutMinutes: 120,
		Instructions: "Você implementa as subtarefas BACKEND despachadas pela La fabrique. Use a skill lf-implementar-tarefa: entregue código, testes reais (incluindo os de integração quando tocar o banco) e evidências à Sentinela, que controla o estado da Issue, a revisão e os efeitos externos." + commonRules,
	},
	{
		Name: workflows.AgentReviewFront, Description: "Revisa tarefas FRONTEND em navegador isolado e registra evidências.",
		Provider: ProviderClaude, Skills: []string{"lf-revisar-frontend"}, Tools: toolsWork, TimeoutMinutes: 90,
		Instructions: "Você revisa tarefas FRONTEND da La fabrique em sessão independente, sem editar o que está sendo revisado além das correções autorizadas. Use a skill lf-revisar-frontend com o navegador isolado atribuído à execução, reproduza os critérios de aceite e registre evidências." + commonRules,
	},
	{
		Name: workflows.AgentReviewBack, Description: "Revisa tarefas BACKEND de alto risco contra o diff e os critérios de aceite.",
		Provider: ProviderClaude, Skills: []string{"lf-revisar-backend"}, Tools: toolsWork, TimeoutMinutes: 90,
		Instructions: "Você revisa tarefas BACKEND de alto risco da La fabrique em sessão independente. Use a skill lf-revisar-backend: avalie o diff e os critérios de aceite, corrija só os achados autorizados e revalide com testes reais na revisão candidata." + commonRules,
	},
	{
		Name: workflows.AgentVisualDiscovery, Description: "Descobre como subir a aplicação do projeto para o review visual e propõe a configuração.",
		Provider: ProviderClaude, Skills: []string{"lf-descobrir-ambiente-visual"}, Tools: toolsRead, TimeoutMinutes: 45,
		Instructions: "Você descobre como iniciar a aplicação de um projeto para o review visual. Use a skill lf-descobrir-ambiente-visual e produza uma proposta editável com serviços, prontidão, acesso via navegador, dados de teste e fontes." + commonRules,
	},
}

// DefaultNames lista os nomes dos agentes padrão (para testes e logs).
func DefaultNames() []string {
	out := make([]string, len(defaultAgents))
	for i, a := range defaultAgents {
		out[i] = a.Name
	}
	return out
}

// InstallDefaults cadastra os agentes padrão que ainda não existem. Idempotente e conservador: um agente
// que já existe com o mesmo nome (mesmo editado) não é tocado nem recriado.
func InstallDefaults(ctx context.Context, db persistence.DBTX) (installed []string, err error) {
	for _, in := range defaultAgents {
		in := in
		if cur, err := GetByName(ctx, db, in.Name); err == nil {
			// agente padrão que ninguém editou (versão 1) e ficou sem ferramentas: recebe as do papel
			if cur.Version == 1 && len(cur.Tools) == 0 && len(in.Tools) > 0 {
				up := Input{Name: cur.Name, Description: cur.Description, Instructions: cur.Instructions, Provider: cur.Provider,
					ConnectionID: cur.ConnectionID, Model: cur.Model, Skills: cur.Skills, Tools: in.Tools, Fallback: cur.Fallback, TimeoutMinutes: cur.TimeoutMinutes}
				if _, uerr := Update(ctx, db, cur.ID, cur.Version, up); uerr == nil {
					installed = append(installed, in.Name+" (ferramentas)")
				}
			}
			continue
		} else if !errors.Is(err, persistence.ErrNotFound) {
			return installed, err
		}
		if _, err := Create(ctx, db, in); err != nil {
			if errors.Is(err, persistence.ErrDuplicate) {
				continue
			}
			return installed, err
		}
		installed = append(installed, in.Name)
	}
	return installed, nil
}
