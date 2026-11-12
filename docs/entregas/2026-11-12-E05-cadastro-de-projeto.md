# Entrega 2026-11-12 — E05 Cadastro de projeto (T05-01)

**Tickets:** [T05-01](../NEW%20MVP/03-BACKLOG.md#t05-01) entregue. [T05-02](../NEW%20MVP/03-BACKLOG.md#t05-02) e [T05-03](../NEW%20MVP/03-BACKLOG.md#t05-03) **não iniciados: bloqueados por dependências pendentes** (T06-01, T06-02, T15-01 e catálogos de E07 a E09). Estado em [07-CHECKLIST.md](../NEW%20MVP/07-CHECKLIST.md). Decisões: [ADR-0006](../adr/0006-cadastro-de-projeto-em-rascunho.md). **Revisão independente: NOT_RUN** (rode `revisor-de-codigo` em sessão separada antes do merge).

## Objetivo
Cadastrar um projeto por etapas, sem agentes ou workflows associados, persistir o rascunho a cada etapa, mostrar pendências com ação concreta e bloquear a ativação sem preenchimento obrigatório válido.

## O que mudou
- **Banco:** migration 5 — colunas de rascunho em `projects` (descrição, etapa do assistente, dono/repositório/branch/conexão GitHub, plataforma/conexão/identificador da origem, `activated_at`) e tabela `project_changes`. Nenhuma migration anterior foi editada.
- **Backend (`internal/projects`, `internal/httpapi`):** `CreateDraft`, `Configure`, `Validate`, `Activate`, `Suspend`, `Resume`, `Changes`; 7 operações novas em `contracts/openapi.yaml` (`POST /projects`, `PATCH /projects/{id}`, `GET …/validation`, `GET …/changes`, `POST …/activate|suspend|resume`); erros novos: 400 com campo e motivo, 409 `project_active`/`invalid_state`, 422 `validation_failed`.
- **Front:** `NewProjectPage` (cria o rascunho), `ProjectSetupPage` (assistente Dados → Repositório → Origem → Revisão, retoma na etapa salva), `ProjectDetailPage` (configuração, pendências, histórico, suspender/retomar), botão "Novo projeto" na lista.
- **Correção de segurança de processo:** `make check-secrets` **falhava desde o E04** (fixtures de teste do redator com literais no formato de token). Os literais passam a ser montados em tempo de execução; o scanner não foi afrouxado. A entrega do E04 foi corrigida.

## Como testar
```sh
make up    # aplica a migration 5; abra http://127.0.0.1:5180/projects e use "Novo projeto"
make lint && make test && make test-integration && make check-secrets
```

## Verificação (2026-11-12, macOS, Go 1.27.3, Node 24, Docker 29.3.1, PostgreSQL 17)
| Verificação | Resultado |
|---|---|
| `make lint` (gofmt, go vet, Biome) | PASS |
| `make test` — Go (incl. contrato) + front (vitest, 45 testes) | PASS |
| `make test-integration` — PostgreSQL real (inclui migração 1→5 com projeto legado e rollback de 4 migrations) | PASS |
| `go test -race` nos testes de projeto, conexão, concorrência e migração | PASS |
| `tsc --noEmit` | PASS |
| `make check-secrets` | PASS (após a correção acima) |
| Stack dev reconstruído (`healthy`), migration 5 aplicada; rotas novas respondem 401 sem sessão | PASS |
| Fluxo completo no navegador contra o stack real (criar → etapas → ativar) | **NOT_RUN** (login exige TOTP; coberto por integração e vitest) |
| Acesso remoto ao repositório/origem, escopos, preview de itens elegíveis | **NOT_RUN** (T05-02) |
| Associação e validação de workflows/agentes/modelos | **NOT_RUN** (T05-03) |
| Revisão independente | **NOT_RUN** |

### Critérios de aceite → evidência
- **"Projeto novo vazio de vínculos persiste e retorna ao assistente":** `TestNewProjectIsDraftWithoutLinksAndResumesWizard` (sem vínculos; etapa e dados lidos de outra sessão); vitest "retoma o rascunho na etapa em que parou".
- **"Sem preenchimento obrigatório válido, ativação é bloqueada":** `TestActivationBlockedUntilRequiredFieldsAreValid` (422, estado, versão e `activated_at` intactos; parcial continua bloqueado); vitest de revisão e de recusa do servidor.
- **Aceite do épico "alterar A preserva B":** `TestChangingProjectADoesNotTouchProjectB` (configuração, versão e trilha de B).
- **Complementares:** compatibilidade e revogação de conexão (`TestConnectionCompatibilityIsReportedWithReason`), validação de campos (`TestInvalidFieldsAreRejectedWithMessage`), ciclo ativar/suspender/retomar e trilha (`TestLifecycleActivateSuspendResumeWithTrail`), corrida entre gravações (`TestConcurrentStepSavesAllowExactlyOneWinner`), sessão e CSRF (`TestProjectWritesRequireSessionAndCSRF`), prioridade 0 (`TestPriorityZeroIsAcceptedOnCreate`).

## Limites e riscos
- **Ativar não prova acesso remoto nem recursos.** O relatório lista isso em `not_verified` e a tela mostra o aviso; T05-02 e T05-03 acrescentam esses itens como bloqueantes. Nenhum consumidor (worker) usa projetos ativos ainda.
- A checagem de conexão é local (existe, ativa, tipo certo). Uma conexão revogada **depois** da ativação não desativa o projeto; só a próxima retomada ou o preflight (E06) a detecta.
- Não há arquivamento nem exclusão de projeto (fora do escopo do ticket).
- Prioridade da fila: 0 a 4, e a promoção por espera é do E11; aqui ela só é armazenada.

## Rollback
`git revert` do commit desta entrega. A migration 5 tem `Down` (`go run ./cmd/migrate down`), que **descarta** os dados de rascunho e a trilha de alterações dos projetos. Em produção, prefira restaurar o backup do banco.
