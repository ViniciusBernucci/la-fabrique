# FAC-014 — Modais para funcionários digitais e GitHub

Data: 2026-10-05. IMPLEMENTADO / AWAITING_HUMAN. Baseline `ca76fc5`; developer diretamente autorizada. Código final `00530790fd61a3a98c9faaa996bf9ecab5f62cd4`. Provider elegível: nenhum; sem banco/migrations/IA/login/deploy/push.

## Funcionamento e escopo

Funcionários aparecem em lista com função, conta/modelo, permissão, timeout/tentativas e estado ativo/inativo. Clique/Enter abre edição local da função com campos e alternativas de handoff existentes. Salvar configuração persiste pela API versionada atual, reconciliando alternativas; sucesso fecha modal, erro preserva entrada. Cancelar/Escape descarta alteração local.

GitHub aparece em linha com host, repositório, branch, estado e política de PR. Clique abre configuração e os controles existentes de verificação, login e preparação/aprovação/cancelamento de PR. Campos de configuração usam clone local; cancelar não altera a configuração do painel. Ações de integração/PR continuam explícitas, usam configuração persistida e preservam aprovação humana anterior. Cancelar o modal não desfaz pedidos de verificação/login/PR já enviados. Campos de preparação de PR continuam como rascunho em memória; fechar o modal não afirma salvar ou enviar esse pedido.

SettingsModal centraliza dialog nativo, título acessível via useId/aria-labelledby, showModal/cleanup, cancelamento por Escape e campos/botões bloqueados durante persistência. Contas IA também usam esse componente. Estilos responsivos existentes de FAC-013 reutilizados, sem dependência nova. Salvar qualquer configuração persiste o draft global corrente pela API existente; demais alterações pendentes do painel são incluídas. Nenhum contrato/backend alterado.

## Verificação e evidências

`npm run typecheck -w @le-fabrique/web`, `npm run test -w @le-fabrique/web` (33 testes passaram), `npm run build -w @le-fabrique/web`, `npm run lint` e `git diff --check` passaram na implementação final. Lint mantém só o warning anterior optional chaining na API. Logs efêmeros `/tmp/fac-014-{typecheck,tests,build,lint}.log`. Primeira rodada de correção foi apenas ordenação de imports sinalizada pelo Biome; corrigida e checks web repetidos na revisão final. Sem correção funcional de runtime. Diff sanitizado: commit FAC-014 em apps/web/src/SettingsPanel.tsx e SettingsModal.tsx; relocação dos formulários explica o volume de linhas, sem credenciais/contas reais no patch.

Sem ferramentas de browser disponíveis, não foi executado teste visual/de interação/foco real; suíte web existente não comprova interações dos novos modais. Não houve leitura/escrita de configuração real por esta implementação. Teste manual: abrir Configurações, conferir três listagens; clicar em funcionário e GitHub; alterar/Cancelar/reabrir para verificar descarte; salvar e recarregar para verificar persistência; testar Escape/Tab/celular; confirmar que ações externas exigem clique explícito e não acontecem só por abrir/salvar o modal.

## Limites, rollback e aceite

Pendências do banco/writer OPS-008/009 permanecem independentes; este incremento não as resolve nem libera execução. Rollback reverte commit de UI em revisão autorizada, preservando configuração persistida. Sem migration/dados/contratos/decisão arquitetural modificados. Documentação configuração/planejamento, README raiz, índice, backlog/changelog e lesson existente de modal atualizados. Aceite humano da revisão exata/validação visual pendentes; não DONE.
