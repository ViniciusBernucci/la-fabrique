# FAC-012AH — Preservar jobs enquanto capacidade global está ocupada

Status: IMPLEMENTADO / AWAITING_HUMAN. Data: 2026-10-03. Baseline `2eced85` (519 testes + 10 PostgreSQL + 2 Redis). Branch `fix/fac-012ah-admission-capacity`, worktree exclusiva `/home/vinicius/le-fabrique-fac-012ah`, checkpoint SHA/patch/untracked limpos, nenhum outro writer nela.

Objetivo: evitar job FAILED sem tentativa quando outro writer ocupa capacidade global. Recusa conhecida pré-claim (nenhuma autoridade/IA/check) pode ser adiada como pausa global; erros de setup/execução/lease/evidência não podem virar retry cego.

Escopo: HTTP 429 com code WRITER_BUSY só para ocupação/unique race/rollback Serializable conhecido, classe tipada de deferral do ControlClient só na rota claim, helper AG e testes API/worker/Redis preservando job/attempts, docs controle/operação/planejamento/lessons/índices/estado final MVP interno. Critérios: sem stop mesmo vencido continua recusado; diferente ticket aguarda sem perder intent, sem grant/fence novo; P2002 recusado sem criar autoridade; job adiado só antes de claim, retomada após liberação recebe perfil/configuração atuais; 409 stale/foreign/unknown da própria run e outras falhas propagam. Um writer, duas correções máximas, sem piloto/provider/gasto/produção/deploy. Rollback mantém exclusão/evidências; aceite exato pendente.

Ajuste de bootstrap observado antes da edição: npm test em worktree nova executa CLI antes de compilar os pacotes internos, falhando com MODULE_NOT_FOUND. Incluir pretest mínimo dos contratos/runtime para tornar o comando reproduzível em checkout limpo; sem mudar asserções ou ocultar falhas.

Resultado: `f2d80f1671ac8b0c76a5c76f35be4ce626fcab6d`, 528 testes + 10 PostgreSQL + 3 Redis passaram; duas correções documentadas. [Evidências](../../../09-entregas/2026/controle/2026-10-03-FAC-012AH-admissao-capacidade.md). Aceite exato pendente.
