# Entrega FAC-010B — Login efemero do Codex

Data: 2026-10-01. Status: DONE; aceita na revisao `d39412282d9401b2a22c70eb4b353883e559be4c`.

## Revisoes e funcionamento

Base aceita e registro do FAC-010A: `d772d127d213999a46dfaedb633b5dc70875be54`; ticket READY: `cc5b234208a1b105b5567be5d69554fade52247a`; codigo verificado: `4d979aa7045022020447114c0dd125dfe5dfadde`.

O painel oferece `Conectar assinatura Codex` somente para instalacao Codex habilitada em `AUTH_REQUIRED`. A API grava `ProviderOnboardingSession` e outbox na mesma transacao; o dispatcher publica na fila `le-fabrique.provider-onboarding`; o worker executa exclusivamente `codex login --device-auth`, com shell desligado, ambiente sem API keys, limite de 64 KiB e janela maxima de dez minutos.

URL e codigo sao extraidos em memoria, validados para HTTPS em dominio oficial OpenAI/ChatGPT e enviados ao endpoint interno autenticado. A API os guarda somente no Redis privado com TTL limitado pela sessao. Listagem, PostgreSQL, outbox e job carregam apenas IDs, provider, estado e timestamps. O browser mantem o desafio apenas no estado React, sem `localStorage` ou `sessionStorage`. Ao concluir, o worker executa `codex login status`; somente essa prova altera a instalacao para `AVAILABLE`. Conclusao remove o desafio e SIGINT/SIGTERM mata processos de login ativos.

O fluxo adotado segue a documentacao oficial de autenticacao do Codex, que indica `codex login --device-auth` para ambientes sem navegador: <https://learn.chatgpt.com/docs/auth>. O repositorio continua proibindo copiar cache de autenticacao, token ou `auth.json` ao controle.

## Checks e evidencias

- `npm ci`: 211 pacotes instalados; 0 vulnerabilidades reportadas.
- `npm run lint`: passou em 96 arquivos.
- `npm test`: passou, 81 testes em 22 arquivos.
- `npm run typecheck`: passou em todos os workspaces.
- `npm run build`: passou; bundle Vite produzido.
- `DATABASE_URL=postgresql://factory:synthetic@127.0.0.1:5432/factory npm exec --workspace @le-fabrique/api -- prisma validate`: schema valido, sem conexao ou migration.
- `git diff --check`: passou.
- Testes novos cobrem host oficial, desafio somente no Redis, TTL, outbox sem desafio, atualizacao por evidencia, expiracao e resultado sanitizado.

Fixtures usam URL/codigo sinteticos e runners simulados. Nenhum login real, browser externo, prompt, inferencia, cota, API ou cobranca foi acionado. A migration foi apenas gerada e validada; nao foi aplicada.

## Limites, operacao e rollback

Claude Code e Antigravity permanecem fora deste incremento: nenhum fluxo automatico seguro foi comprovado. O login ainda exige acao humana no site oficial e pode expirar; a configuracao local nao desliga extras na conta externa, que continua exigindo verificacao humana. Redis e endpoints internos precisam permanecer privados na VPS.

Antes de aplicar a migration, rollback e a reversao do commit `4d979aa7045022020447114c0dd125dfe5dfadde`. Depois de aplicada, criar migration compensatoria que remova primeiro a fila/uso e preserve historico conforme a politica operacional. Nao houve merge ou deploy.
