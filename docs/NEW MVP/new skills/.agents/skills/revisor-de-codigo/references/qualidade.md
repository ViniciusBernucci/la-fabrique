# Critérios adaptados de clean-code-reviewer

Origem: agente `clean-code-reviewer`, preservado sem alterações. Esta referência é a adaptação portátil; aplique o protocolo do SKILL.md. Exemplos Laravel/Angular só se aplicam à stack encontrada. Regras específicas do CLAUDE.md precisam existir no projeto revisado.

## Eixo 1 — Aderência às convenções efetivamente adotadas

Leia as convenções locais, inclusive as seções relevantes do CLAUDE.md quando existir. A numeração abaixo pertence ao projeto de origem; não a presuma em outro repositório. Confira o código contra as regras lidas. Violações que aparecem com
mais frequência neste projeto:

**Angular** — `standalone: true` escrito à mão · falta de `OnPush` · `@Input`/`@Output` em vez
de `input()`/`output()` · injeção por construtor em vez de `inject()` · `*ngIf`/`*ngFor` em vez
de `@if`/`@for` · `ngClass`/`ngStyle` · estado derivado calculado à mão em vez de `computed()` ·
`mutate()` · rota de feature sem lazy loading · template-driven form sem pedido explícito.

**Laravel** — regra de negócio no Controller · `$request->validate()` no lugar de Form Request ·
query complexa no Controller · Model retornado direto sem Resource · falta de `declare(strict_types=1)` ·
parâmetro ou retorno sem tipo · `mixed` evitável · `catch (\Exception $e)` genérico · rota fora
de `api/v1/` · status HTTP não semântico · side effect inline em vez de Event/Listener.

**Caminhos** — arquivo criado fora do mapa da seção 1 do CLAUDE.md. Isso é **[ALTO]**, não estilo.

**Interoperabilidade** — tipo TypeScript que não espelha o Resource correspondente · contrato
novo não documentado em `.claude/context/api-contracts/` · erro de API fora do formato
`{ message, errors }`.

## Eixo 2 — Code smells

- **Duplicação** que já se repete 3+ vezes e deveria virar função/método/service compartilhado.
  Não proponha abstração por uma terceira ocorrência hipotética; demonstre o custo atual.
- **Nome ruim** — `data`, `temp`, `result`, `item2`, `$a`, `flag`, abreviação obscura, nome que
  mente sobre o que a função faz. Proponha o nome substituto, não só a crítica.
- **`any` sem justificativa** e cast forçado (`as unknown as X`) escondendo modelagem errada.
- **Função/método longo demais** — o corte não é linha, é número de responsabilidades. Aponte
  quais são e onde parte.
- **Aninhamento profundo** que resolveria com early return ou guard clause.
- **Comentário que descreve o que o código faz**, comentário obsoleto, código comentado
  esquecido, `TODO`/`console.log`/`dd()`/`dump()` deixados para trás.
- **Boolean parameter** e flag que faz a função ter dois comportamentos distintos.
- **Números e strings mágicas** que deveriam ser enum ou constante nomeada.
- **Abstração prematura** — violação de YAGNI: interface/factory/config para um caso único.
  Cobre também o oposto: copiar-colar onde a abstração já existe e é adequada.
- **Escopo estourado** — mudança que não pertence à tarefa (descontando formatação automática).

## Eixo 3 — Teste como documentação

- Teste que testa implementação (mock do próprio objeto sob teste, assert em detalhe interno).
- Nome de teste que não descreve comportamento.
- Caminho de negócio importante sem teste — descreva o caminho, não peça "cobertura".


Ao recomendar refatoração, distinga mudança mecânica de mudança em fluxo, contrato ou assinatura. Agrupe violações da mesma regra e apresente custo concreto; não imponha preferência pessoal.

Use o formato único de achados, relatório e plano definido no SKILL.md. Não aplique alterações.
