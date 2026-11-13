# FAC-021 — La fabrique e escala da home
Data: 2026-10-05. Status: IMPLEMENTADO / AWAITING_HUMAN.
Domínio: controle web e rótulos do runtime.
Base: `e87206f`; revisão de código `cde0cc624ac01f0a5bde1a5240f7eac1a1594659`, branch de preparação feat/fac-020-home-dashboard, integrada em developer.
## Objetivo e critérios
Pedidos explícitos: nome correto **La fabrique** em todas as referências do sistema; reduzir cena central para cerca de 60% do tamanho anterior. [Ticket READY FAC-021](../../../08-desenvolvimento/tickets/controle/tickets/FAC-021.md). Integrar em developer e remover a worktree conforme FAC-020A.
## Implementação e funcionamento
Nome corrigido em home/aria-label, projeto demonstrativo, login/cabeçalho, title HTML, favicon, mensagem da aba de autorização, clientInfo.title do catálogo, descrições do sandbox/perfil Codex e descrição do unit worker. Documentos/guias também receberam correção nominal identificada pelo FAC-021, inclusive documentos históricos; significado/decisões/tickets originais preservados. Evidências históricas, patches e hashes não foram reescritos.

A cena central usa width:60%, proporção 906/558, margin:20px auto e moldura discreta. Sua altura também diminui proporcionalmente. Menus/cartões mantêm largura normal e sobem após a cena menor. Até 760px, a cena usa espaço disponível menos 24px para legibilidade. Imagem permanece PNG enviado, incluindo a marca NÚCLEOS da referência. Nome correto no aplicativo: La fabrique.

Identificadores técnicos existentes (`@le-fabrique/*`, paths, fila, perfil, service e clientInfo.name) permanecem estáveis para compatibilidade. Nenhuma troca de pacotes, migração de dados ou lógica de permissões/autenticação. Descrição do unit foi alterada em arquivo, sem reinstalação/restart do serviço.
## Diff e evidências
[Patch sanitizado com contexto zero](../evidencias/controle/evidencias/FAC-021/implementation.patch), relativo e87206f→cde0cc6. [Desktop atualizado](../evidencias/controle/evidencias/FAC-021/desktop.png), [mobile](../evidencias/controle/evidencias/FAC-021/mobile.png), [script](../evidencias/controle/evidencias/FAC-021/browser-check.mjs), [resultado na worktree](../evidencias/controle/evidencias/FAC-021/browser-results.json) e [resultado na developer/5173](../evidencias/controle/evidencias/FAC-021/browser-results-developer.json).
## Verificação
- Typecheck web: PASS. Build web: PASS, Vite 8.3.1, 132 módulos.
- Testes web: 42/16 arquivos; runtime: 65/8 arquivos; worker: 215/26 arquivos. Total: 322, todos PASS.
- Lint raiz: PASS, 250 arquivos, mesmo único aviso preexistente useOptionalChain na API.
- Busca por nome humano anterior em apps/packages/infrastructure: nenhuma ocorrência ativa. Namespace técnico não conta como marca apresentada.
- Chromium 153.0.8010.12/Playwright 1.63.0: título e marca La fabrique, largura renderizada da cena =60% da coluna desktop, nenhuma API ao entrar, nenhum overflow em 1536/1280/1024/768/390/320px, busca, modais/Escape/foco e proteção de login PASS em 5174 e 5173.
- `git diff --check`: PASS. Build integrado completo em developer: PASS para contracts/runtime/API/worker/web na revisão cde0cc6; build web final atualizado após FAC-022.
- Login 401 é interceptado com credencial sintética; não comprova login/worker/providers reais.

Comando browser: `PREVIEW_URL=http://127.0.0.1:5173`, `PLAYWRIGHT_MODULE` apontando ao Playwright temporário e `LD_LIBRARY_PATH=/tmp/extracted/usr/lib/x86_64-linux-gnu`, executando o script do FAC-021. Nenhuma dependência da aplicação adicionada. Código e checks pertinentes ficaram iguais entre merge e evidência de browser integrado.
## Limitações e rollback
Home com dados simulados; referência central raster, com texto original na arte. Desktop reduzido; mobile conserva legibilidade. Reverter cde0cc6 desfaz nome/escala sem banco. Sem push, publicação ou migração. Aceite visual humano pendente.
## Documentação e lessons
READMEs raiz/controle/runtime/operação, política/guias com correção de nome, INDEX, CHANGELOG, BACKLOG, tickets, relatório, evidências e [baseline/review](../../../10-lessons/11-baseline-regressao-review.md). Nenhuma nova decisão de stack/contrato ou ADR necessário.
## Uso de IA e aceite
Codex da sessão, um writer e zero subagentes/handoffs/chamadas adicionais a providers. Não inspecionou credenciais nem alterou cobrança. Modelo/cota/consumo não comprovados. AWAITING_HUMAN; integração e remoção da worktree autorizadas, sem presumir DONE.

Atualização posterior FAC-022: código atual do template/fontes `4e042a1` integrado em developer; nome/cena do FAC-021 preservados. [Evidências atuais](2026-10-05-FAC-022-template-tipografia.md).
