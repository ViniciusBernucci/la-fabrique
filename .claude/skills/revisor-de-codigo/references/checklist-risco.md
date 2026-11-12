# Checklist de risco

Aplique só às linhas alteradas e ao contexto imediato que elas afetam. Cada item é uma pista para
investigar, não um achado automático: confirme o cenário antes de classificar.

## Lógica e dados (qualquer stack)

- `null`/`undefined`/valor zero não tratado no caminho **real** do dado.
- Comparação com tipo trocado, off-by-one, condição sempre verdadeira ou sempre falsa, `return`
  antecipado que engole um caso válido.
- Escrita em várias tabelas sem transação, ou transação que não cobre todas as escritas.
- Leitura → decisão → escrita sem lock ou sem controle de versão; duplo envio sem proteção (dois
  comandos concorrentes sobre o mesmo recurso).
- Erro engolido: tratamento vazio, ou que só loga e segue como se tivesse dado certo.
- Regra nova que ignora dados já existentes na tabela.
- Bordas: lista vazia, um item só, zero, string vazia vs `null`, ordenação por coluna não única,
  fuso horário (armazenar em UTC).

## Go (API, worker, migrate)

- **Contexto e concorrência:** `ctx` não propagado ou ignorado; goroutine sem término (vazamento);
  dado compartilhado sem sincronização; `defer` em loop; `rows.Close()`/`tx.Rollback()` ausentes.
- **Erros:** `err` descartado (`_ =`) em caminho que importa; `errors.Is/As` ausente onde o erro é
  traduzido; texto de driver/banco devolvido ao cliente.
- **SQL:** concatenação ou `fmt.Sprintf` de entrada externa em consulta (use parâmetros `$n`);
  consulta sem o escopo de projeto validado; N+1 (consulta dentro de loop); `UPDATE` sem
  `WHERE version = $n` em entidade com controle otimista.
- **Idempotência e efeitos externos:** efeito (push, PR, comentário, transição) sem registro de
  intenção/chave de negócio antes do envio; retry que duplica o efeito.
- **Autorização:** rota sem `requireSession`; recurso buscado só por ID, sem escopo; sessão não
  verificada no servidor (confiar só no front).
- **Segredos e logs:** segredo, token, cookie, `DATABASE_URL` ou dado pessoal em log, resposta ou
  erro; atributo sensível fora da redação do logger.
- **Arquivos e processos:** caminho vindo do usuário sem validação (`..`, symlink, absoluto);
  `exec` com entrada externa; leitura sem limite de tamanho; URL do usuário buscada pelo servidor.
- **Configuração:** variável nova sem validação em `internal/config` (falha diagnosticável) ou
  ausente de `.env.example`; valor padrão inseguro.

## Banco e migrations

- Migration **já aplicada** editada → **crítico** (crie uma nova). Migration sem `Down` ou sem
  teste de caminho em PostgreSQL real.
- Mudança incompatível (renomear/remover coluna em uso) numa única etapa, sem expandir → migrar →
  contrair.
- Falta de `NOT NULL`, `CHECK`, `UNIQUE` ou FK composta `(project_id, …)` onde a invariante deveria
  estar no banco; índice ausente para a consulta nova.
- Dado existente que a migration não converte (default silencioso errado).

## API e contrato

- Endpoint novo ou alterado sem `contracts/openapi.yaml` atualizado (há teste de contrato).
- Erro fora do formato `{"error":{"code","message","action","correlation_id"}}`.
- Paginação/limites ausentes em listas; payload sem limite de tamanho.
- Mudança incompatível de resposta usada pelo `apps/web`.

## React / TypeScript (`apps/web`)

- **N requisições por item:** `Promise.all(itens.map(fetch…))`, requisição dentro de loop ou de
  render → **alto no mínimo**; precisa de endpoint batch/paginado.
- Efeito sem limpeza (timer, listener, `AbortController`, stream SSE); dependências de `useEffect`
  que causam laço ou dado obsoleto.
- Estado de carregamento, vazio, erro e 401 não tratados (use os componentes de `States`).
- **XSS:** `dangerouslySetInnerHTML`, HTML montado de dado externo; segredo no bundle (tudo que vai
  para o front é público).
- Autorização só no front (botão ou rota escondidos) sem checagem equivalente no servidor.
- Tipo TypeScript que não bate com o contrato da API; `any` escondendo a divergência.
- Cor, fonte ou espaçamento fixos em vez dos tokens `--hud-*`; dados de demonstração no painel.
- `key` instável em listas; função pesada no render.

## Docker e infraestrutura

- Porta publicada que não deveria (banco/API fora de loopback); serviço sem healthcheck, sem
  `restart`, sem limite de recursos; rede `internal` mal usada.
- Container rodando como root, sem `cap_drop`, com socket do Docker ou volumes sensíveis montados.
- Segredo em imagem, `Dockerfile`, compose ou `.env.example`; `.env` rastreado.
- Imagem sem versão fixada; comando que só funciona na máquina do autor.

## Testes

- Caminho crítico novo sem teste: diga qual caminho, não peça "mais cobertura".
- Teste que nunca falharia: assert que recalcula o esperado do mesmo jeito que o código, ou mock
  do próprio objeto testado. Teste de banco com mock em vez de PostgreSQL real.
