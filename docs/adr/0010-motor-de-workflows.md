# ADR-0010 — Motor de execução de workflows

- **Estado:** ACEITA · **Ticket:** [T07-02](../NEW%20MVP/03-BACKLOG.md#t07-02) · **Épico:** E07 · **Decisões:** D09, D13, D44–45, D58–60, D64

## Decisão
1. **Motor puro e determinístico** (`workflows.Engine`): percorre o grafo **um nó por vez** (escritor único por projeto), sem banco nem relógio próprios (`Now` e o estado são injetados). Mesma entrada produz exatamente o mesmo estado (teste de determinismo).
2. **Estado serializável** (`RunState`, JSON): fila de caminhos ativos, variáveis, contadores de loop, chegadas em convergências, esperas com prazo, decisões pendentes, fins alcançados e rastro. Com o estado e o grafo congelado a execução retoma exatamente de onde parou (espera durável retomada por outro motor, em outro instante).
3. **Execução de trabalho por interface** (`Executor`): agente, skill, script, checks, Git e docs são delegados; o motor só sabe rotear `out`/`error`. As saídas ficam em `Vars[<id do nó com _>]` e em `last`. Os executores reais pertencem a E09, E11, E13 e E15.
4. **Controle:** `condition` (true/false), `loop` com **limite obrigatório** (1 a 100; ao atingir o limite segue por `done` e marca `loop_<id>_exhausted`), `wait` (prazo persistido; só segue quando `Now >= prazo`), `human_decision` (pausa até `Decide`; decisão repetida ou inválida é recusada), `fork` (executa `a` e depois `b`, **sem concorrência**), `converge` (`any`: cada chegada continua; `all`: espera os caminhos **ativos** e segue uma vez quando nada mais está em andamento, ignorando ramos não percorridos).
4b. **Erros:** falha do executor ou porta `error` segue a rota de erro; sem rota, a execução falha citando o nó. Expressão impossível de avaliar falha a execução com o nó. Limite de 10 000 passos.
5. **Expressões** (`ParseExpr`): `||`, `&&`, `!`, `== != < <= > >=`, literais e caminhos de variáveis; sem funções nem efeitos; variável ausente vale `null`. O validador acusa expressão inválida com nó e campo (`invalid_expression`).
6. **Subfluxos por expansão em linha** (`Flatten`): os nós `subflow` são substituídos pelo grafo do subfluxo (IDs prefixados `<nó>/…`; início vira passagem; fim de sucesso continua em `out`; fim de falha continua em `error` ou permanece como falha). Assim há um único grafo e um único estado a persistir/congelar. **Recursão** (direta ou indireta) e **profundidade** acima de 5 são recusadas na expansão; subfluxo inexistente ou com problemas também.
7. `NewRun` recusa grafo inválido (inclui loop sem limite) e grafo ainda com `subflow`. Na validação de grafo achatado aceita nós inalcançáveis ou sem entrada (um subfluxo que sempre falha deixa a continuação sem caminho).

## Limites
- Não há persistência de execuções no banco, nem scheduler, nem publicação (T07-03, T10, T12).
- `any` pode executar o que vem depois mais de uma vez quando dois caminhos chegam; para junção única use `fork` + `converge all`.
