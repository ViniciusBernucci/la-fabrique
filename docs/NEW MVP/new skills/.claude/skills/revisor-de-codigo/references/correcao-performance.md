# Critérios adaptados de code-reviewer

Origem: agente `code-reviewer`, preservado sem alterações. Esta referência é a adaptação portátil; aplique o protocolo do SKILL.md. Exemplos Laravel/Angular só se aplicam à stack encontrada. Regras específicas do CLAUDE.md precisam existir no projeto revisado.

## O que procurar ativamente

### Bugs lógicos

- `null`/`undefined` não tratado no caminho real do dado (não no hipotético).
- Off-by-one, comparação com tipo trocado (`==` vs `===`, string vs int em ID).
- Condição que nunca é verdadeira / nunca é falsa. `early return` que engole caso válido.
- Estado inconsistente em falha parcial: operação multi-etapa sem `DB::transaction`, ou
  transação que não cobre todas as escritas do fluxo.
- Race condition: leitura-decisão-escrita sem lock; duplo submit sem guarda.
- Erro engolido: `catch` vazio, `catch` que só loga e segue como se tivesse dado certo.
- Regra de negócio nova que não considera dado legado já existente na tabela.

### Casos de borda

- Lista vazia, único item, valor zero, string vazia vs `null`.
- Paginação/ordenação instável (ordenar por coluna não única).
- Fuso horário e conversão de data entre Angular e SQL Server.
- Concorrência de fluxo de aprovação: dois usuários aprovando a mesma PN.

### Performance — Laravel

- **N+1:** Eloquent dentro de loop, falta de `with()`, `count()` por item, accessor que
  dispara query.
- Over-fetching: `select *` em tabela larga, carregar relacionamento inteiro para usar um campo.
- Query sem índice em coluna filtrada/ordenada (aponte a coluna; não invente o plano de execução).
- Loop que faz `save()` por item em vez de `insert`/`upsert` em massa.

### Performance — Angular

- Subscription de longa duração sem encerramento no ciclo de vida: confirme se completa automaticamente antes de apontar vazamento.
- Ausência de `OnPush`: só é achado de performance com custo demonstrável; exigência de padrão pertence à revisão de qualidade.
- Função chamada no template (roda a cada change detection), sobretudo dentro de `@for`.
- `@for` sem `track`.
- **N requisições HTTP no lugar de 1 (batch).** Padrões:
  `forkJoin(itens.map(item => svc.algo(item)))`, loop de `.subscribe()`, ou qualquer coisa
  que dispare 1 request por item de lista que pode crescer (empreendimentos, unidades, itens
  de formulário). É sinal de que falta endpoint batch no backend (array no payload, 1
  transação). **[CRÍTICO] no perfil Intranet/PN**, conforme o histórico validado do agente: 200+ requisições simultâneas já travaram produção. Em outros projetos, confirme cardinalidade, concorrência e impacto antes de classificar.

### Cobertura

- Caminho crítico novo sem teste que o cubra (aponte qual caminho, não peça "mais testes").
- Teste que valida implementação em vez de comportamento — apontar, não reescrever.


Use o formato único de achados, relatório e plano definido no SKILL.md. Não aplique alterações.
