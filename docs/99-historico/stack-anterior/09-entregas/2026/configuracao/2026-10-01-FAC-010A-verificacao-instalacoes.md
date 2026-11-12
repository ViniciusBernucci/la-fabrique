# Entrega FAC-010A — Verificacao gerenciada de instalacoes

Data: 2026-10-01. Status: DONE; aceita na revisao `7c8d95eecc33488c43d7bf6d6166bedd6c143341`.

## Revisoes e funcionamento

Base aceita `f3872072b2770970814c694153c6583ad5abc057`; ticket `7cd2bf572bf32f3058f89afc80b4dca3d3065dd9`; codigo verificado `b6012bb262f0d3342dd1c9633ee9079089e76b76`.

O painel solicita verificacao por instalacao. API cria `ProviderVerification` e outbox na mesma transacao; dispatcher publica na fila dedicada; worker executa perfil fixo por provider, limita processo a 15 s/64 KiB e devolve somente estado, versao, modelos observados e mensagem controlada. Resultado e estado da configuracao mudam atomicamente. Reentrega ativa e conclusao identica sao idempotentes.

Codex usa `codex login status`; Claude usa `claude auth status --json`; Antigravity usa `agy models`. O executavel editavel no painel nunca vira comando. Mensagens de login com exit code zero continuam `AUTH_REQUIRED`. Ambiente remove chaves de API conhecidas antes do processo.

## Checks

- `npm ci`: 211 pacotes, 0 vulnerabilidades reportadas.
- `npm run lint`: passou, 90 arquivos.
- `npm run typecheck`: passou em todos os workspaces.
- `npm test`: passou, 74 testes em 20 arquivos.
- `npm run build`: passou; bundle Vite gerado.
- `prisma validate` com URL PostgreSQL sintetica: passou sem conexao/migration.
- `git diff --check`: passou.

Testes usam runners simulados; nenhuma verificacao real, inferencia, login, cota ou API foi acionada. Migration nao foi aplicada.

## Limites, proximo passo e rollback

Ainda nao ha login interativo, device code, discovery garantido de modelos nem adapter secundario. FAC-010B deve criar canal efemero autenticado antes de iniciar login; codigo/callback nao pode entrar em PostgreSQL/log. A tela usa polling de 3 s e preserva edicoes locais enquanto atualiza estado observado.

Rollback antes de migration: reverter `b6012bb262f0d3342dd1c9633ee9079089e76b76`. Depois de aplicada, usar migration compensatoria e preservar historico. Sem merge/deploy.
