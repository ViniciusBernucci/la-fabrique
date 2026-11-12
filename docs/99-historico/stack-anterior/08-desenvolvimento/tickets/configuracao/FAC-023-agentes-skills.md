# FAC-023 — Agentes e skills por projeto
Status inicial: READY em 2026-10-05, pedidos explícitos do responsável.
## Objetivo
Equipes deve separar Agentes/Skills, permitir cadastrar mais de seis agentes e registrar skills por projeto, com edição/persistência e associação aos agentes.
## Baseline e escopo
Base 30fbd91. Worktree exclusiva /home/vinicius/le-fabrique-fac-023, branch feat/fac-023-agents-skills; mesmo writer, nenhum handoff. packages/contracts, apps/web, settings da API, testes/docs de configuração/controle/runtime. Registro aditivo no JSON FactorySettings existente, sem migration. Preservar atribuições de execução e limites de um writer.
## Critérios
Cadastro de mais de seis agentes, abas Agentes/Skills, projeto requerido em skill, instruções, vínculo de skill por projeto, edição/remoção, dados disponíveis após recarregar; runtime valida IDs/vínculos e API rejeita projeto ausente. Configuração antiga sem registros segue válida. Modais/fontes/template preservados. Testes de contratos/API/web, typecheck/lint/build, browser com persistência fixture.
## Provider, orçamento e segurança
Codex da sessão, modelo/cota/cobrança não comprovados. Nenhum cliente/provedor será executado, gasto habilitado ou credencial lida. Dois rounds de correção, um writer. Cadastro não altera paralelismo nem concede execução automática. Merge developer/limpeza da worktree autorizados no fluxo da sessão; sem push/deploy/produção.
## Docs e aceite
Relatório datado por domínio, README atual, índices/changelog/backlog/lessons/patch/evidências. AWAITING_HUMAN após checks; DONE só após aceite exato.

## Revisão entregue
IMPLEMENTADO / AWAITING_HUMAN: código a25638acde42ba50576908f1bf6725570a360a99. Critérios de cadastro/vínculos/persistência fixture comprovados; execução automática fora do incremento. [Relatório e evidências](../../../09-entregas/2026/configuracao/2026-10-05-FAC-023-agentes-skills-projeto.md). Merge fast-forward developer até 6760b17 e limpeza da worktree realizados após checks e ancestry; browser principal/5173 PASS. Ver conclusão administrativa no relatório.
