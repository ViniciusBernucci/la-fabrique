# OPS-003 — Consolidar revisões aceitas e deltas locais

Status: AWAITING_HUMAN

## Objetivo

Registrar os aceites humanos explícitos de FAC-012A/B e integrar em `developer` os dois deltas locais que o responsável autorizou: bind loopback da porta publicada e migration corretiva gerada para alinhar schema Prisma/histórico.

## Critérios de aceite

1. Registrar FAC-012A aceito para código `3b6f3e4624b5935e4fe6ea067406c631a6b0ec32` e relatório/documentação `3ef3d543b98fb48226714787315f4de947fa6dd9`.
2. Registrar FAC-012B aceito para código `e7b9bf60e03089c502255d19edbb5d1e31725d2c` e documentação revisada `3e28363b0642f8e05840bb649b681e3837e1e015`.
3. Bind do Nginx publicado em `127.0.0.1:8080` para não expor diretamente a porta de origem.
4. Incluir a migration corretiva Prisma com nome descritivo, sem aplicá-la a nenhum banco.
5. Preservar as branches originais e não remover dados/volumes; documentar rollback.
6. Lint, typecheck, testes, build, Prisma validate, Compose config e `git diff --check` passam.

## Baseline e limites

Base `3e28363b0642f8e05840bb649b681e3837e1e015`, branch isolada `fix/ops-003-reconcile-accepted-state`, worktree `/home/vinicius/le-fabrique-ops-003`. Sem deploy, push, login, acesso a credenciais, execução de migration ou limpeza de fila/dados. Usar URL de banco sintética para validação local.

## Provider

Nenhum; apenas documentação, configuração de bind e migration aditiva/corretiva sem aplicação.

## Entrega

Revisão `e09204c5205af047d79bcd7d530a1acd1db563ca`; relatório `documentacoes/operacao/2026-10-02-OPS-003-aceites-e-deltas-locais.md`. OPS-003 aguarda aceite específico desta integração.
