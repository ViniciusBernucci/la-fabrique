# ADR-0006 — Cadastro de projeto em rascunho e ativação

- **Estado:** ACEITA · **Ticket:** [T05-01](../NEW%20MVP/03-BACKLOG.md#t05-01) · **Épico:** E05 · **Decisões:** D06, D22, D29–30, D58, D60, D91, D94

## Contexto

O projeto precisa ser cadastrado por etapas, voltar ao assistente onde parou e só ser ativado com a configuração obrigatória válida. A validação de acesso remoto (T05-02) e a de recursos associados (T05-03) dependem de épicos ainda não entregues (E06 a E09).

## Decisão

1. **Rascunho no próprio projeto.** Dono, repositório, branch base, referência de conexão GitHub, plataforma da origem (Jira ou ClickUp), referência de conexão e identificador na origem ficam em colunas de `projects` (migration 5). Vazio significa "não preenchido". Há **uma** origem por construção (um único conjunto de colunas). Nenhum agente, workflow ou recurso global é associado na criação; isso só acontece por escolha explícita em T05-03.
2. **Credenciais só por referência.** Os campos guardam `connections.id` (FK `RESTRICT`), nunca o valor. A API devolve o id; nomes vêm de `GET /connections`.
3. **Persistência por etapa.** `PATCH /projects/{id}` recebe `expected_version` (conflito → 409) e substitui o grupo da etapa (`repository` ou `source`) por inteiro. Campos vazios são aceitos no rascunho e viram pendências; campos preenchidos são validados em formato (400 com o campo e o motivo). `wizard_step` guarda a etapa para retomar. Criar o projeto conclui a etapa "Dados" e o assistente retoma em "Repositório".
4. **Estado só muda por rotas próprias.** `PATCH` não aceita `state`. `activate` (rascunho → ativo), `suspend` (ativo → suspenso) e `resume` (suspenso → ativo) exigem `expected_version`, rodam sob bloqueio da linha (`FOR UPDATE`) e gravam a trilha na mesma transação. Transição fora do estado de origem → 409 `invalid_state`.
5. **Validação como relatório extensível.** `GET /projects/{id}/validation` devolve itens `ok`/`pending`, cada pendência com a **ação concreta**, e `not_verified` com o que ainda não é avaliado. Esta versão checa só preenchimento e a compatibilidade **local** das conexões (existe, ativa, tipo certo para o papel). `activate` e `resume` revalidam e respondem 422 `validation_failed` com pendências. T05-02 e T05-03 acrescentam itens ao mesmo relatório (acesso remoto, escopos, elegibilidade, workflows e agentes); até lá **ativar não prova acesso remoto**, e a tela diz isso.
6. **Configuração travada enquanto ativo.** Alterar `repository` ou `source` de projeto ativo → 409 `project_active`: a troca exige suspender, editar e reativar (revalidando), para nunca mudar sob execução. Nome, descrição e prioridade seguem editáveis.
7. **Trilha** (`project_changes`): criação, atualização (só nomes de campos), troca de origem (de/para: plataforma, id de conexão, identificador), ativação, suspensão e retomada. Nunca valores de segredo.
8. **Identificador (`slug`) imutável**; prioridade 0 a 4 (0 é válida; omitir usa 2).

## Consequências

- Ativar um projeto hoje significa "configuração obrigatória preenchida e coerente", não "pronto para executar". O worker não consome projetos ativos ainda.
- Rotas de projeto usam CSRF (`protected`), mas não reautenticação: não manipulam credenciais, só referências.
- Mudanças futuras de schema do projeto seguem em novas migrations; o relatório cresce sem mudar o contrato dos itens.

## Adendo — T05-02, parte da origem (Jira/ClickUp)

9. **Critérios na própria origem:** `eligibility_marker` (rótulo/tag), `eligible_statuses`, `priority_map` (prioridade da origem em minúsculas → 0..4) e `access` (`read` | `read_write`) fazem parte do grupo `source` do `PATCH`. Valores fora da faixa, marcador inválido (no Jira, sem espaços ou aspas, porque vira JQL) e escopo desconhecido falham na gravação com campo e motivo.
10. **Verificação remota registrada:** `POST /projects/{id}/source/check` usa a conexão do cofre e o adaptador do E06 para provar credencial, projeto/lista, estados e prioridades do mapa e listar até 20 itens elegíveis (marcador **e** estado elegível, sem concluídos), com a prioridade já mapeada. Falhas remotas voltam como passos `failed` com motivo (HTTP 200); **nada é escrito na origem**.
11. **Validade da verificação:** o resultado fica gravado com uma impressão digital (plataforma, conexão, **versão do segredo**, lista, marcador, estados, mapa, escopo). Qualquer mudança nesses itens, inclusive rotacionar o segredo, torna a verificação obsoleta. `activate` e `resume` exigem verificação **bem-sucedida e atual** (item `source.remote`), além de critério de elegibilidade (`source.eligibility`).
12. **Origem exclusiva:** uma mesma plataforma+lista não pode alimentar dois projetos ativos (mesma conexão ou mesmo destino); o item `source.exclusive` aponta o projeto que já a usa.
13. **Limites declarados:** escrita na origem não é provável sem escrever (passo `write` fica `skipped`; confirma no primeiro uso, T06-03). O repositório GitHub continua **não verificado** até o adaptador GitHub (T15-01) e aparece em `not_verified`.
