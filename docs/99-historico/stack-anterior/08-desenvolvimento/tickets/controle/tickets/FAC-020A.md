# FAC-020A — Ativar home no aplicativo em desenvolvimento
Status inicial: READY em 2026-10-05, solicitado pelo responsável ao informar que o aplicativo habitual não abre o dashboard principal.
## Objetivo
Integrar a entrega FAC-020 à branch developer usada pelo Vite na porta 5173, mantendo a home como entrada padrão.
## Escopo e autorização
Integração local por fast-forward de feat/fac-020-home-dashboard em developer autorizada pelo pedido atual de tornar a tela criada a home do software. Nenhum deploy/produção, push, migration ou execução de provider. Preparação/documentação na worktree exclusiva FAC-020, sem novo writer. Root developer limpo em 8285173; feature limpa em e87206f. Demais processos não serão interrompidos.
## Critérios e baseline
Código 904d0b5 já verificado com 42 testes/typecheck/lint/build. Conferir fast-forward sem conflitos, Vite real em 5173 servindo HomeDashboard, navegador abrindo home sem API, retorno/login, atualização documental e estado Git limpo. Aceite visual não presumido.
## Limites e rollback
Um writer e até duas correções. Reverter somente código 904d0b5 se necessário, sem apagar branches/worktrees ou dados. Modelo/cobrança/cota não comprovados; nenhuma nova chamada de IA/API/fallback.
## Entregáveis
Relatório FAC-020A, README atual, índices/changelog/backlog e lesson de verificação do serviço ativo; evidência do browser na porta habitual.

## Entrega
Status: AWAITING_HUMAN. Código cde0cc6 integrado em developer; browser na porta 5173 PASS. Integração/limpeza autorizadas; aceite visual não presumido.
