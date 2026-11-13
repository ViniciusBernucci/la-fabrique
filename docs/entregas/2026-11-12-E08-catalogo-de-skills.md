# Entrega 2026-11-12 — E08: catálogo de skills e importação (T08-02 parcial)

**Ticket:** [T08-02](../NEW%20MVP/03-BACKLOG.md#t08-02) **parcial** (Git UNSUPPORTED). Decisões: [ADR-0014](../adr/0014-catalogo-de-skills-e-importacao-segura.md). Revisão independente: NOT_RUN.

## O que mudou
- **Banco:** migration 10 (`skills`, `skill_files`, `skill_imports`).
- **Backend:** `internal/skills` (manifesto, leitura segura de tar.gz/zip, `Ingest` com política, catálogo, execução, `NodeExecutor`); `policy.ExplicitDeny`; 7 operações de API.
- **Front:** menu **Skills** (cadastro manual, importação de arquivo, lista com origem/versão/hash/aviso de aprovação, trilha de importações).

## Verificação (2026-11-12)
| Verificação | Resultado |
|---|---|
| `go test -race ./internal/skills` (manifesto, travessia/links/limites em tar e zip, política, hash, execução determinística/híbrida/nó, script no sandbox Docker sob política) | PASS |
| `make test-integration` (as 4 modalidades, idempotência, replace/versão, arquivos guardados, pacotes maliciosos recusados e auditados, Git 501, 413, CSRF; rollback de 10 migrations) | PASS |
| vitest (65 testes; 6 novos), `tsc`, Biome | PASS (cópia no disco interno) |
| Importação por Git | **UNSUPPORTED** (T15-01) |
| IA real nas skills de IA/híbridas | **NOT_RUN** (executor de mentira) |

## Aceite → evidência
- "Importações ficam reutilizáveis e auditáveis": `TestPackageImportIsReusableAuditableAndVersioned` (arquivos com SHA-256, reimportação idempotente, `replace` → versão 2, trilha com `imported`/`unchanged`/`replaced`).
- "Pacote malicioso falha sem executar script": `TestMaliciousPackagesAreRejectedAndAudited` (travessia, link, script proibido, comando proibido, sem manifesto, manifesto inválido → 422; nada criado; recusas na trilha) e `TestMaliciousPackageNeverRunsDuringImport` (um script do pacote que criaria um arquivo marcador **não executa**, nem quando é aceito).
- "Exercitar skill híbrida e nó isolado" e "uma skill por modalidade, com agente e de forma independente": `TestCatalogSkillsRunIndependentlyAndAlongsideAnAgent`, `TestDeterministicAndHybridExecution`, `TestNodeExecutorRunsSkillNodesAloneAndDelegatesTheRest`.

## Rollback
`git revert`. A migration 10 tem `Down` (descarta skills, arquivos e trilha).
