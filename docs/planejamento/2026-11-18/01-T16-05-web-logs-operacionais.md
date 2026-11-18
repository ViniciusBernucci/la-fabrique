# Prompt — T16-05, frente `web`

- **Frente:** `web` (React/TS — `apps/web/`)
- **Branch:** `developer` (sem worktree — frente única sequencial)
- **Base:** o commit final da frente `api` deste mesmo ticket (ainda não existe — **não comece
  antes dele terminar**). Confira com `git log --oneline -5 developer`: o commit mais recente deve
  mencionar T16-05/logs operacionais e incluir `contracts/openapi.yaml` com os paths
  `/operations/events` e `/operations/stream`. Se não encontrar, pare e avise — a frente `api` não
  terminou ainda.
- **Tickets:** T16-05 (parte `web`)
- **Perfil de risco:** padrão (tela nova consumindo contrato já fechado; sem migration, sem
  segredo novo nesta frente)
- **Manifesto:** `C:\Users\vinicius.silva\Documents\Documentos\Projetos Dev\la_fabriquev2\docs\planejamento\2026-11-18\lote-2026-11-18-painel-operacao-fabrica.md`
- **Especificação canônica:** `C:\Users\vinicius.silva\Documents\Documentos\Projetos Dev\la_fabriquev2\docs\planejamento\2026-11-18\00-plano-T16-05-painel-operacao-fabrica.md`
- **Dependências:** frente `api` deste ticket (contrato e endpoints precisam existir).

## Prompt para o executor (Claude, Codex ou Antigravity)

1. **Conferência do workspace.**
   - `git rev-parse --show-toplevel`
   - `git branch --show-current` → `developer`
   - `git status --porcelain` → vazio (ou só arquivos deste planejamento)
   - Confirme que o commit da frente `api` (T16-05) já está em `developer`: `git log --oneline
     -10 developer` deve mostrar o commit dela. Sem isso, **pare e reporte** — não implemente
     contra um contrato que ainda não existe.

   Qualquer divergência: pare e reporte.

2. **Leituras obrigatórias.** `CLAUDE.md`, `docs/desenvolvimento/01-convencoes.md`, o `00-plano`
   acima (caminho absoluto, leia inteiro — principalmente "Contratos de API" e CA-10/CA-11) e os
   dois paths novos em `contracts/openapi.yaml` (já gravados pela frente `api`).

3. **Preparação.** `cd apps/web && npm ci` sem alterar `package.json` nem o lockfile. Stack de dev
   no ar (`make up`) para rodar contra a API de verdade ao conferir manualmente.

4. **Implementação.** Critérios desta frente: **CA-10 e CA-11** do `00-plano` (os demais já foram
   validados pela frente `api`).

   a. **Cliente HTTP** (`apps/web/src/lib/api.ts`): acrescente os tipos `OperationalEvent`
      (`seq, level, source, message, payload, created_at`) e uma função de histórico
      (`listOperationEvents(after, {level, source, q, limit})`) no mesmo estilo da função de
      eventos de execução já existente (linha ~1184 hoje).

   b. **Stream.** Olhe `apps/web/src/lib/useEventStream.ts`: hoje é específico da URL
      `/projects/{id}/executions/{id}/stream` e do tipo `ExecutionEvent`. Generalize-o para aceitar
      a URL como parâmetro (mantendo o hook atual funcionando, sem quebrar `RunPanel.tsx` nem os
      testes que já existem para ele) **ou** crie um hook irmão
      (`useOperationsStream`) copiando a mesma lógica de cursor/reconexão/backoff para
      `/api/operations/stream`. Escolha o que exigir menos mudança no hook existente; não refatore
      além do necessário para reutilizar a lógica de reconexão.

   c. **Tela** `apps/web/src/pages/OperationsPage.tsx` (nova): lista os eventos (nível, origem,
      mensagem, hora), com:
      - Filtro por nível (`info`/`warn`/`error`) e por origem (os seis valores do enum), aplicado
        no cliente sobre o que já chegou pelo stream e também repassado para a chamada de
        histórico inicial (`listOperationEvents`) — veja "Contratos de API" no `00-plano`: o
        histórico filtra no servidor, o stream não.
      - Busca textual livre (mesmo campo usado nas outras telas de busca do projeto, se houver um
        componente comum — confira em `apps/web/src/components/`).
      - Indicador de status da conexão (`connecting`/`live`/`reconnecting`), reaproveitando o
        padrão visual já usado em `RunPanel.tsx` para o mesmo propósito.
      - Use só tokens `--hud-*` de `apps/web/src/styles/tokens.css` — nenhuma cor fixa.
   d. **Rota e menu:**
      - Registre a rota (ex.: `/operations`) em `apps/web/src/App.tsx`, junto das demais rotas
        globais (`/decisions`, `/usage`, etc. — confira o padrão exato no arquivo).
      - Acrescente `{ name: "Operação", to: "/operations", icon: ... }` no array `navigation` de
        `apps/web/src/components/AppShell.tsx` (escolha um ícone já existente em
        `apps/web/src/components/Icon.tsx` que faça sentido; não crie ícone novo se um servir).
   e. **Fora do escopo desta frente:** qualquer endpoint novo, qualquer mudança no backend, ações
      sobre o evento (resolver/arquivar), filtro de nível/origem no servidor do stream.

5. **Registro de entrega e commit.**
   - Crie `docs/entregas/2026-11-18-T16-05-logs-operacionais-web.md` com o que foi feito, CA-10 e
     CA-11 com evidência (print/descrição do teste, não narrativa).
   - Atualize `docs/NEW MVP/07-CHECKLIST.md`: marque `T16-05` como `[x]` (ou ajuste a nota deixada
     pela frente `api`, se ela marcou `[~]` esperando esta parte).
   - Rode `make matriz` depois de editar o checklist.
   - Commit único na branch `developer`, **sem `Co-Authored-By`** nem atribuição ao agente, com
     código + docs + checklist juntos.
   - Sem push, sem PR, sem merge em `main`.

6. **Relatório final no chat:** SHA do commit, caminho do registro de entrega, validações
   executadas (`make lint`, `npm run typecheck` em `apps/web`, `make test-web`, `make e2e` se a
   tela nova justificar, build) com resultado real (PASS/FAIL/NOT_RUN), e para CA-10/CA-11:
   ATENDIDO, NÃO ATENDIDO ou NÃO VALIDADO, com a evidência. Diga explicitamente se testou a tela no
   navegador (golden path: eventos chegando ao vivo; borda: queda e reconexão do SSE, filtro
   combinado com busca).

## Prompt de revisão (sessão nova)

Use a skill `revisor-de-codigo` na branch `developer`, com a base = commit da frente `api` (T16-05)
e a especificação `C:\Users\vinicius.silva\Documents\Documentos\Projetos Dev\la_fabriquev2\docs\planejamento\2026-11-18\00-plano-T16-05-painel-operacao-fabrica.md`
(critérios CA-10 e CA-11).
