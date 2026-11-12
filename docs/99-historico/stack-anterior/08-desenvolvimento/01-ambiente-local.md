> Leitura: [← Anterior](00-README.md) · [Índice didático](../02-INDEX.md) · [Próximo →](02-convencoes.md)

# Ambiente local real

Fonte: README a2cc5e0 e package.json. Comandos documentados, NÃO EXECUTADOS nesta migração. Não carregar .env real para validação documental.

## Desenvolvimento local

Requer Node.js 22.20.0, npm 10.9.3 e Docker com Compose. Para executar os serviços de dados e os processos em modo de desenvolvimento:

```bash
cp .env.example .env
docker compose -f compose.dev.yaml up -d
npm ci
npm run db:generate
npm run db:deploy
npm run dev
```

Antes de iniciar API, worker e painel, `npm run dev` recompila contratos e runtime pelo hook `predev`. A API usa a exportação CommonJS em `packages/contracts/dist`; o painel usa o código-fonte. Sem essa preparação, um build antigo pode rejeitar campos novos, como o nome do agente (`nickname`). Se a compilação falhar, os serviços não são iniciados. Após atualizar o código, reinicie `npm run dev` na raiz; executar somente o workspace web não atualiza a API.

`npm run dev` carrega o `.env` raiz automaticamente; nao e necessario executar `source .env`. O painel fica em `http://localhost:5173` e encaminha `/api` para a API em `http://localhost:3000/api`, inclusive quando o painel e aberto pelo endereco de rede exibido pelo Vite. `db:deploy` aplica somente migrations versionadas; use `db:migrate` apenas ao criar conscientemente uma nova migration.

Para validar a control plane, copie `.env.production.example` para `.env`, substitua todos os valores sintéticos e execute `docker compose up -d --build`. O proxy web fica em `http://localhost:8080`; API e Redis têm bind exclusivo em `127.0.0.1` para o worker host; PostgreSQL permanece sem porta publicada. O Compose não inicia worker. A preparação do serviço host está em [documentacoes/infraestrutura/README.md](../07-operacao/infraestrutura/01-referencia-v2.3.md); não instale/ative sem preflight e aceite operacional.

Checks locais: `npm run lint`, `npm run typecheck`, `npm test` e `npm run build`.


Health ready só prova conexão: conferir compatibilidade do schema e tentativas sem stop antes de aplicar migrations. Os relatos OPS-008/009 são históricos; esta migração não consulta banco nem altera serviço.
