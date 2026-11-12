# ADR-0033 — Sincronização manual e preflight da origem

- **Estado:** ACEITA · **Ticket:** [T06-04](../NEW%20MVP/03-BACKLOG.md#t06-04) · **Épico:** E06 · **Relacionadas:** [ADR-0018](0018-ledger-de-efeitos-externos.md), [ADR-0019](0019-escrita-na-origem-sem-duplicacao.md)

## Decisão
1. **Só lê quando pedem.** Há dois gatilhos: sincronização **manual** (`POST /api/projects/{id}/sync`) e **preflight** de uma nova execução (`Syncer.Preflight`, lê só o item). Não existe leitura periódica: com a fila ociosa, nada consulta a plataforma (provado com um relógio que avança um mês e um contador de leituras).
2. **Estado normalizado e gravado** por item: identidade remota, chave, título, descrição, estado e categoria, prioridade da origem e mapeada, rótulos, dependências, elegibilidade, versão remota e **hash do conteúdo**. O hash não inclui a versão remota nem edição local.
3. **Detecção única:** cada leitura compara com o gravado, registra as diferenças campo a campo (de/para), versão antes/depois, horário e origem, e o novo estado vira a referência. Repetir sem mudança não gera nada; uma mudança vista pelo preflight não reaparece na sincronização manual e vice-versa. Leituras concorrentes serializam por bloqueio do projeto (a segunda não vê diferença).
4. **Importação deduplicada** pela identidade remota (um item, uma linha), com paginação (até 50 páginas de 100; excedido, a leitura é declarada incompleta).
5. **Elegibilidade e prioridade reavaliadas** a cada leitura (estado elegível, não ser subtarefa nem concluído, marcador no preflight; prioridade pelo mapa do projeto).
6. **Itens que saem e voltam:** item que some da listagem vira `gone` (uma vez) e, se voltar, `reappeared`. **Leitura que falha** (limite, rede, permissão) fica registrada como `failed` e **não altera nenhum item**: uma leitura incompleta nunca dá item como removido.
7. **Edição humana não é sobrescrita em silêncio:** a prioridade editada localmente (0..4) prevalece; se a origem mudar a prioridade, abre-se um **conflito** (o valor local fica, o da origem é guardado), o preflight **não libera** o item e a decisão é `keep_local` ou `use_remote`. O conflito não se repete a cada leitura.
8. **Preflight decide:** libera só item existente, elegível, não concluído, não subtarefa e sem conflito aberto; diz o motivo quando não libera. Falha de leitura no preflight não libera e é registrada.

## Limites
- Só Jira e ClickUp. A leitura do GitHub (PR/merge) é do T15-03; o preflight não está ligado ao agendador (T10-02), e não há tela.
- Adaptadores reais NOT_RUN (fixtures e origem em memória/HTTP de mentira).
- A classificação do impacto de requisitos novos e a troca de plataforma são do T06-05.
