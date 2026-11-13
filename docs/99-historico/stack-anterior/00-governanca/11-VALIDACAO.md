> Leitura: [← Anterior](10-MAPA-MIGRACAO.md) · [Índice didático](../02-INDEX.md) · [Próximo assunto →](templates/00-README.md)

# Verificação da migração real — DOC-MV-001

Data 2026-10-07 Europe/Berlin; base a2cc5e0, branch docs/manual-vivo-inicial. Esta página substitui relato de checks do espelho; esse original permanece no snapshot do pacote. Não reusar testes históricos como PASS de HEAD.

## Checks executados

| Critério | Comando/procedimento | Resultado / evidência |
|---|---|---|
| Links, anchors, fences, índice, política comum e limpeza de bridges redundantes | python3 scripts/validar-documentacao.py | PASS no [resultado final](RESULTADO-VALIDACAO.json) |
| Originais intactos e cobertura por seção | python3 scripts/validar-documentacao.py --reference-root /home/vinicius/le-fabrique | PASS; snapshots/origens/hash de seção/destinos conferidos |
| Validador executável | python3 -m py_compile scripts/validar-documentacao.py | PASS; sem dependências externas |
| UI com novos paths de exemplo | npm run typecheck -w @le-fabrique/web | PASS: tsc -b --noEmit |
| UI/navegação/fluxos pertinentes | npm run test -w @le-fabrique/web | PASS: 18 arquivos, 46 testes, Vitest 5.0.2 |
| Build web | npm run build -w @le-fabrique/web | PASS: 135 módulos, Vite 8.3.1; log real |
| Lint repo | npm run lint | PASS com um warning useOptionalChain preexistente em fonte API não alterada; 257 arquivos, nenhuma correção automática |
| Diff e whitespace corrente | git diff --cached --check excluindo snapshots/relatos históricos/evidências brutas | PASS após corrigir EOF dos templates; conteúdo histórico e logs mantêm whitespace original |
| PDF/proveniência por página | PyMuPDF 1.26.7 temporário; leitura 45 páginas + renders 4/6/11/12/19/20/45 | PASS extração/proveniência, sete páginas inspecionadas visualmente; tabela perfis, stack histórica, rotas/estados e JSON de exemplo conferidos |
| Mermaid | Revisão textual dos níveis/canais/fronteiras e fences | PASS textual; render NOT_RUN após diagnóstico de libs locais, sem ampliar sandbox |
| Binários dos guias | codex --version; claude --version; agy --version | 0.159.2 / 2.1.285 / 1.2.17; não é teste de autoload/auth/elegibilidade |

[Checks web/lint/build](../09-entregas/2026/evidencias/DOC-MV-001/checks.json), [lint bruto](../09-entregas/2026/evidencias/DOC-MV-001/lint.txt), [build bruto](../09-entregas/2026/evidencias/DOC-MV-001/build-web.txt). Typecheck/test foram observados diretamente na sessão; registro local é resumo explícito, não log bruto inventado. Node efetivo 22.23.3/npm 10.9.9; README recomenda 22.20.0/10.9.3, manifests aceitam versão efetiva. Dependências existentes reutilizadas por symlink, sem modificar manifests/lockfile.

## Preservação, PDF e diagramas

[Manifesto de snapshots](MANIFESTO-FONTES.json) mantém 601 entradas repo/pacote. [Mapa por seção](MAPA-SECOES-REPO.json) cobre 2.444 trechos Markdown/preâmbulos com hashes; [destinos](DESTINOS-REPO.json) fecham realocação antes de bridges. Conferência opcional de referência compara originais da worktree de origem ainda intactos; após integração não comparar bridges a originais, verificar snapshots.

[PDF por página](PDF-PAGINAS.json) identifica PDF SHA d064f9c50132c593a452924c78c5b3168771a2a6024ee51e1847ec57b3a1d0c4 e texto/hash/medidas/blocos. Texto extraído no histórico pdf-verificacao, PDF íntegro mantido. [Renders de evidência](../09-entregas/2026/evidencias/DOC-MV-001/pdf-p11.png) mostram tabela de perfis; não houve recriação/reflow do PDF. Layout de sete páginas foi conferido, não se declara QA visual de todas as 45.

[Mermaid/diagnóstico](MERMAID-RESULTADOS.json): primeiro Chromium sem libatk; com bibliotecas já extraídas, libcups ausente. Duas tentativas, checkpoint/diagnóstico; nenhuma flag para relaxar sandbox ou instalação global. Render NÃO VERIFICADO, diagrams revisados por texto e canais contra código. Não chamar desenho de eficácia demonstrada.

## Revisão semântica própria

Conferidos ControlService/controllers, Prisma enums/cardinalidades/migrations, contratos Zod, AppModule, worker consumer/config/lease/journal/workflow/gate, adapters/runtime interface e templates: READY/claim real; enums por entidade; aceite VALIDATING versus result AWAITING_HUMAN; handoff interno versus resume; índice writer e outbox PG; Redis fila/cache; atual React/Nest/TS; proposals tenant/broker; rota física e ausência de resume na RuntimeAdapter; ADR colidente; UI/home/menus atuais; limite bundle e tooling backup/metas.

Os documentos e assets foram processados integralmente para preservar bytes/seções/hash; referência local completa permanece navegável com errata. Revisão semântica independente/humana da totalidade e aceite exato continuam PENDENTES. Este relatório não afirma revisão independente aprovada.

## Não executado e fallback

NOT_RUN: testes API/worker/DB/Redis/SEC reais, preflight/autenticação/financeiro, migration, operação/produção/reboot/backup externo, render Mermaid e nova sessão dos três agentes. Binário instalado não comprova elegibilidade/cobrança; não ler auth ou ampliar permissões para autoload. Abrir explicitamente guias e duas políticas no job, registrar paths/hashes/omissões. Instrução e versões em [carregamento](05-fontes-carregamento.md).

CI inexistente no inventário versionado; validador está disponível para execução manual, não pipeline instalado. Fontes externas datadas não revalidadas; conteúdo histórico não vira recomendação vigente de plano/preço. Nenhum merge/push/deploy/publicação/DONE de software por esta migração.

## Whitespace histórico preservado

O check integral staged inicialmente apontou EOF de cinco templates novos (corrigido) e whitespace em snapshots, patches/logs íntegros e hard breaks Markdown de relatos anteriores. Não alterar originais para forçar PASS. Check corrente exclui docs/99-historico e relatos/evidências antigas, incluindo explicitamente DOC-MV-001; resultados integrais guardados para diagnóstico. Isso distingue formato herdado de regressão documental atual, sem declarar check integral limpo.
