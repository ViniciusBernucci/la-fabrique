# FAC-036 — Leitura e evidências da sessão

2026-10-09, Europe/Berlin. Base 38998ebf313625d2202dcb51c574803d78a3a1ef, branch feat/fac-036-instrucoes-md, worktree /home/vinicius/le-fabrique-fac-036. Workspace principal limpo na inspeção; sessões Codex abertas impedem edição concorrente nele. Worktree criada para este único writer.

Fontes abertas explicitamente: AGENTS.md, políticas numeradas integrais, roteiro, Manual Vivo, README, piloto, prompt inicial, guia de integração, ADR-003, perfil local e skill implementador-de-ticket, módulo configuração, feature control-settings, entidades, checklist/template de entrega, backlog e ticket FAC-036. Autoload/hierarquia instalada NÃO VERIFICADOS; fallback de leitura explícita utilizado. Referências históricas completas não carregadas por não serem necessárias ao incremento; nenhuma credencial/provider consultado.

Hashes SHA-256 das fontes normativas:

- AGENTS.md: 7ca4664cba5de95bb0d19d0b8131c078f3d917486b143afda60edc841a0665ca
- docs/00-governanca/01-POLITICA-IA.md: cf459b7d679a9796427d40b734dc0f2cabf5405de94c7ab05912574e1c3e6971
- docs/00-governanca/02-POLITICA-DOCUMENTACAO.md: 10d419ece5476593c37a092525d18f8c358f95f582eb5faf6f42540f1b1c022e

Saída observada de npm run test -w @le-fabrique/web: 24 arquivos, 67 testes PASS. npm run test -w @le-fabrique/contracts: 5 arquivos, 54 testes PASS. Primeira execução settings.service.spec.ts: falha de carregamento antes dos testes por cliente Prisma ausente após npm ci --ignore-scripts --offline. npm run db:generate gerou cliente local, sem acessar banco; reexecução registrada em settings-test.log. Primeira lint apontou controles ASCII no regex; substituído por refine com charCodeAt. Checks finais registrados separadamente. Nenhuma falha de baseline de produto identificada; instalação nova da worktree foi necessária para checks.
