# Checklist de risco

Aplique só às linhas alteradas e ao contexto imediato que elas afetam. Cada item é uma pista para
investigar, não um achado automático: confirme o cenário antes de classificar.

## Lógica e dados (qualquer stack)

- `null`/`undefined` não tratado no caminho **real** do dado.
- Comparação com tipo trocado (string vs int em ID), off-by-one, condição sempre verdadeira ou
  sempre falsa, `return` antecipado que engole um caso válido.
- Escrita em várias tabelas sem `DB::transaction`, ou transação que não cobre todas as escritas.
- Leitura → decisão → escrita sem lock; duplo envio sem proteção (dois usuários aprovando a mesma
  PN).
- Erro engolido: `catch` vazio, ou que só loga e segue como se tivesse dado certo.
- Regra nova que ignora dados legados já existentes na tabela.
- Bordas: lista vazia, um item só, zero, string vazia vs `null`, ordenação por coluna não única,
  fuso horário entre Angular e SQL Server.

## Laravel

- **N+1:** Eloquent dentro de loop, falta de `with()`, `count()` por item, accessor que dispara
  query. `save()` por item onde caberia `insert`/`upsert`.
- **SQL injection:** `DB::raw`, `whereRaw`, `selectRaw`, `orderByRaw` ou `DB::statement` com
  interpolação ou concatenação de entrada externa.
- **Mass assignment:** `$request->all()` em `create`/`update`/`fill`; campo sensível (`status`,
  `role`, valores) em `$fillable` sem controle.
- **Autorização/IDOR:** rota sem middleware de auth; `Model::find($request->id)` sem escopo do
  usuário; ação sobre recurso de terceiro sem Policy/Gate.
- **Exposição:** Model devolvido sem Resource, vazando coluna sensível; segredo, senha, token ou
  dado pessoal em log ou resposta.
- **Upload:** sem validação de MIME/extensão, gravado em caminho público, nome de arquivo vindo do
  usuário.
- **Outros:** `exec`/`shell_exec`/`Process` com entrada externa; `unserialize` de dado externo;
  `Http::get($url)` com URL do usuário; `redirect($request->input(...))`.
- **Banco:** migration sem `connection('sqlsrv_NEW_CRM')` explícita, ou que toque
  `sqlsrv_CRM`/`sqlsrv_CorporeRM` → **crítico**.
- **Contrato:** resposta fora do formato `{ message, errors }`; endpoint novo ou alterado sem
  contrato atualizado em `.claude/context/api-contracts/`.

## Angular

- **N requisições por item:** `forkJoin(itens.map(...))`, `subscribe` dentro de loop → **alto no
  mínimo**. Já derrubou produção; precisa de endpoint batch.
- Subscription longa sem `takeUntilDestroyed()`/`async` (confirme se ela não completa sozinha).
- Função pesada chamada no template, sobretudo dentro de `@for`; `@for` sem `track`.
- **XSS:** `bypassSecurityTrust*`, `[innerHTML]` com dado externo, `innerHTML` via `ElementRef`.
- Segredo no bundle (`environment.ts`, service): tudo o que vai para o front é público.
- Autorização só no front (botão ou rota escondidos) sem checagem equivalente no backend.
- Tipo TypeScript que não bate com o Resource do backend.

## Testes

- Caminho crítico novo sem teste: diga qual caminho, não peça "mais cobertura".
- Teste que nunca falharia: assert que recalcula o esperado do mesmo jeito que o código, ou mock
  do próprio objeto testado.
