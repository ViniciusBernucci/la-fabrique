# PLAN-ADM-001 — Proposta do painel administrativo da conta

Data: 2026-10-09 (Europe/Berlin). Papel: arquiteto-de-software, versão local adaptada. Branch: developer. Base: `ee4485bffb517247cd7b584653ad9ec1423f9202`; status inicial limpo. Decisão PROPOSTA; implementação PLANEJADA; análise documental realizada; aceite humano pendente. Sem commit, subagentes, migrations, chamadas de provider ou ativação de serviços.

## Objetivo, problema e critérios

Planejar painel da conta do usuário com perfil, segurança, planos, dados, equipe e permissões. O responsável esclareceu que a administração global da La fabrique será feita depois. Critério desta análise: separar software observado de propostas, vincular recomendações ao código e delimitar dependências/decisões para o Tech Lead. Não constitui ticket de implementação READY.

## Recomendação e comportamento proposto

[Proposta completa no handoff canônico](../../08-desenvolvimento/08-tech-lead-handoff.md#proposta-focal--painel-administrativo-da-conta). Perfil pessoal, organização/equipe, convites, papéis fixos e grants por projeto, segurança e auditoria primeiro; uso/limites/notificações depois; plano/cobrança conforme alcance comercial confirmado. Todos restritos à conta atual. Catálogo global de planos, gestão de todas as contas, workers/pausa global/infraestrutura fora do escopo.

Acesso atual usa token administrativo compartilhado, configurações globais e nenhuma identidade humana/membership no Prisma. FAC-037 já cadastrou integrações API/CLI; não há inferência agentiva por API. A evolução de conta precisa adaptar as rotas e ownership existentes, incluindo SSE/artifacts/jobs, antes de prometer segregação entre contas.

## Arquivos, decisões e contratos

Atualizados: handoff, backlog, changelog, índice principal e catálogo de entregas; este registro é o sexto arquivo. Sem novo capítulo didático: proposta focal incorporada ao capítulo existente, sem alteração necessária no mapa de ordem. Dados/API/eventos de produção inalterados. Entidades, famílias de contratos, autorização e migração estão apenas propostas no handoff. ADR-003 aceito preservado; propostas LF-MT não promovidas. ADR de identidade/ownership depende de consolidar a opção escolhida antes da implementação.

## Diff e evidências

Diff recuperável por `git diff` e arquivo novo em `git status --short`; nenhum patch de código ou segredo. Fontes primárias: AdminAuthGuard, App, DashboardLayout, SettingsService e schema Prisma; base identificada acima. Consultados perfil/skill adaptados, políticas, Manual Vivo, README, piloto, ADRs pertinentes, capítulos de configuração, tenancy, dados, segurança, API e backlog. Nenhuma inspeção de credenciais, DB ativo ou produção.

Leitura explícita de governança, autoload NÃO VERIFICADO. SHA-256:

| Path | Hash |
|---|---|
| AGENTS.md | 7ca4664cba5de95bb0d19d0b8131c078f3d917486b143afda60edc841a0665ca |
| docs/00-governanca/01-POLITICA-IA.md | cf459b7d679a9796427d40b734dc0f2cabf5405de94c7ab05912574e1c3e6971 |
| docs/00-governanca/02-POLITICA-DOCUMENTACAO.md | 10d419ece5476593c37a092525d18f8c358f95f582eb5faf6f42540f1b1c022e |
| skills/00-perfil-la-fabrique.md | 1c9823965d4f3015d5610f96a6482c4a88e2e38ede14bcf50a2be1a77ddf1621 |
| skills/arquiteto-de-software/SKILL.md | 3d7c43c57092d327ad7e5f35eeace7d07ae2ff171d06f920c59565cb377fabe0 |

Omissões intencionais: histórico integral, produção, credenciais, fixture/provider e pricing externo; desnecessários para esta proposta focal. Não há ticket de implementação associado. Saídas amplas de inventário tiveram truncamento; afirmações de estado foram confirmadas em leituras focais de código. As duas políticas foram lidas integralmente em chamada explícita.

## Verificações

| Critério | Base/configuração | Procedimento | Resultado |
|---|---|---|---|
| Estado de identidade/configuração | ee4485b, arquivos versionados | Leitura focal de guard, schema, App, SettingsService e navegação | PASS para inventário, sem comprovar operação |
| Baseline documental | ee4485b, árvore limpa | python3 scripts/validar-documentacao.py | FAIL preexistente: sete Markdown de handoff-FAC-035 fora do índice |
| Documentação final | Diff documental sobre ee4485b | python3 scripts/validar-documentacao.py | FAIL: mesmos sete erros do baseline; 398 Markdown, 2439 links e 601 snapshots verificados, sem novos erros |
| Whitespace/diff | Diff documental sobre ee4485b | git diff --check | PASS, exit code 0 |
| Suíte de software | Código inalterado | Não executada | NOT_RUN: edição documental |

## Segurança, riscos, limites e rollback

Não há novos controles implementados. Auth compartilhada e configuração global exigem migração; painel visual não produz isolamento. Plano comercial não autoriza gastos, não ultrapassa writer global e não transforma uso desconhecido em cobrança API. Integrações por conta e revogação de jobs exigem trabalho específico. Questões de login, MFA, titularidade das integrações e alcance comercial permanecem abertas.

Rollback documental: remover este registro e somente os trechos PLAN-ADM-001 dos cinco documentos editados, preservando mudanças posteriores de terceiros. Não há rollback de DB ou serviço. Reversão não ensaiada.

## Documentação e lessons

[Handoff](../../08-desenvolvimento/08-tech-lead-handoff.md#proposta-focal--painel-administrativo-da-conta), [backlog](../../08-desenvolvimento/09-backlog.md#plan-adm-001--arquitetura-do-painel-administrativo-da-conta), [changelog](../../04-CHANGELOG.md#2026-10-09--plan-adm-001), [índice](../../02-INDEX.md) e [catálogo](../00-README.md). Matriz de cobertura funcional, C4 e contratos atuais não alterados: nenhuma implementação/eficácia nova para cobrir. Sem lesson nova: nenhum conceito foi aplicado no software.

## Uso de IA, custos, pendências e aceite

Análise pela sessão Codex corrente; modelo/versão efetivos e billing não verificados por cliente nesta tarefa. Uso/custo incremental desconhecidos, nenhuma chamada de inferência/provider disparada pelo software. Sem contratação ou habilitação de API/extras.

Escopo da camada de usuário confirmado; disponibilidade para empresas clientes e significado de planos ainda pendentes. A proposta pode ser revisada pelo responsável e depois transformada em tickets pelo Tech Lead. Sem aprovação independente ou aceite estrutural inferidos; não declarar DONE.
