# FAC-016 — .env e processo dev ativo

O supervisor dev carrega .env na inicialização. Provisionar WORKER_PROVIDER_ROOT e caminhos dos clientes não altera ambiente já herdado por tsx watch. Reiniciar npm run dev é necessário; hot reload do código não equivale a recarregar o supervisor. A inspeção publicou somente presença de variáveis operacionais, nunca ambiente completo.

O pedido real 83098492-a06f-4cd4-9b5c-f665e796f8e3 retornou COMPLETED/AUTH_REQUIRED para Codex 0.159.2. Nenhum processo do usuário foi encerrado pelo agente. [Evidências e procedimento](../configuracao/2026-10-05-FAC-016-feedback-verificacao-codex.md). AWAITING_HUMAN.
