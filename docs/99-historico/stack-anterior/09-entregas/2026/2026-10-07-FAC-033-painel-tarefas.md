# FAC-033 — Painel Tarefas demonstrativo

Data: 2026-10-07, Europe/Berlin. IMPLEMENTADO como demonstração local; AWAITING_HUMAN. Branch developer, base 9439cde4ce52a09225c9476c19baa89319802523 + alterações anteriores preservadas. Sem commit/PR/deploy. Continuidade autorizada pelo usuário após checkpoint de writer.

## Objetivo, solução e funcionamento

[Ticket READY](../../08-desenvolvimento/tickets/FAC-033.md). Menu Tarefas abre quadro por nove etapas ou lista, com filtros e contagens. Tickets exibem código, projeto, agente responsável, prioridade, fase, prazo e dependências. Modal cria/edita título, especificação, critérios, responsável, prioridade, fase, etapa e dependências; histórico conserva alterações.

Simulação do onboarding cria um projeto e cinco agentes antes de gerar/distribuir nove tickets sintéticos. Agentes do sistema podem ser consultados na sessão autenticada e vinculados por ID ao projeto correto. A atribuição automática manual escolhe o primeiro agente habilitado do projeto; não há execução nem seleção inteligente por skills. Sem agentes habilitados, rascunho pode ficar sem responsável e atribuição automática retorna erro.

Aviso permanente identifica dados mockados e armazenamento no navegador. Remover exemplos exige confirmação e preserva tarefas manuais/equipes; dependências dos exemplos removidos são limpas. Simular onboarding substitui os exemplos e suas edições, preservando tarefas manuais. Dados inválidos no armazenamento são preservados com bloqueio de gravação, sem recuperação destrutiva automática; falha na gravação não aplica a alteração em memória.

## Arquivos, dados e decisões

App.tsx conecta navegação; DashboardLayout.tsx inclui Tarefas preservando menu FAC-032 anterior. TasksPanel.tsx apresenta quadro/lista/modal; tasks-model.ts define schemas Zod e regras; tasks.css define layout responsivo; testes ao lado. Sem dependências novas, migrations ou ADR: stack ADR-003 preservada.

Namespace local `le-fabrique.tasks.demo.v1`, schemaVersion=1. Datas/IDs dos exemplos são sintéticos identificados, não evidência de execução. Consulta administrativa usa GET /settings e catálogo de projetos existente; não acrescenta API nem altera Ticket/Run. Etapas visuais não publicam READY/outbox ou autorizam DONE real.

## Diff, fontes e revisão

[Hashes finais e fontes explicitamente lidas](evidencias/FAC-033/revisao.json). Novo código em arquivos próprios; diff dos dois arquivos de integração recuperável com `git diff -- apps/web/src/App.tsx apps/web/src/DashboardLayout.tsx`. Demais arquivos novos constam no manifesto. O diff contra HEAD de DashboardLayout também inclui FAC-032 prévio, que deve ser preservado ao revisar/reverter FAC-033. Autoload NÃO VERIFICADO. Manual, README, piloto, ADR-003, prompt, guia, políticas e capítulos afetados lidos; histórico integral e credenciais não carregados.

## Checks e evidências

Revisão: árvore de trabalho final identificada pelo manifesto; autorrevisão, não revisão independente.

| Critério / comando | Resultado | Evidência |
|---|---|---|
| Baseline `npm run test -w @le-fabrique/web` | PASS | 46 testes, 18 arquivos antes do diff |
| Final `npm run test -w @le-fabrique/web` | PASS | 53 testes, 20 arquivos |
| `npm run typecheck -w @le-fabrique/web` | PASS | saída 0 na revisão final |
| `npm run build -w @le-fabrique/web` | PASS | 138 módulos, bundle gerado |
| Biome nos sete arquivos web afetados | PASS | sem achados na revisão final |
| Atribuição/dependências/remoção/payload | PASS | tasks-model.spec.ts: seis cenários |
| Navegação/rotulagem demonstrativa | PASS | TasksPanel.spec.tsx: renderização estática |
| Browser interativo / persistência real / visual mobile | NOT_RUN | nenhum browser disponível nesta sessão; testes estáticos não provam interação |
| PostgreSQL/Redis/worker real | NOT_RUN | integração real fora desta fase demonstrativa |
| Revisão independente | NOT_RUN | nenhuma sessão independente acionada |

Primeira checagem TypeScript identificou índices potencialmente ausentes em fixtures; corrigidos com validação explícita e checagem final PASS. Gate documental final PASS: `python3 scripts/validar-documentacao.py`, 381 Markdown correntes, 2318 links locais, 601 snapshots, zero erros. Verifica estrutura/hashes, não valida comportamento interativo.

## Riscos, limites e rollback

Dados locais não são base compartilhada da aplicação. Onboarding real, seleção por papel/skills, persistência PostgreSQL com controle de versão, consumo pelo worker, sincronização entre abas e testes interativos permanecem PLANEJADOS. Não afirmar tarefa mockada como trabalho executado ou aceite. Referências a agentes removidos/desabilitados aparecem como indisponíveis e precisam ser corrigidas para salvar. Responsável sem conta autenticada pode ser cadastrado para planejamento, sem elegibilidade automática de execução.

Rollback: retirar import/ramo Tarefas no App, destino e item Tarefas no layout, e arquivos novos FAC-033; preservar alterações prévias FAC-032/documentação. Remover somente o namespace local se autorizado, pois contém também tarefas manuais. Nenhuma operação de banco requer rollback.

## Documentação e lessons

Atualizados [controle](../../03-modulos/controle/00-README.md), [tickets](../../03-modulos/tickets/00-README.md), [feature](../../04-features/01-control-settings.md), [contrato local](../../05-contratos/schemas/00-entidades.md), índice, changelog, backlog, matriz e catálogo de entregas. Sem novo capítulo didático: mapa de ordem mantém sequência existente. Lesson nova N/A; este incremento aplica schemas runtime e modal existente, sem criar conhecimento arquitetural novo.

## IA, custos, pendências e aceite

Cliente Codex; modelo efetivo, modo de cobrança, tokens e custos não observados. Sem providers reais acionados, autenticação nova, gastos adicionais autorizados, subagentes ou handoff. Critérios de regras/renderização ATENDIDOS em testes; comportamento interativo/persistência real NÃO VALIDADO. Revisão independente e aceite humano exato pendentes; não declarar DONE.
