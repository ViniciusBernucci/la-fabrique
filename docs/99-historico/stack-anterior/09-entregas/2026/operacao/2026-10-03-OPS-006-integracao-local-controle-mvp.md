# OPS-006 — Integração local e controle do MVP

Data: 2026-10-03. Status: IMPLEMENTADO / AWAITING_HUMAN.
Baseline: `ce93b20440c3f7ecc216ae6513396596cdf9a76e`.
Revisão integrada de código: `26e412d5d89b337535623a7216c629dc8191c207`.
Branch de preparação: `ops/ops-006-final-local-integration`.

## Objetivo e resultado

Executada a autorização prévia do responsável para reunir o trabalho local em developer e remover worktrees integrados. OPS-005 foi integrado pelo merge `cb95cf5`; FAC-012L (incluindo FAC-012K) pelo merge `79ae282`. Developer avançou por fast-forward para `26e412d`. Merge local não registra aceite humano dos incrementos e não implica push/deploy.

O consumer real permanece desabilitado por padrão e a fixture só existe em testes. Compose usa controle/bancos, sem worker container; os binds de API/Redis permanecem locais. Unit host é template não instalado. A resolução dos conflitos preservou consumer gated, shutdown, config de ambas as branches e histórico de changelog; reintroduziu explicitamente o import OrchestrationJob que o merge textual havia removido.

`26e412d` alinha o env example host com gate/roots/CLIs/lease do FAC-012L e remove a suposição de UID do user manager. `%U` no system manager resolve para o UID do gestor, independente de User=; por isso XDG_RUNTIME_DIR/DBus ficam explicitamente configurados no env externo após provisionamento. Evidência: man systemd.unit instalado (systemd 249) e [fonte oficial systemd](https://github.com/systemd/systemd/blob/main/man/systemd.unit.xml).

## Controle e lacunas

[CONTROLE-MVP.md](../../../08-desenvolvimento/06-controle-mvp.md) resume funcionalidades, evidências, aceites e pendências reais de código. Também foi corrigida a divergência do PLANO-MVP, que ainda mostrava FAC-012A pendente apesar do aceite já registrado pelo OPS-003. A/B permanecem DONE; C–L e tickets operacionais pendentes de aceite continuam assim.

A auditoria das APIs/painel/worker confirmou lacunas: resultados/artefatos no painel, replay e recuperação, WAITING_PROVIDER/handoff integrados, múltiplas contas isoladas por instalação, pausa/cancelamento/retomada/aceite de run e gate documental. Piloto/implantação são manuais e não bloqueiam a implementação dessas lacunas internas.

## Checks e diff

No worktree isolado: `npm ci` (211 pacotes; auditoria reportou zero vulnerabilidades), `npm run db:generate` (somente Prisma Client local), `npm run lint` (154 arquivos), `npm run typecheck`, `npm test` (227 testes: launcher 2, contracts 21, runtime 44, API 55, worker 100, web 5), `npm run build` e `git diff --check` passaram. Não houve falha combinada que exigisse nova rodada de correção.

`docker compose --env-file .env.production.example config --quiet` passou; `config --services` listou postgres/redis/api/web. `systemd-analyze verify infrastructure/systemd/le-fabrique-worker.service` terminou com exit 0; avisos pertenciam às unidades netplan-ovs-cleanup/snapd do host, não ao template. Nenhuma unidade foi instalada/iniciada.

Diff sanitizado recuperável: `git diff ce93b20..26e412d -- apps packages compose.yaml infrastructure .env.example .env.production.example`. Arquivos documentais são entregues em commit separado para evitar hash circular.

## Integração e limpeza comprovadas

Antes da limpeza, `git status --porcelain` estava vazio nos worktrees OPS-005/K/L. Após avançar developer, `git merge-base --is-ancestor` retornou sucesso para `7df2765b875af110ccdef1524b616b656e63aa9e`, `a305b38607bb70c5747b4fa8d6f0e5a579621258` e `a1d904b`. Foram removidos por git worktree remove os diretórios `/home/vinicius/le-fabrique-ops-005`, `/home/vinicius/le-fabrique-fac-012k` e `/home/vinicius/le-fabrique-fac-012l`, incluindo artefatos locais regeneráveis. Nenhum commit/branch ref foi apagado; recuperação por git worktree add e npm ci permanece possível.

Depois dessas remoções, git worktree list mostrou somente developer e o worktree OPS-006 de finalização. A documentação será integrada por novo fast-forward; retirar OPS-006 limpo é o último passo após essa integração.

## Limites, rollback e aceite

Nenhum banco/fila/serviço ativo, credencial, provider, repositório remoto, piloto, migration, push ou deploy foi acionado. Os merges não comprovam atualização da instância ativa. Os dois processos Node localizados no diretório principal eram sessões Codex, não servidores em watch que seriam reiniciados pela atualização.

Rollback: criar revisão isolada, preservar workspaces/checkpoints, manter gate de execução false e reverter somente deltas identificados; não reintroduzir fixture em produção nem resetar developer. Branch refs preservadas permitem reconstruir worktrees. Nenhuma reversão de dados é necessária porque não houve operação em serviços.

Atualizados controle, README raiz/infraestrutura/operação/planejamento, plano, backlog, changelog, índices e lesson de sandbox/systemd. Provider/modelo efetivo/tokens/custo da sessão não disponíveis por telemetria confiável; testes não chamaram clientes para inferência e não habilitaram gastos. Aceite humano da revisão documental exata permanece pendente; objetivo global do MVP continua ativo.
