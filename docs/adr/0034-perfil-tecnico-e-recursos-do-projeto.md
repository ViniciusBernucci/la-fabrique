# ADR-0034 — Perfil técnico e recursos do projeto

- **Estado:** ACEITA · **Ticket:** [T05-03](../NEW%20MVP/03-BACKLOG.md#t05-03) · **Épico:** E05 · **Relacionadas:** [ADR-0006](0006-cadastro-de-projeto-em-rascunho.md), [ADR-0013](0013-catalogo-de-agentes-e-sobrescritas.md), [ADR-0027](0027-publicacao-e-congelamento-de-workflows.md), [ADR-0028](0028-selecao-de-fluxo-por-regras.md)

## Decisão
1. **Descoberta por inspeção segura, só sugere:** `projectres.Inspect` lê uma lista FIXA de manifestos (package.json e lockfiles, go.mod, Makefile, pyproject, requirements, Cargo, pom, Gemfile, Compose, `.env.example`) que o **painel envia** (nada é lido do disco do servidor, nada é executado, limite de 256 KiB por arquivo). Cada comando sugerido traz a origem (ex.: `package.json#scripts.test`); o gerenciador de pacotes vem do lockfile. Arquivos `.env` nunca são lidos; de `.env.example` saem **só os nomes** das variáveis. O resultado fica gravado como sugestão e **não altera o perfil**.
2. **Perfil escolhido:** comandos `install|build|test|lint|start` (uma linha, sem segredo — recusa pelo `Redactor`), serviços e **referências de segredo** `NOME → connection:<id>` de conexão **ativa**; valor literal, conexão inexistente ou revogada são recusados. Comando igual a uma sugestão guarda a origem dela; os demais são `manual`. Gravação com versão otimista.
3. **Recursos por associação explícita:** workflows, agentes, skills, ferramentas e modelos **globais**. Só agentes aceitam sobrescrita local (modelo, conta, instruções extras, ferramentas **só reduzidas**, tempo limite), validada pelas mesmas regras do nó (`agents.Resolve`). A sobrescrita vive na associação; **o recurso global nunca muda**, e outro projeto enxerga a base. A substituição é atômica (recusa não desfaz o anterior).
4. **Validação no relatório de ativação** (`projects.AddValidator`, sem tocar nos itens existentes): `resources.workflows` (associados e alvos da seleção precisam estar **publicados** e revalidar), `resources.agents` (conta ativa, modelo compatível com provedor/conta/ferramentas, modelo ativo, cadeia de fallback com contas ativas) e `resources.secrets`. Cada pendência diz o recurso e a ação. Sem associações, os itens passam (projeto novo continua vazio de atribuições).
5. **Aptidão para executar** (`Ready`, `GET /readiness`): projeto **ativo**, itens acima válidos e **fluxo padrão da seleção publicado**; devolve o alvo. A execução parte da versão publicada (`releases.StartExecution`).

## Limites
- Não há leitura do repositório no GitHub (T15-01/T05-02 parcial): os manifestos vêm do chamador. Sem tela do assistente.
- A sugestão de comandos é heurística e simples (Makefile por alvo, Compose por indentação); a execução dos comandos continua sujeita à política e ao perfil do sandbox (E11).
- Skills e ferramentas associadas são só registradas (existência), sem validação de compatibilidade além dos agentes.
