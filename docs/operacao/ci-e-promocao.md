# CI, promoção e rollback (T02-03)

Fluxo definido em [`.github/workflows/ci.yml`](../../.github/workflows/ci.yml). **Estado: workflow escrito, NÃO executado no GitHub** (sem autorização para push e sem proteção de `main`; ver [github.md](github.md)).

## O que cada job garante
| Job | Verifica |
|---|---|
| `go` | gofmt, `go vet`, testes de unidade e contrato com `-race`, integração em PostgreSQL 17 real (serviço descartável) |
| `web` | Biome, `tsc -b`, vitest, build de produção |
| `security` | `scripts/check-secrets.sh`, `govulncheck`, `npm audit --omit=dev --audit-level=high` |
| `build` (depende dos três) | imagens `api`, `worker`, `migrate`, `web` com **tag = SHA do commit**; smoke (migrate + API + `/api/readyz`) em rede Docker própria; `evidence-<sha>.json` com commit, run e IDs das imagens, enviado como artefato `evidencia-<sha>` |

## Garantias de aceite
- **PR com falha é bloqueada:** os jobs falham o *check*; para o merge ser realmente impedido é preciso marcar `go`, `web`, `security` e `build` como checks obrigatórios na proteção de `main` (passo pendente de [github.md](github.md)).
- **Imagens e evidências apontam ao mesmo commit:** a tag das imagens e o nome/conteúdo da evidência usam `github.sha`.
- **Segredo de produção fora dos jobs de PR:** o workflow não referencia `secrets.*` nem `environment`; só existem credenciais efêmeras do CI (banco descartável e `VAULT_KEY` gerada no job). PRs de forks nunca recebem segredos do repositório.

## Promoção e rollback
1. Merge em `main` → o CI roda no mesmo commit. A VPS **promove por tag de commit**: `IMAGE_TAG=<sha>` e `make up-prod` (ver [ambientes-docker.md](ambientes-docker.md)); a evidência do commit acompanha o registro de implantação.
2. **Rollback:** reimplantar o SHA anterior com `IMAGE_TAG=<sha-anterior>`; migrations têm `Down`, mas em produção prefira restaurar o backup do banco ([backup-restauracao.md](backup-restauracao.md)) se o schema avançou.
3. Promoção por *digest* (em vez de tag) exige registro de imagens; **PENDENTE**.

## Limites declarados
- Ações do GitHub estão fixadas por tag maior (`@v4`/`@v5`), não por SHA; fixar por SHA é pendência.
- As imagens não são publicadas em registro nem assinadas: o artefato registra os IDs locais do job.
- `actionlint` não estava disponível na máquina: o YAML foi validado só por leitura.
