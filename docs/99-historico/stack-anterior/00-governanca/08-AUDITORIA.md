> Leitura: [← Anterior](07-PENDENCIAS.md) · [Índice didático](../02-INDEX.md) · [Próximo →](09-INVENTARIO.md)

# Auditoria da integração no repo real

Data 2026-10-07 Europe/Berlin; root de origem /home/vinicius/le-fabrique, developer, SHA a2cc5e0db0f30bf227dfc64300f74bde23a6f412. Único untracked prévio: reorganizacao-documental/, preservado sem escrita. Worktree documental /home/vinicius/le-fabrique-manual-vivo, branch docs/manual-vivo-inicial, mesmo SHA. Worker ativo observado; nenhum cliente codex exec/claude de execução ou filho do worker observado no preflight. Não inferimos configuração/stop do serviço; workspace documental separado e exclusivo, sem dispatch/DB/provider.

## Conciliação com evidência

| Conflito | Evidência primária | Tratamento |
|---|---|---|
| Laravel/Angular PDF versus stack | manifests React/Vite/Nest/TS e ADR-003 local ACEITO | PDF histórico intacto; stack atual não reaberta, nenhum código migrado |
| Repo ausente no pacote | apps/api, web, worker, contracts, runtime, Prisma/migrations/tests | Módulos/contratos reais preenchidos; declaração de ausência pertence ao espelho histórico |
| POST /runs versus tickets/:id/runs | controllers/control.service/outbox/orchestration | Nenhuma das duas existe: READY grava intenção, claim cria Run; sem alias/Idempotency-Key genérico |
| Estados PAUSED_BUDGET/BLOCKED e caminho ideal | enums Prisma/Zod, run-delivery.service | Ticket/Run/Attempt separados; PAUSED manual; aceite persistido VALIDATING→DONE; workflow AWAITING_HUMAN separado |
| FAC-002 aceito versus LF-MT | BACKLOG e FAC-002 preflight/aceite; propostas LF-MT do pacote | Aceite antigo intacto por revisão/escopo; gates posteriores não comprovados nem executados |
| ADR-003 duplicado | stack-typescript ACEITO 2026-09-29 versus multi-tenant PROPOSTO 2026-10-06 | Ambos preservados por nome/origem, colisão explícita; proposta não substitui aceito. ADR-001 não localizado |
| Auth/runtime versus broker | ProviderIdentity/perfis Codex/Claude reais; ausência broker/tenant no código | Separação de auth preservada; broker exclusivo/RLS são alvo, CLI sem ponte suportada incompatível com perfil futuro |
| Consumer ausente e recovery pendente em README antigo | main.ts/execution.processor/workflow/recovery/journal | Estado atual explicado nos módulos; texto integral por incremento em referências com errata, sem apagar cronologia |
| Teto 64 KiB versus 8 MiB | contratos/ArtifactReader/body-limits | 8 MiB JSON/6 MiB raw atuais; relatos anteriores mantêm teto de sua revisão |
| Backup/retention/estimates | tooling offline FAC-012AF, plano/dimensionamento | Preservar limites/comandos/ensaios históricos; 7/4, 30/90, RPO 24h/RTO 4h e esforços/metas são propostas não medidas |
| Índices, paths e política concorrentes | INDEX/guia/regra/documentação local | docs canônico, atalhos antigos após cópia integral; políticas comuns únicas e fallback explícito |
| Fontes de fornecedores vencidas | FONTES datado setembro e fontes do pacote | Proveniência datada; sem revalidar preço/plano/capacidade nesta migração, preflight atual pendente |
| UI menu e vídeo relatos cumulativos | DashboardLayout/HomeDashboard e FAC-025/030/031 | Home estática; abre recolhido; seta por estado. Relatos superados preservados no histórico |
| FAC-013 duas intenções | plano antigo “terceiro adapter” e entrega posterior “modal contas” | IDs não renumerados; catálogo antigo histórico, backlog local mantém evolução/ambiguidade explicitada |

## Inventário e cobertura

[Inventário real](INVENTARIO-REPO.json) inclui hashes, bytes, tipo, assunto, status, seções e destino; [mapa de seções](MAPA-SECOES-REPO.json) liga cada trecho ao destino/tratamento. Não apenas títulos: cada seção tem hash dos bytes e referência completa preservada. [Destinos](DESTINOS-REPO.json), [manifesto](MANIFESTO-FONTES.json) e [mapa navegável](10-MAPA-MIGRACAO.md). Código/schema/config/testes/artefatos inventariados sem .env/credenciais/dependências/outputs. Não há CI versionada localizada. Regras de fixtures preservadas em seus escopos.

Documentos locais foram lidos integralmente pelo processamento de migração e separados por seções; revisão semântica concentrou-se nas políticas, estado atual, contratos/ADRs, fluxos, módulos e operação frente às fontes de código citadas. Relatos/testes históricos são preservados integralmente e não recebem validação operacional nova. A conferência humana independente de todo o conteúdo e o aceite da revisão exata permanecem pendentes; não declarar auditoria independente aprovada.

Snapshots íntegros repo/pacote e PDF/ZIP preservados, sem editar para resolver conflitos. Conteúdo útil local tem capítulo/relato/lesson/ticket canônico antes do bridge. Propostas do pacote substituídas por AS-IS têm original íntegro e tratamento explícito no mapa; capítulos de direção adicionais permanecem ativos como PLANEJADO. Conteúdo antigo útil não é descartado por ser mais velho.

## Limites reais

Esta tarefa não comprova eficácia de sandbox/broker/tenant, login/financeiro, migrations no DB existente, stop do serviço, implantação/reboot/backup externo/RPO/RTO. Autorrelato do modelo e exit code são insuficientes. [Validação](11-VALIDACAO.md) registra somente comandos/resultados realmente executados; [pendências](07-PENDENCIAS.md) distingue lacunas resolvidas e restantes.
