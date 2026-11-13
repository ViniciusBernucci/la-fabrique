# ADR-0007 — Adaptadores Jira e ClickUp simétricos

- **Estado:** ACEITA · **Tickets:** [T06-01](../NEW%20MVP/03-BACKLOG.md#t06-01), [T06-02](../NEW%20MVP/03-BACKLOG.md#t06-02) · **Épico:** E06 · **Decisões:** D03, D05–06, D50, D85, D88–92

## Decisão

1. **Uma interface, `integrations.Tracker`**, implementada por Jira (REST v3, busca `GET /search/jql` com `nextPageToken`) e ClickUp (API v2, páginas numeradas). Itens são normalizados em `Item`: ID remoto estável (Jira: ID numérico, não a chave, que muda ao mover), chave legível, tipo, `Subtask` e `ParentID`, estado com categoria comum (`todo`/`in_progress`/`done`), prioridade, rótulos, dependências, **versão** (marca de atualização remota, que vira a revisão observada) e URL.
2. **Dependência significa "este item depende de X"** nas duas plataformas. Jira: vínculo `Blocks`, em que o `inwardIssue` bloqueia o item; ClickUp: `task_id` espera `depends_on`.
3. **Erros tipados e mensagens fixas** (`auth`, `permission`, `not_found`, `rate_limited` com `RetryAfter`, `unavailable`, `invalid`). Nada de corpo remoto, URL ou credencial na mensagem. O adaptador **não repete**: quem chama decide, e escritas só são repetidas depois de conferir o estado no destino (T13-04/T06-03).
4. **Escrita segura por construção:** `Transition` consulta as transições válidas e recusa as demais **sem** chamar a escrita. Subtarefa não tem subtarefa e exige título; entradas viram caminho/JQL só depois de validadas por regex (JQL: chave do projeto e rótulo restritos e citados). Links exigem `https`.
5. **Idempotência onde a plataforma permite:** o link remoto do Jira usa `globalId` estável (repetir atualiza o mesmo link). O ClickUp não tem link nativo, então o link vira comentário e **repetir duplica**; a deduplicação é do chamador (T06-03).
6. **Segurança de rede:** sem redirecionamentos, timeout de 15 s, corpo limitado a 4 MiB. Jira só aceita `https://<site>.atlassian.net`. Segredo Jira `email:token` (Basic); ClickUp token puro no `Authorization`.
7. **ClickUp não distingue 401 de "sem acesso"** (devolve 401 para recurso fora do alcance); o erro `auth` cobre os dois e a mensagem diz "credencial recusada".

## Consequências

- T06-03 (subtarefas sem duplicação) e T06-04 (sincronização) usam esta interface e o ledger de efeitos (T13-04).
- Paginação do ClickUp usa página fixa de 100; `ListOptions.Limit` vale só para o Jira.
- Mapeamento de prioridade/estado para a fábrica fica na configuração do projeto (T05-02).
