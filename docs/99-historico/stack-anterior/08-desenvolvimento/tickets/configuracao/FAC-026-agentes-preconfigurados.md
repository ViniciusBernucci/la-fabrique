# FAC-026 — Agentes pré-configurados na lista
2026-10-06. READY por pedido do responsável.
Objetivo: retirar agrupamento Funções de execução, apresentar seis agentes pré-configurados na lista Agentes com toggles individuais; clique abre explicação de função/fluxo e botão Configurações do agente abre comportamento existente.
Baseline 99b819b, worktree /home/vinicius/le-fabrique-fac-026 e branch feat/fac-026-preset-agent-list; mesmo writer, sem handoff/subagentes/clientes concorrentes.
Escopo apps/web/src, testes web/browser e docs configuração/controle/índices/lessons. Reusar assignments/versioned PUT, sem novo contrato/backend/migration. Preservar agentes personalizados/skills, permissões e gates.
Critérios: seis pré-configurados visíveis junto de personalizados, switch persistido por agente sem abrir modal, clique informativo sem escrita, seis descrições verdadeiras incluindo integração atual, configuração existente via botão, falha PUT mantém switch/rascunho; fontes/template/mobile/teclado e checks pertinentes.
Provider Codex da sessão, modelo/cota/custo não comprovados. Não executar providers/ativar gastos. Duas correções no máximo. Merge developer e limpeza autorizados anteriormente; sem push/deploy. Docs/patch/evidências/rollback obrigatórios; AWAITING_HUMAN sem DONE antes de aceite exato.

IMPLEMENTADO / AWAITING_HUMAN na revisão combinada b1d794035342128b9ed832d5fe9b74c887eb7075. [Relatório/checks/rollback](../../../09-entregas/2026/configuracao/2026-10-06-FAC-026-agentes-preconfigurados.md).
