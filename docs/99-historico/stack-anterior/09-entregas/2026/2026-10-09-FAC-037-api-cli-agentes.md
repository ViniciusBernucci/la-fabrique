# FAC-037 — Cadastro por API/CLI e IA/modelo dos agentes

Data: 2026-10-09, Europe/Berlin. IMPLEMENTADO cadastro/atribuição / AWAITING_HUMAN. Base de código 36e48f91a29e808ebd625bf11a4e24264c78a867, branch feat/fac-037-api-cli-agents em `/home/vinicius/le-fabrique-fac-037`; diff local sem commit. Sem merge/push/PR/deploy. Execução API PLANEJADA.

## Objetivo, problema e critérios de aceite

O pedido do responsável exige escolher chave de API ou assinatura CLI antes do formulário de cadastro e escolher IA/modelo cadastrados para os agentes. O formulário anterior só aceitava CLI e combinava conta/modelo numa opção. [Ticket READY autorizado nesta sessão](../../08-desenvolvimento/tickets/configuracao/FAC-037-api-cli-agentes.md).

Critérios de cadastro, criptografia, compatibilidade e seleção ATENDIDOS por implementação e checks automatizados. Interação completa em browser, execução por API e aplicação operacional NÃO VALIDADAS. Cadastro não faz inferência nem autentica a chave por chamada externa.

## Solução e comportamento implementado

Adicionar conta ou Cadastrar integração de IA apresenta duas escolhas antes de criar o rascunho. Assinatura preserva seu fluxo oficial. API permite OpenAI/Claude com senha de entrada protegida e catálogo informado por conta. Antigravity permanece CLI. Cancelar descarta o cadastro. IA do agente lista apenas contas cadastradas/habilitadas, identificando fornecedor e modo; Modelo da IA mostra somente seu catálogo. Trocar a IA redefine o modelo para o padrão ou primeiro disponível. Esse seletor é compartilhado entre pré-configurados e personalizados.

Ao salvar, a chave é enviada separadamente da configuração. Chave vazia de conta existente preserva o segredo; fornecedor diferente exige nova chave. Modo de autenticação não muda no mesmo ID. Conta API pode ser atribuída, mas o runtime CLI não a executa; a UI informa a indisponibilidade. Remover a conta e salvar limpa o segredo na mesma transação. Falha de versão ou criptografia não confirma a gravação.

## Arquivos e decisões

- `packages/contracts/src/index.ts`: modos e write-only apiKeys; contratos/testes de cadastro.
- `apps/web/src/SettingsPanel.tsx`, `SettingsModal.tsx`, `AgentModelSelector.tsx`, `TeamsPanel.tsx`, `settings-view-model.ts`: fluxo inicial, chave transitória, identificação do fornecedor e IA/modelo separados.
- `apps/api/src/settings/settings.service.ts`, `settings.controller.ts`, `api-key-encryption.ts`: validação, cifra, persistência/limpeza transacional; testes adjacentes.
- Onboarding/verificação CLI e `apps/worker/src/agent-route.ts`: bloqueio de contas API, inclusive evidência tardia.
- Prisma schema/migration e `.env.example`: storage privado e configuração da chave mestra.

ADR-003/stack preservados. O pedido expande cadastro antes limitado a CLI, sem revogar a proteção de execução/gastos. Não houve instalação de provider, nova dependência ou acesso a credenciais reais. Outras sessões estavam abertas no workspace principal; criou-se worktree própria, sem iniciar writer concorrente ali.

## Dados, API, eventos e configuração

[Contrato de dados](../../05-contratos/schemas/00-entidades.md#chaves-de-api-fac-037). PUT `/api/settings` recebe apiKeys opcional; GET e snapshots não devolvem segredo. Tabela `provider_api_credentials` separada do JSON público, AES-256-GCM com nonce aleatório e AAD por instalação/fornecedor. Migrations novas não foram aplicadas.

Para ativar cadastro API, o operador precisa aplicar `20261009090000_provider_api_credentials` pelo procedimento normal de implantação e configurar `PROVIDER_API_KEY_ENCRYPTION_KEY` com 32 bytes hexadecimais privados no processo da API. Sem chave válida, a API recusa a escrita, sem valor padrão. Preservar chave mestra e ciphertext em backup privado; a chave mestra não deve ir ao Git/painel/worker/contexto. Nenhum orçamento, extras ou fallback são ativados pelo cadastro.

## Diff e artefatos

[Fontes explicitamente lidas e hashes](evidencias/FAC-037/fontes.sha256), [patch do código](evidencias/FAC-037/codigo.patch) e [manifesto do código alterado](evidencias/FAC-037/codigo.sha256). Descoberta automática de instruções NÃO VERIFICADA; AGENTS, políticas completas, Manual, README, piloto, ADR-003, prompts/guia, perfil/skill local e capítulos afetados lidos explicitamente. Sem leitura de credenciais. Hash do commit da entrega não existe.

## Testes e evidências

Configuração: worktree local sobre 36e48f9, diff FAC-037; chaves, modelos e contas sintéticos. Nenhum provider real chamado.

| Comando/critério | Resultado | Evidência |
|---|---|---|
| npm test -w @le-fabrique/contracts | PASS: 55 testes | Cadastro API/CLI, Antigravity API rejeitado, chave fora de DTO |
| npm test -w @le-fabrique/api | PASS: 192 testes | Cifra autenticada, chave vinculada à conta, preservação/remoção, versão stale e CLI guard |
| npm test -w @le-fabrique/web | PASS: 66 testes | IA habilitada, catálogo da selecionada, estados vazios, regressões do painel |
| npm run typecheck | PASS | Contratos/runtime/API/worker/web/scripts/integração |
| npm run build | PASS | Cinco workspaces; build web repetido após identificação de fornecedor |
| npm run lint | PASS com aviso anterior | useOptionalChain em run-delivery.service.ts, arquivo sem alterações nesta entrega |
| npm test -w @le-fabrique/worker | PASS: 216 testes | Conta API nunca seleciona adapter CLI |
| python3 scripts/validar-documentacao.py | FAIL de baseline; zero erros novos | [Baseline](evidencias/FAC-037/documentacao-baseline.json) e [revisão atual](evidencias/FAC-037/documentacao-atual.json): sete handoffs FAC-035 fora do índice |
| git diff --check | PASS | Sem whitespace inválido no patch |
| Browser, PostgreSQL/migration, inferência API e revisão independente | NOT_RUN | Sem ativação operacional ou sessão independente |

Primeira execução de worker encontrou node-pty sem módulo nativo na instalação isolada feita com ignore-scripts; `npm rebuild node-pty` preparou a dependência existente. Isso não altera dependências/versionamento ou providers. Checks posteriores pertinentes foram executados: 529 testes nos quatro workspaces afetados passaram. O gate documental foi comparado com uma extração temporária read-only da revisão base: sete erros preexistentes idênticos e nenhum erro adicional. Não foram alterados arquivos do handoff FAC-035 para corrigir uma pendência de outra entrega. Não houve falha funcional conhecida de baseline alterada para ocultar regressão.

## Segurança, riscos e limitações

Chave permanece apenas no campo transitório durante edição, requisição autenticada e serviço de cifra; não entra em FactoryConfiguration, configuração do worker, lista de contas, mensagens, logs ou artefatos. AES mestra é responsabilidade da implantação e exige backup separado. Segredo cifrado na tabela não elimina risco de comprometimento do processo/host. Sem migration e chave mestra privadas, cadastro API não fica operacional. Não há adapter de execução API: selecionar uma conta API não a transforma em CLI nem autoriza uma chamada paga. RLS/tenancy/broker permanecem propostas posteriores existentes.

Autorrevisão de contrato/código não é revisão independente. O teste de cifra usa segredo sintético e não prova operação em produção. Não foi feito backup/restore real da tabela nova. Nenhuma regra local de sources ou auth oficial foi alterada.

## Rollback

Reverter somente este diff/commit quando existir, preservando trabalho de outras frentes. Se ainda não aplicado, deixar de aplicar a migration. Se aplicado e usado, primeiro exportar ciphertext em backup privado com sua chave mestra e remover atribuições/contas API via configuração; a versão anterior rejeita authMode API_KEY. Não apagar tabela/segredos em produção sem procedimento autorizado. Rollback operacional NÃO ENSAIADO; não houve alteração no banco ativo.

## Documentação atualizada

Módulo configuração, feature de settings, contrato de dados, ticket/backlog, matriz de cobertura, changelog e índices de entregas/global atualizados. Sem novo capítulo didático ou mudança no mapa de ordem; ticket/relato são apêndices. Não há lesson nova: são aplicação de contratos, concorrência e gestão de segredo, sem conceito adicional a consolidar.

## Uso de IA e custos

Codex nesta sessão, modelo efetivo/versão/uso/custos não observados por evidência do cliente. Sem auth lida/exportada, contratação, inferência API ou chamadas de login/preflight real. Gastos fixos e incrementais da sessão desconhecidos; não convertidos de tokens. Um writer na worktree; sem subagentes ou handoffs. Hora de execução e comandos observados no ambiente local em 2026-10-09.

## Pendências, próximos passos e aceite

Responsável: usuário/revisor independente. Revisão exata, browser e implantação pendentes; nenhum DONE atribuído. Inferência/execução por API requer adapter agentivo próprio e controle financeiro; não foi fingida por conexão CLI. Trabalho preservado na branch isolada; não integrado à developer. Esta entrega fecha implementação do cadastro/seleção e explicita limites operacionais.

## Adendo — commit, integração e limpeza autorizados

Em 2026-10-09 (Europe/Berlin), o responsável autorizou commit, merge na developer e remoção das worktrees/branches órfãs. Código/documentação commitados em `67ce0a09272966b7b1234663d087e79d12b4381f` e integrados por fast-forward à developer, sem conflitos. Revisão anterior encerrada; nenhum outro processo tinha cwd nas worktrees removidas. Hashes dos 20 arquivos do código foram conferidos na developer e correspondem ao manifesto da implementação.

Removidas as worktrees FAC-035 e FAC-037, limpas e integralmente integradas, e 72 branches locais de tarefas cuja ancestralidade na developer foi comprovada. Preservadas main/developer e a worktree/branch FAC-036-instrucoes-md: possui alterações e arquivos novos ainda não commitados/integrados, portanto não é órfã descartável. Não houve remoção forçada, reset ou alteração desse trabalho. [Inventário anterior](evidencias/FAC-037/inventario-antes-limpeza.json) e [resultado da limpeza](evidencias/FAC-037/limpeza-concluida.json) preservam paths/SHAs.

O cliente Prisma foi regenerado localmente com `npm run db:generate` (PASS), sem aplicar migration nem acessar banco. As worktrees removidas não eram necessárias para guardar mudanças: código/evidências permanecem no histórico da developer. Nenhum push, deploy, alteração de credenciais ou ativação de inferência por API. Revisão independente, browser e implantação permanecem pendentes; o pedido de integração não foi tratado como aprovação técnica independente ou DONE.

Nota de validação: artefatos `.patch` preservam espaços de contexto obrigatórios do formato unified diff. O whitespace do código foi conferido excluindo esses artefatos com `git diff HEAD^ HEAD --check -- . ':(exclude)**/*.patch'` (PASS); não se removeu espaço sintático do patch apenas para silenciar o validador.

Verificação após integração: `npm run typecheck` PASS na developer em 67ce0a0, incluindo scripts/integração. Validador documental novamente encontrou somente os sete erros preexistentes do handoff FAC-035; nenhum erro de link ou índice acrescentado pelo adendo. Nenhum código mudou após os 529 testes da implementação, e os hashes do código integrado foram conferidos.
