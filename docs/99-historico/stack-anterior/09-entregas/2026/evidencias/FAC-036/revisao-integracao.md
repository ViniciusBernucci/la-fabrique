# FAC-036 — Revisão e verificação após integração

2026-10-24, Europe/Berlin. Revisão `82bcfc7` (merge de `feat/fac-036-instrucoes-md` em `developer`), conferida também em `05a9877`. Sessão Claude Code distinta da sessão Codex que implementou o FAC-036; esta sessão resolveu os conflitos do merge em SettingsPanel/TeamsPanel, portanto não é revisão/QA independente formal.

## Escopo revisado

Diff `38998eb..40c9159` em apps/web (MarkdownInstructions, markdown-instructions, SettingsPanel, TeamsPanel, styles), packages/contracts (schemas e specs) e spec de SettingsService, mais a resolução de conflito com FAC-037: descrição breve e arquivo .md do FAC-036 mantidos junto ao AgentModelSelector do FAC-037, sem o select antigo de conta/modelo.

## Achados

Nenhum achado bloqueante.

- Leitura: extensão/nome validados pelo mesmo schema do contrato; tamanho limitado a 4 bytes por caractere antes de alocar; UTF-8 com `fatal: true`; BOM preservado (`ignoreBOM: true`); vazio/NUL recusados.
- Limite em caracteres usa `string.length` (unidades UTF-16), igual ao `max` do zod no servidor; cliente e servidor recusam o mesmo conteúdo.
- Contrato: campos novos opcionais; registros antigos válidos; instruções deixam de ser normalizadas por trim, preservando espaços.
- Nome do arquivo só aparece no atributo `download` e no texto da tela; não vira caminho no servidor.
- Observação (sem correção): MarkdownInstructions só libera `onBusyChange(false)` se continuar montado. Desmontar durante a leitura deixaria o salvamento bloqueado até recarregar, mas o modal desabilita Cancelar, × e Esc enquanto lê, então o caminho não é alcançável hoje.

## Verificação em browser

Chromium headless via CDP contra fixture descartável: PostgreSQL e Redis em containers temporários com migrations aplicadas, API e web locais com token fixture, sem worker, provider ou serviço externo. Bibliotecas do Chromium extraídas em diretório temporário, sem instalação global. Fixture e containers encerrados ao final; ambiente real e bancos de desenvolvimento não usados.

Resultado em [browser.json](browser.json), procedimento em [browser-procedure.mjs](browser-procedure.mjs), capturas [desktop](agente-preconfigurado-desktop.png), [skill](skill-desktop.png) e [mobile](agente-preconfigurado-mobile.png): 7 verificações PASS, zero exceções de runtime. Cobertas: salvar descrição e .md (BOM, CRLF, espaços e acentos) e conferir pela API; recarregar, reabrir e baixar bytes idênticos; vazio e .txt geram erro sem substituir o anterior; cancelar descarta a substituição; agente personalizado e skill persistem o arquivo; 390 px sem overflow.

## Checks após integração

`npm run typecheck` PASS; `npx biome check apps packages docs` PASS (1 aviso preexistente fora do FAC-036); testes contracts 59, web 73 e api 206 PASS (25 ignorados por exigirem PostgreSQL de teste).

## Pendente

Aceite humano da revisão exata. Revisão independente formal continua NOT_RUN.
