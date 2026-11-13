# OPS-001 — Bootstrap confiavel do desenvolvimento local

Status: DONE

## Objetivo

Fazer o comando raiz `npm run dev` iniciar API, worker e painel com a configuracao do `.env` raiz, tolerar a corrida normal entre worker e API e orientar a aplicacao das migrations ja versionadas sem criar migration acidental.

## Atual e esperado

Atualmente, `npm run dev` inicia os tres processos sem carregar o `.env` raiz. Quando o operador exporta as variaveis manualmente, o worker ainda pode encerrar com `fetch failed` se tentar registrar antes da API. O guia local tambem usa `db:migrate`, que executa `prisma migrate dev` e pode gerar artefatos de migration durante um simples bootstrap.

O esperado e um unico fluxo reproduzivel: copiar `.env.example`, subir PostgreSQL/Redis, instalar dependencias, gerar o client Prisma, aplicar migrations versionadas com `db:deploy` e executar `npm run dev`. O worker deve aguardar a API por uma janela limitada e falhar claramente se ela continuar indisponivel.

## Escopo permitido e proibido

- Permitido: script raiz de desenvolvimento, `.env.example`, inicializacao do worker, testes unitarios e documentacao de infraestrutura/operacao.
- Permitido: retry limitado apenas para o registro inicial do worker, com atraso fixo, contagem observavel e sem registrar credenciais.
- Proibido: limpar Redis, remover jobs historicos, editar migrations, aplicar migration real, alterar dados, habilitar provider de IA ou executar cliente oficial.
- Proibido: retry infinito, mascarar falha posterior de heartbeat, mudar portas publicas, fazer deploy ou merge.

## Criterios de aceite verificaveis

1. `npm run dev` carrega o `.env` raiz sem exigir `source .env`.
2. `.env.example` usa `VITE_API_URL=/api`, permitindo ao navegador acessar a API pelo proxy do Vite inclusive ao abrir o endereco de rede da VPS.
3. O worker repete somente o registro inicial por no maximo 30 tentativas com intervalo de um segundo; sucesso encerra a espera e esgotamento preserva a falha.
4. Testes comprovam sucesso depois de falhas transitorias e falha final depois do limite, sem espera real.
5. A documentacao local usa `npm run db:deploy`; `db:migrate` permanece disponivel apenas para autoria consciente de novas migrations.
6. Falhas de jobs sinteticos antigos sao documentadas como historico da fila, sem exclusao automatica de dados.
7. Lint, typecheck, testes, build, validacao Prisma e `git diff --check` passam.

## Baseline e comandos de checks

- Base aceita: `c069926cb4b43a1519b5fd10b212b8fbe36cbfaa`, que registra o aceite da FAC-011D.
- Branch/worktree: `fix/ops-001-dev-bootstrap`, `/home/vinicius/le-fabrique-dev-bootstrap`.
- Baseline observado pelo responsavel: API inicia ao exportar `.env`; worker isolado registra, mas jobs sinteticos historicos falham por revisao-base indisponivel.
- Checks: `npm ci`, `npm run db:generate`, `npm run lint`, `npm run typecheck`, `npm test`, `npm run build`, `npm run db:deploy -- --help` ou validacao equivalente sem aplicar migration, teste controlado do launcher e `git diff --check`.

## Risco e orcamento

Mudanca local e reversivel, sem provider externo e sem custo adicional. Um worker por processo, sem novo writer, ate duas rodadas de correcao. O retry total fica limitado a aproximadamente 29 segundos de espera entre 30 tentativas. Nenhuma API de IA, extra usage, credito, autorecharge ou fallback e elegivel.

## Dependencias

FAC-000, FAC-004 e aceite da FAC-011D registrados. PostgreSQL e Redis locais continuam requisitos do bootstrap.

## Entregaveis e documentacao afetada

`package.json`, launcher em `scripts/`, worker e testes, `.env.example`, README raiz, documentacao de infraestrutura e operacao, indice, changelog, backlog e lesson de heartbeat/quiescencia.

## Evidencias e aceite

Ticket READY: `ab48aead8eec1680e7b15b5f3b7abbaa9340a04d`. Codigo verificado: `12018c3596efceb7bf4d171ca4787a81bfd9c136`. Relatorio: `documentacoes/infraestrutura/2026-10-02-OPS-001-bootstrap-desenvolvimento-local.md`.

Aceite explicito do responsavel registrado em 2026-10-02 para a revisao documental exata `7c1d1dcb44ab3464085aad729ae2b5dbdc473057`.
