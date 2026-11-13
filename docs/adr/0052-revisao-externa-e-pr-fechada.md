# ADR-0052 — Revisão externa e PR fechada

- **Estado:** ACEITA · **Ticket:** [T15-04](../NEW%20MVP/03-BACKLOG.md#t15-04) · **Épico:** E15 · **Relacionadas:** [ADR-0051](0051-observacao-do-merge-e-propagacao.md), [ADR-0049](0049-documentacao-viva-e-draft-pr.md), [ADR-0039](0039-fila-local-por-projeto.md)

## Decisão
1. **Descoberta sob demanda e só de autores autorizados:** `External.Discover` lê os comentários da conversa e da revisão da PR (`GitHubAPI.ListComments`, via `gitops.PRClient`) apenas quando chamado. Só contam logins da lista `Authorized` (vazia = ninguém); vazios e de outros autores são ignorados e contados. O texto é redigido (`security.Redactor`) antes de gravar. Leitura que falha não altera nada.
2. **Mesmo comentário não duplica trabalho:** cada comentário é registrado uma vez por identificador e conteúdo (`pr_comments`, chave única). Um comentário **editado** vira novo trabalho e dispensa a versão anterior ainda não tratada. Com a PR aberta, os comentários pendentes viram **um** item `pr_fix` na fila (`EnqueuePRFix`: vai à frente na próxima vaga elegível); comentários novos com o item ainda ativo entram no mesmo item.
3. **Correção só conta com prova:** `MarkFixed` exige documentação atualizada e checks aprovados; sem isso nada é marcado. Repetir não muda nada.
4. **PR fechada sem merge:** a entrega (`closed`) conserva PR, commits, documentação e comentários. Comentários novos ficam **retidos** (`held`), sem entrar na fila. `Offer` oferece as três ações, registrando a oferta uma única vez.
5. **Ações e histórico (`delivery_actions`):**
   - `cancel`: a entrega vira `cancelled`, itens de correção ativos são cancelados, comentários retidos; nada é apagado;
   - `reopen`: reabre a PR no GitHub, **confere** que ficou aberta e só então a entrega volta a `draft_open` e os comentários retidos voltam a ser tratáveis; recusa ou conferência que falha deixa tudo como estava;
   - `retry`: autoriza a próxima tentativa (limite `MaxAttempts`, padrão 3) sem tocar na anterior.
   Repetir a mesma ação não repete efeito. Cada uma grava quem decidiu e a observação.
6. **Decisão respeitada:** `Deliverer.Deliver` recusa uma tentativa já encerrada (integrada, fechada ou cancelada) e recusa a tentativa N+1 quando a N foi fechada ou cancelada sem a ação `retry`; `Observe` não reabre uma entrega cancelada.

## Limites
- Sem GitHub real (servidor HTTP de mentira) e sem rotas/tela; nada aciona `Discover` pelo worker ou painel (ponto D89 sem ligação).
- Quem executa a correção (agente na vaga) e chama `MarkFixed` ainda não está ligado ao worker; o item na fila e a prova exigida são o contrato.
- Reabrir cobre a PR da mesma branch; se a branch foi apagada o GitHub recusa e a entrega continua como estava (a saída é `retry`).
- Leitura de comentários limitada a 100 por origem (conversa e revisão).
