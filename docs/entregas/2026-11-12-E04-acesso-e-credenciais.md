# Entrega 2026-11-12 — E04 Acesso e credenciais (T04-01, T04-02, T04-03)

**Tickets:** [T04-01](../NEW%20MVP/03-BACKLOG.md#t04-01) · [T04-02](../NEW%20MVP/03-BACKLOG.md#t04-02) · [T04-03](../NEW%20MVP/03-BACKLOG.md#t04-03) (parcial). Estado em [07-CHECKLIST.md](../NEW%20MVP/07-CHECKLIST.md). Decisões: [ADR-0005](../adr/0005-autenticacao-sessoes-e-cofre-de-conexoes.md). **Revisão independente: NOT_RUN** (feita pelo próprio autor; rode a skill `revisor-de-codigo` em sessão separada antes do merge).

## Objetivo
Restringir o painel a um administrador com senha + TOTP, endurecer sessões e ações sensíveis, e guardar conexões (GitHub, Jira, ClickUp, CLI, API, MCP) em um cofre cifrado com chave fora do banco, sem vazar valores em logs ou respostas.

## O que mudou
- **Banco:** migration 3 (2º fator, códigos de recuperação, desafios, throttle, trilha, metadados de sessão; apaga o admin `dev-admin`) e 4 (`connections`). Nenhuma migration anterior foi editada.
- **Backend (`internal/security`, `internal/httpapi`, `internal/integrations`):** argon2id; TOTP RFC 6238 com anti-replay; AES-256-GCM (`Sealer`); login em duas etapas; recuperação de uso único; throttle por conta/IP; reautenticação; CSRF (token + `Origin`/`Sec-Fetch-Site`); HTTPS obrigatório em produção + HSTS; trilha de autenticação; cofre (`Vault`: criar, listar, testar, rotacionar, revogar, `Resolve`/`ResolveRef`); `Secret`, `Redactor` (logs + respostas JSON + detalhes de teste); testador de conexões.
- **Removido:** `POST /api/dev/session`, `DEV_AUTH_ENABLED` (agora **recusada**), botão "Entrar (somente dev)" e `UnauthorizedState`.
- **CLI:** `make create-admin` (`cmd/migrate create-admin`, senha por variável de ambiente, nunca por argumento).
- **Contrato:** 19 operações novas em `contracts/openapi.yaml` (auth, sessões, conexões) e `Session` com `csrf_token`/`reauth_until`/`totp_reset_required`.
- **Front:** `LoginPage` (senha → TOTP/recuperação; 1º acesso cadastra o 2FA e exibe os códigos uma vez), `SecurityPage` (sessões, substituição do 2FA, trilha), `ConnectionsPage` (cofre; o segredo nunca volta à tela), `ReauthProvider` (diálogo de senha + código e repetição da ação), token CSRF no cliente HTTP. Menu: **Conexões** e **Segurança**.
- **Infra/config:** `VAULT_KEY`, `VAULT_KEY_ID`, `TRUST_PROXY`, `REQUIRE_HTTPS` (Compose, `.env.example`, `.env.production.example`); nginx com `set_real_ip_from` e `Host` com porta (necessário à checagem de origem).
- **Dependência nova:** `golang.org/x/crypto` (argon2).
- **Atenção local:** o `.env` de dev (ignorado pelo Git) ganhou uma `VAULT_KEY` aleatória e perdeu `DEV_AUTH_ENABLED`; o banco de dev perdeu o `dev-admin` (migration 3). Crie a conta com `make create-admin`.

## Como testar
```sh
make up && make create-admin   # depois abra http://127.0.0.1:5180 e cadastre o 2FA
make lint && make test && make test-integration
```

## Verificação (2026-11-12, macOS, Go 1.27.3, Node 24, Docker 29.3.1, PostgreSQL 17)
| Verificação | Resultado |
|---|---|
| `make lint` (gofmt, go vet, Biome) | PASS |
| `make test` — Go (config, httpapi, logging, security, worker, contrato) + front (vitest, 34 testes) | PASS |
| `make test-integration` — 40 testes em PostgreSQL real (inclui migração com/sem `dev-admin`) | PASS |
| `go test -race` unidade + contrato + integração (concorrência) | PASS |
| `tsc -b` e `npm run build` | PASS |
| `make check-secrets` | **FAIL no commit `b2eaff0`** (corrigido na entrega do E05) — fixtures de teste do redator imitavam tokens reais; o relato original de PASS estava errado. Hoje: PASS |
| Stack dev reconstruído (`healthy`), migrations 3–4 aplicadas; 401 sem sessão, `/api/dev/session` 404, login inexistente 401, origem estrangeira 403 | PASS |
| `nginx -t` (imagem de produção, `set_real_ip_from`) | PASS |
| Login/2FA contra GitHub, Jira e ClickUp **reais** | **NOT_RUN** (sem credenciais autorizadas; os testes usam servidores falsos) |
| Ensaio completo do stack de produção, TLS real, IP real atrás do proxy do host | **NOT_RUN** |
| Prova de "segredo ausente em prompts" | **NOT_RUN** (ainda não há pipeline de prompts — E09/E10) |
| Revisão independente | **NOT_RUN** |

### Critérios de aceite → evidência
- **T04-01 "Senha sem 2FA não abre sessão":** `TestPasswordAloneNeverOpensSession` (0 linhas em `admin_sessions`; rotas de dados 401; token do desafio não vale como sessão).
- **T04-01 "Recuperação funciona uma vez mesmo com requisições concorrentes":** `TestRecoveryCodeWorksExactlyOnceUnderConcurrency` (6 desafios × mesmo código e 6 requisições no mesmo desafio: exatamente 1 sucesso; reuso → 401).
- **T04-02 "Sessão revogada perde acesso":** `TestRevokedSessionsLoseAccess`.
- **T04-02 "Requisição forjada ou sessão sem reautenticação é recusada":** `TestCSRFAndOriginAreEnforced`, `TestSensitiveActionsRequireRecentReauthentication`.
- **T04-03 "Rotação mantém vínculos sem expor valor anterior":** `TestRotationKeepsLinksAndDropsPreviousValue`, `TestConcurrentRotationsAreSerializedByVersion`.
- **T04-03 "Revogação é observável antes do próximo uso":** `TestRevocationIsObservableBeforeNextUse` (o destino não é chamado após revogar; o segredo é apagado do banco).
- **T04-03 "Busca com segredo sentinela não acha texto em logs, respostas ou prompts":** `TestSentinelSecretNeverAppearsAnywhere` cobre respostas HTTP, logs reais, trilha, e as tabelas `connections`, `auth_events`, `admin_sessions`, `admins`, `outbox`, `execution_events`; **prompts: NOT_RUN** (ver acima).

## Limites e riscos
- **T04-03 fica parcial:** sem teste remoto de `cli`/`api`/`mcp` (devolvem `not_run`), sem rotação da chave mestra, sem prova em prompts e sem teste contra serviços reais. Mecanismos oficiais de CLI são do E09.
- **Perder `VAULT_KEY` perde conexões e o segredo TOTP** (o login por código de recuperação segue funcionando). Guarde cópia fora da VPS ([backup](../operacao/backup-restauracao.md)).
- Bloqueio por conta permite que um atacante trave o login por 15 min (aceito; ADR-0005).
- Não há QR code: o 2FA é cadastrado por chave ou URI `otpauth://`.
- Não expor o painel na internet antes do ensaio de produção com TLS (T02-02).

## Rollback
`git revert` do commit desta entrega. Migrations 3 e 4 têm `Down` (`go run ./cmd/migrate down` duas vezes), mas **o `dev-admin` apagado não volta** e a remoção das colunas descarta o 2FA já cadastrado. Em produção, prefira restaurar o backup do banco.
