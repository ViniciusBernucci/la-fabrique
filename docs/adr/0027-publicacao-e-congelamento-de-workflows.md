# ADR-0027 — Publicação e congelamento de workflows

- **Estado:** ACEITA · **Ticket:** [T07-03](../NEW%20MVP/03-BACKLOG.md#t07-03) · **Épico:** E07 · **Relacionadas:** [ADR-0009](0009-documento-de-workflow-e-validacao.md), [ADR-0010](0010-motor-de-workflows.md), [ADR-0013](0013-catalogo-de-agentes-e-sobrescritas.md), [ADR-0021](0021-modelos-e-gateways-de-api.md)

## Decisão
1. **Versão publicada é imutável** (`workflow_versions`, gatilho que recusa UPDATE/DELETE). Guarda o grafo, o grafo expandido (o que o motor roda), as versões de subfluxo fixadas, o snapshot da configuração efetiva dos nós e o hash do conteúdo. Só o ponteiro `workflows.published_version_id` se move.
2. **Publicar valida tudo antes:** grafo e grafo expandido sem problemas; cada agente existe, a conta efetiva é compatível, o modelo é compatível (`models.Check`), as ferramentas e skills existem; subfluxos precisam ser do tipo `subflow` e estar **publicados** (a versão vigente de cada um fica fixada). Qualquer problema bloqueia (422 `publish_blocked`, com a lista `problems` por nó).
3. **Snapshot sanitizado:** a conta aparece só como referência de conexão; o JSON passa pelo redator do cofre. Skills congelam nome, id, versão e hash; ferramentas, nome, tipo e versão. O **hash do conteúdo** cobre grafo expandido, snapshot e fixações; publicar o mesmo conteúdo devolve a versão vigente (200, `unchanged`).
4. **Rascunho não executa:** `releases.StartExecution` só cria a execução a partir da versão publicada (`ErrNotPublished`), depois de revalidá-la, e grava `executions.workflow_version_id`. Publicar outra versão ou editar o catálogo depois **não altera** a execução existente nem o que foi congelado. Despublicar impede novas execuções e preserva as existentes.
5. **Revalidação em runtime respeita revogação:** `Revalidate` confere agente existente, conta ativa e compatível (revogada ⇒ `connection_revoked`), modelo ativo e compatível, ferramentas e skills ainda existentes. Não compara com o conteúdo atual do catálogo (o publicado é o que roda). Falha impede a execução (`RevalidationError`).
6. API: `POST /api/workflows/{id}/publish` (com `expected_version` do rascunho), `…/unpublish`, `GET …/versions`, `GET /api/workflow-versions/{id}` e `…/revalidate`.

## Limites
- O motor/worker ainda não consome `StartExecution` nem chama `Revalidate` a cada nó (T10); só a criação da execução revalida.
- Seleção de fluxo e subfluxo por regras é do T07-04; a tela de versões não foi feita.
- Skills e ferramentas congeladas são referências (nome, versão, hash): o conteúdo não é copiado para a versão.
