# OPS-003 — Aceites e reconciliação dos deltas locais

Data: 2026-10-02  
Status: IMPLEMENTADO / AWAITING_HUMAN  
Domínio: operação, infraestrutura e controle  
Base SHA: `62deb5d9838433f0d02c5c3021aa96492f9f7031`  
Revisão do código/configuração: `e09204c5205af047d79bcd7d530a1acd1db563ca`  
Branch: `fix/ops-003-reconcile-accepted-state`

## Objetivo

Registrar os aceites explícitos de FAC-012A/B e incorporar os dois deltas locais identificados nas worktrees preservadas, sem aplicar migration, iniciar serviços ou fazer deploy.

## Implementação

- FAC-012A: aceito para código `3b6f3e4624b5935e4fe6ea067406c631a6b0ec32` e revisão documental `3ef3d543b98fb48226714787315f4de947fa6dd9`.
- FAC-012B: aceito para código `e7b9bf60e03089c502255d19edbb5d1e31725d2c` e revisão documental `3e28363b0642f8e05840bb649b681e3837e1e015`.
- `compose.yaml`: porta do Nginx publicada em `127.0.0.1:8080`, permitindo encaminhamento por proxy HTTPS do host sem bind público direto.
- Migration `20261002010538_reconcile_prisma_schema_drift`: remove defaults PostgreSQL de UUIDs gerados pelo Prisma Client e renomeia o índice conforme o schema atual. Nenhum banco foi conectado ou alterado.
- O README raiz agora aponta para `BACKLOG.md` como estado por ticket e `documentacoes/INDEX.md` como índice das evidências.

## Diff sanitizado

`git diff 62deb5d9838433f0d02c5c3021aa96492f9f7031..e09204c5205af047d79bcd7d530a1acd1db563ca`:

```text
compose.yaml                                               bind da porta do proxy ao loopback
apps/api/prisma/migrations/                                migration corretiva versionada; não aplicada
README.md / documentacoes/                                 aceite FAC-012A/B e estado atualizados
BACKLOG.md / CHANGELOG.md / documentacoes/INDEX.md         status/evidência centralizados
```

## Verificação

- `npm ci`: 211 pacotes adicionados, 217 auditados, zero vulnerabilidades reportadas.
- `npm run db:generate`: Prisma Client 6.12.0 gerado.
- `DATABASE_URL=<URL sintética> npm exec -w @le-fabrique/api -- prisma validate`: schema válido.
- `POSTGRES_PASSWORD`, `ADMIN_API_TOKEN`, `WORKER_API_TOKEN`, `WORKER_ID` e `DATABASE_URL` sintéticos + `docker compose config --quiet`: passou; não iniciou containers.
- `npm run lint`: 140 arquivos, sem erros.
- `npm run typecheck`: contratos, runtime, API, worker e web passaram.
- `npm test`: 169 testes passaram (launcher 2, contracts 19, runtime 38, API 50, worker 56, web 4).
- `npm run build`: todos os workspaces passaram; Vite compilou 118 módulos.
- `git diff --check`: passou.

## Limites e rollback

A migration não foi aplicada; verificações de provider/GitHub e worker real ficam em tickets posteriores. Rollback: reverter `e09204c5205af047d79bcd7d530a1acd1db563ca`; como não houve aplicação de migration nem deploy, não há rollback de dados. Restaurar o bind anterior exige avaliar primeiro o proxy/firewall do ambiente.

## Documentação e uso de IA

Atualizados READMEs de infraestrutura, operação, runtime e controle; relatórios FAC-012A/B, planejamento, índice, backlog e changelog. Não foi chamado provider do produto nem houve custo.

## Aceite

O responsável autorizou os deltas e aceitou FAC-012A/B. OPS-003 aguarda revisão humana da integração/configuração desta entrega.
