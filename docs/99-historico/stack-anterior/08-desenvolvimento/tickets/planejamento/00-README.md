> Leitura: [Índice didático](../../../02-INDEX.md) · [Próximo assunto →](../../../09-entregas/00-README.md)

# Planejamento atual

FAC-018 AWAITING_HUMAN: login oficial Claude e Google/Antigravity no painel; keyring VPS e consentimento real pendentes. [Evidências](../../../09-entregas/2026/configuracao/2026-10-05-FAC-018-login-claude-antigravity.md).

FAC-019 AWAITING_HUMAN: abas Contas, Integrações IA, Equipes. [Evidências](../../../09-entregas/2026/configuracao/2026-10-05-FAC-019-abas-configuracoes.md).

FAC-014 IMPLEMENTADO/AWAITING_HUMAN: [ticket](FAC-014-modais-funcionarios-github.md), [relatório](../../../09-entregas/2026/configuracao/2026-10-05-FAC-014-modais-funcionarios-github.md). Modais de funcionários/GitHub, sem alteração de contrato.
FAC-013 IMPLEMENTADO/AWAITING_HUMAN: [ticket](FAC-013-modal-contas-ia.md), [relatório](../../../09-entregas/2026/configuracao/2026-10-05-FAC-013-modal-contas-ia.md). Lista compacta/modal de contas IA; validação visual humana pendente.
OPS-009 IMPLEMENTADO/AWAITING_HUMAN: [ticket](OPS-009-reconciliar-migration-historica.md), [evidências](../../../09-entregas/2026/infraestrutura/2026-10-05-OPS-009-reconciliar-migration-historica.md). Reconciliação Prisma histórica não equivale a liberação de writer.
OPS-008: correção dos comandos Prisma e favicon implementada; atualização do banco dev pendente de reconciliação do histórico e prova de stop. [Ticket](OPS-008-bootstrap-migrations-telas.md), [relatório](../../../09-entregas/2026/infraestrutura/2026-10-05-OPS-008-bootstrap-migrations-telas.md).
OPS-007 IMPLEMENTADO/AWAITING_HUMAN: integração local autorizada FAC-012Z–AH, checks e remoção das worktrees incorporadas. [Ticket](OPS-007-consolidacao-worktrees.md), [relatório](../../../09-entregas/2026/operacao/2026-10-05-OPS-007-consolidacao-worktrees.md).
FAC-012AH IMPLEMENTADO/AWAITING_HUMAN: [ticket](FAC-012AH-admissao-capacidade.md), [evidências](../../../09-entregas/2026/controle/2026-10-03-FAC-012AH-admissao-capacidade.md). 528 testes + 10 PostgreSQL + 3 Redis. Lacunas internas auditadas implementadas; piloto não é dependência. Aceite/ativação separados.

FAC-012AG IMPLEMENTADO/AWAITING_HUMAN: [ticket](FAC-012AG-pausa-global.md), [evidências](../../../09-entregas/2026/controle/2026-10-03-FAC-012AG-pausa-global.md). 519 testes + 10 PostgreSQL + 2 Redis. Followup capacidade pré-claim, nenhum piloto.

FAC-012AF IMPLEMENTADO/AWAITING_HUMAN: [ticket](FAC-012AF-backup-restauracao.md), [evidências](../../../09-entregas/2026/operacao/2026-10-03-FAC-012AF-backup-restauracao.md). 501 testes internos + 8 PostgreSQL; próximo pausa global/kill switch de agendamento na UI.

FAC-012AE IMPLEMENTADO/AWAITING_HUMAN: [ticket](FAC-012AE-eventos-projeto.md), [evidências](../../../09-entregas/2026/controle/2026-10-03-FAC-012AE-eventos-projeto.md). 485 testes + 7 PostgreSQL; próximo interno backup/restauração, sem piloto.

FAC-012AD IMPLEMENTADO/AWAITING_HUMAN: [ticket](FAC-012AD-painel-operacao.md), [evidências](../../../09-entregas/2026/controle/2026-10-03-FAC-012AD-painel-operacao.md). 461 testes + 4 PostgreSQL; próxima lacuna de software: SSE previsto no ADR-003.

FAC-012AC IMPLEMENTADO/AWAITING_HUMAN: [ticket](FAC-012AC-writer-global.md), [evidências](../../../09-entregas/2026/controle/2026-10-03-FAC-012AC-writer-global.md). 445 testes + 3 PostgreSQL. Piloto externo não é requisito de conclusão do software.

FAC-012AB IMPLEMENTADO/AWAITING_HUMAN: [ticket](FAC-012AB-ensaio-integrado-mvp.md), [evidências](../../../09-entregas/2026/operacao/2026-10-03-FAC-012AB-ensaio-integrado-mvp.md). Revisão `1714347`; 438 testes; aceite e operação reais pendentes.
FAC-012AA IMPLEMENTADO/AWAITING_HUMAN: [ticket](FAC-012AA-confinamento-claude.md), [evidências](../../../09-entregas/2026/runtime/2026-10-03-FAC-012AA-confinamento-claude.md). Código `56f2e38`; 433 testes/checks; validação nativa do serviço e aceite pendentes.

FAC-012Z IMPLEMENTADO/AWAITING_HUMAN: [ticket](FAC-012Z-gate-documentacao-tecnica.md), [evidências](../../../09-entregas/2026/controle/2026-10-03-FAC-012Z-gate-documentacao-tecnica.md). Código `9c48dd9`; 422 testes; aceite exato pendente.

FAC-012Y implementa handoff configurado; AWAITING_HUMAN, sem prova operacional. Pendências de software: confinamento Claude e documentação técnica externa. [Evidências](../../../09-entregas/2026/handoff/2026-10-03-FAC-012Y-handoff-configurado.md).

FAC-012X (`345e8ef`, 399 testes/checks), AWAITING_HUMAN: espera por auth/cota/busy com stop/snapshot/persistência e retomada explícita. Próximos: handoff automático, confinamento Claude e docs técnicas. [Evidências](../../../09-entregas/2026/operacao/2026-10-03-FAC-012X-espera-provider.md).

FAC-012W (`2a1ae96`, 387 testes/checks), AWAITING_HUMAN: artefatos até 8 MiB JSON/6 MiB raw no fluxo worker/API/painel. Próximos: handoff/WAITING_PROVIDER, confinamento Claude e docs técnicas; operação/backup manuais. [Evidências](../../../09-entregas/2026/controle/2026-10-03-FAC-012W-artefatos-ampliados.md).

FAC-012V (`22f6375`, 380 testes/checks), AWAITING_HUMAN: identidade oficial privada por instalação no login/status/runtime. Claude escrita bloqueada; próximos: confinamento/handoff/WAITING_PROVIDER, docs técnicas e transporte maior. [Evidências](../../../09-entregas/2026/runtime/2026-10-03-FAC-012V-identidades-instalacoes.md).

FAC-012U (`9d4a4f1`, 370 testes/checks), AWAITING_HUMAN: recuperação administrativa apenas de finalização, sem reexecutar IA. Hard crash unknown continua conservadoramente bloqueado; próximos providers/identidades/handoff, docs técnicas e transporte maior.

FAC-012S (`be4e2ff`, 336 testes) e T (`42e7037`, 355 testes) implementados, AWAITING_HUMAN: comandos administrativos e retomada explícita de snapshot comprovadamente parado. Próximos: recovery de finalização, provider/identidades/handoff, docs técnicas e transporte maior. [Estado atual](../../06-controle-mvp.md).

FAC-012R implementado (`abfb033`), AWAITING_HUMAN, 315 testes/checks. Interrupções cooperativas preservadas; abruptas continuam bloqueadas. Próximos: comandos de run/retomada e providers/identidades/handoff/docs do projeto.

FAC-012Q AWAITING_HUMAN (`bfd8532`): relatório de entrega e aceite exato no painel, 306 testes. Documentação técnica dentro do projeto não é gerada por esse relatório; interrupção/recovery, controles e provider/handoff/identidades ainda pendentes.

FAC-012P implementado (`739cff0`), AWAITING_HUMAN: diff/bundle íntegro limitado no painel, 289 testes. Próximas lacunas: documentação/gate, aprovação/controles, interrupção/recuperação e provider/handoff/identidades.

FAC-012O AWAITING_HUMAN (`016a5ef`): journal pré-API e redelivery testados, 266 testes. Próximo: entrega de diff/artefatos e comandos explícitos de recuperação; interrupção antes do journal segue pendente.

FAC-012N implementado (`94b5d64`) e AWAITING_HUMAN: reconciliação após checkpoint parado, 257 testes. Próximo: journal de resultado local antes da persistência remota; demais lacunas seguem no controle do MVP.

FAC-012M está AWAITING_HUMAN: histórico e resultado estruturado no painel implementados em `14ae5ba`, 240 testes/checks verdes. Diff completo, recuperação/handoff/isolamento e comandos de run continuam pendentes no [controle do MVP](../../06-controle-mvp.md). FAC-012L liga workflow ao consumer no código, com gate false; descrições antigas abaixo de biblioteca/probe são históricas.

OPS-006 consolida OPS-005/K/L em developer e adiciona [controle atual do MVP](../../06-controle-mvp.md). Checks combinados passaram; integração local e limpeza não substituem aceites humanos. Próximo incremento: resultados recuperáveis/observáveis pelo software.

FAC-000 está DONE. O piloto externo foi adiado por decisão do responsável até o núcleo da plataforma estar pronto para validação.

FAC-001A, FAC-002 a FAC-009, FAC-010A/B/C, FAC-011A/B/C/D, FAC-003A, FAC-012A/B e OPS-001 estão DONE. O workflow ainda não está ligado ao consumer real. FAC-010 permanece `WAITING_PROVIDER`. O projeto externo continua indefinido e o ensaio FAC-012 permanece adiado.

FAC-012C está implementado em `a8f7a66dfc1b5c58a0596611dc300559504709ce` e aguarda aceite humano. Impõe caminhos graváveis explícitos no `SandboxRunner`, mas não ativa nem confina ainda a escrita do agente via CLI.

FAC-012D implementa escrita Codex restrita a `allowedPaths` congelados no snapshot e está `AWAITING_HUMAN`. Ver relatório operacional e evidências; consumer, provider real e piloto continuam fora de escopo.
FAC-012E implementa leitura interna versionada e sem segredos das contas/modelos configurados pela interface (`4aca1e1`) e aguarda aceite humano. O consumer permanece fora deste incremento.

FAC-012F implementado no commit `950f3b62651b1e918bae09d89996af0d95e9fbee`; aguarda aceite humano. Resolve Developer/Reviewer da configuração atual sem cache/fallback, mas ainda não está conectado ao consumer nem executa adapters.

FAC-012G implementado após a base `03d7a9b` e aguarda aceite humano. O workflow consulta rotas Developer/Reviewer independentemente, sem integrar a fila nem executar clientes.

FAC-012H implementado em branch isolada; aguarda revisão/aceite humano. O sandbox agora troca para uma raiz mínima e testes sintéticos bloqueiam leitura do host e tentativa de remontar o workspace.

FAC-012I está implementado e aguarda aceite humano: contexto e checks explicitamente aprovados por projeto persistem na definição versionada e são exigidos antes de READY; nenhum consumer/provider foi ligado.

FAC-012J implementado em biblioteca isolada; aguarda aceite humano. Prepara checkout em root privado a partir do SHA do evento e usa credencial GitHub oficial efêmera após provar keyring, sem expor token. Não foi integrado ao consumer nem exercitado com rede/credenciais reais.

FAC-012K implementa uma primitiva isolada de lease viva e cancelamento conservador; aguarda aceite humano e segue desligada do consumer, sem provider, checkout ou fila real.

FAC-012L implementado em `54bf483`; aguarda aceite humano. Consumer real integra checkout/workflow, rotas da interface e lease/fencing/checkpoint, desabilitado por padrão. Testes incluem cancelamento Linux do sandbox; identidade de serviço e providers reais seguem não verificados. Recuperação, artefatos no painel e handoff ainda não estão ligados ao consumer.

OPS-004 foi executado localmente: FAC-012D e FAC-012E–J foram integrados em `developer`, os checks combinados passaram e as worktrees D/OPS-004 foram removidas após prova de ancestry e limpeza. Branch refs foram preservadas. Aguarda revisão humana; não implica aceite dos tickets nem push/deploy.

OPS-005 preparou um caminho de serviço worker dedicado no host da VPS, onde o `systemd-run --user` do sandbox pode ser usado sem container privilegiado. O consumidor fixture foi removido do código e execução real segue desligada. O unit não foi instalado e a worktree aguarda revisão humana.

Resultados de fixtures sintéticas comprovam contratos e infraestrutura da fábrica, mas não contam como entrega ou aceite do piloto externo.
