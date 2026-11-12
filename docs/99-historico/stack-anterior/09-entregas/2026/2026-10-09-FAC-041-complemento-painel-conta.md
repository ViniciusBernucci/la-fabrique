# FAC-039–041 — Complemento do painel administrativo da conta

2026-10-09, Europe/Berlin. Implementação na branch `feat/fac-038-painel-conta`, base `ee4485bffb517247cd7b584653ad9ec1423f9202`. Implementado; verificações automatizadas e browser fixture PASS. Revisão independente/QA e aceite humano pendentes. Continuação autorizada pelo usuário para terminar o painel da camada de usuário; administração global continua fora do escopo.

## Objetivo, problema e critérios de aceite

Completar a proposta do [handoff](../../08-desenvolvimento/08-tech-lead-handoff.md), respeitando [FAC-039](../../08-desenvolvimento/tickets/FAC-039.md), [FAC-040](../../08-desenvolvimento/tickets/FAC-040.md) e [FAC-041](../../08-desenvolvimento/tickets/FAC-041.md). O primeiro incremento [FAC-038](2026-10-09-FAC-038-painel-conta-usuario.md) permanece histórico; seu registro não constitui a verificação deste complemento.

## Solução e comportamento implementado

Projetos próprios da conta, grants por pessoa, definições, tickets, execuções, entrega/artefatos, aceite e controles passam pelas rotas da conta. Claim e renovação de lease revalidam autorizações; revogação exige pausa sem fingir término do writer. Legado sem organização permanece separado, sem atribuição arbitrária.

Integrações privadas têm segredo AES-GCM, grants por projeto, configuração de agentes e onboarding CLI segregado. Credencial GitHub é entregue somente ao supervisor do attempt autorizado, com lease/fence vigente, e usada no ambiente do subprocesso Git; não vai para especificação/modelo/exportação. API integrações são registros privados; a execução mantém o caminho de CLI por assinatura aprovado, sem fallback pago.

Uso agrega projetos, tickets/runs, espera por aceite, armazenamento persistido/reservado, tempo de execução com término confirmado e tokens efetivamente observados por provider. Ausência e agregações incompletas são explícitas; não calcula saldo de assinatura ou preço API. Notificações persistidas respeitam preferências e grants. Exportação paginada inclui perfil, projetos/definições, tickets, runs e tentativas/resultados sanitizados; gestores também exportam membros, atividade e registros de integrações. Artefatos têm download separado. Retenção configurável preserva auditoria/evidências de execução.

Perfil inclui avatar raster limitado. Titularidade exige senha/MFA e aceite do destinatário verificado; arquivamento exige reautenticação, nenhuma assinatura ativa e nenhuma execução sem término comprovado.

Email verificado, alteração de endereço e recuperação usam tokens hash de uso único. MFA TOTP tem anti-replay e recovery codes de uso único. SMTP usa outbox criptografado, retries e mensagem estável; entrega é pelo menos uma vez, sem promessa de exactly-once. Catálogo de planos versionado, quotas transacionais, checkout/portal Stripe opcionais e webhook raw assinado/idempotente, com reconciliação pelo estado atual do provider. O retorno do navegador não confirma pagamento. Resultado incerto de checkout conserva a intenção para reconciliar, sem criar cobrança nova automaticamente.

## Arquivos e decisões

`apps/api/src/account/`, migrações de conta e features, contratos `packages/contracts/src/account.ts`, componentes `apps/web/src/Account*`, ajustes de settings/orchestration e supervisor worker. [ADR-007](../../06-decisoes/ADR-007-conta-usuario-e-sessoes.md) permanece proposta implementada, sem aceite estrutural presumido. Não muda a stack aceita em ADR-003 nem cria isolamento contra código hostil/LF-MT/RLS.

## Dados, API, eventos e configuração

Migrações aditivas e ownership nullable preservam legado. FKs compostas protegem grants e cardinalidade de proprietário; notificações de ticket derivam de trigger. Configuração comercial, SMTP, chave de criptografia e Stripe constam somente em exemplos e no [runbook](../../07-operacao/01-deploy.md). Nenhuma credencial real lida, envio externo, cobrança, contratação, deploy ou migração de produção. Preços/limites comerciais dependem do responsável; catálogo vazio não inventa planos.

## Diff e artefatos

Código commitado em `1e6274aa6f4d5b8d9fbf6e28590c37fdafb8244e`. Evidências em [FAC-041](evidencias/FAC-041/), [manifesto de código](evidencias/FAC-041/codigo-hashes.json) e [estado dos gates](evidencias/FAC-041/estado.json). O manifesto exclui documentação/evidências para evitar circularidade; contém 57 paths explícitos de implementação, testes, migrations e exemplos de configuração. O commit preserva untracked e permite reproduzir o diff sobre a base; nenhuma credencial real incluída.

## Testes e evidências

| Critério / configuração | Comando ou procedimento | Resultado | Evidência |
| --- | --- | --- | --- |
| Suíte na revisão final, PostgreSQL descartável | `ACCOUNT_TEST_DATABASE_URL=<fixture account_test> npm test` | PASS, 648: scripts 7, contratos 58, runtime 65, API 231 (25 PostgreSQL), worker 217, web 70 | [tests.txt](evidencias/FAC-041/tests.txt) |
| Tipos/build monorepo + revalidação final afetada | `npm run typecheck`, `npm run build`, build API e typecheck/build/test web finais | PASS | [typecheck](evidencias/FAC-041/typecheck.txt), [build](evidencias/FAC-041/build.txt), [web final](evidencias/FAC-041/web-final.txt) |
| Exportação HTTP/privacidade/grants final | Teste PostgreSQL focal A/B e worker authority | PASS, 2 cenários; demais 23 não selecionados neste comando | [export-final](evidencias/FAC-041/export-final.txt) |
| Estilo | `npm run lint`, `git diff --check` excluindo evidências raw | PASS; um warning preexistente de optional chain em run-delivery.service.ts | [lint](evidencias/FAC-041/lint.txt) |
| Schema/migrations | Prisma validate e migrate status com DATABASE_URL explícita de fixture | PASS, 24 migrations aplicadas anteriormente somente no banco efêmero | [schema](evidencias/FAC-041/prisma-validate.txt), [migrations](evidencias/FAC-041/migrations-status.txt) |
| Browser Chromium, 1280×900 / 390×844 | 10 abas, cadastro, projeto, ticket, integração CLI registrada, JSON realmente baixado sem campos privados, overflow mobile | PASS, 15 verificações, zero exceções Runtime | [browser](evidencias/FAC-041/browser.json), [procedimento](evidencias/FAC-041/browser-procedure.mjs), capturas PNG no mesmo diretório |
| Dependências de produção | `npm audit --omit=dev --json` | PASS, zero vulnerabilidades | [audit](evidencias/FAC-041/audit-production.json) |
| Manual Vivo | `python3 scripts/validar-documentacao.py` | FAIL somente no baseline: 7 markdown fora do índice em handoff-FAC-035; nenhum novo erro de links/estrutura | [documentação](evidencias/FAC-041/documentacao.json) |
| Revisão independente / QA independente | Sessão distinta ainda não autorizada | NOT_RUN; browser/testes são verificações do implementador | [estado](evidencias/FAC-041/estado.json) |
| Produção / SMTP / Stripe / providers reais | Não acionados | NOT_RUN; fora dos checks de fixture | Runbook e estado |

A primeira verificação falhou em uma expectativa de teste sobre mensagem sanitizada do checkout; a segunda passou antes dos ajustes finais de troca de projeto e nova seção exportada. Browser subsequente identificou a seção `attempts` ausente do enum da rota; corrigida e coberta por testes HTTP e download real. `tests.txt` é a repetição completa depois dessas correções. Sem ampliar regressões de baseline para esta tarefa. Ambiente Chromium recebeu bibliotecas Ubuntu extraídas em diretório temporário, sem instalação global. API fixture usou Redis simulado e nenhum dispatcher/worker. Logs preservam bytes/whitespace de saída; check de whitespace de código/documentação exclui evidências raw. Logs usam .txt para serem recuperáveis no Git sem contrariar o ignore de .log. Serviços fixture/container encerrados após os checks; ambiente real preservado.

## Segurança, riscos e limitações

Risco R3: fronteiras de acesso, credenciais privadas e cobrança. Chave de criptografia requer backup/rotação operacional compatível com registros existentes. SMTP/Stripe reais e autenticação real dos providers não foram ativados; fixtures verificam contratos, não disponibilidade externa. Agregação de consumo limitada a 10.000 tentativas recentes, indicando parcialidade. Integrações não conferem automaticamente projetos nem acesso global. Não atribuímos projetos legados sem confirmação de titularidade. Review independente/QA permanecem pendentes conforme política local.

## Rollback

Reverter código exige cessar writers e preservar evidências. Migrações contêm dados de conta: não executar DROP ou remoção de ownership para desfazer a interface. Backup/restauração de dados/chave e compatibilidade API/worker devem ser avaliados antes de produção. Rollback de produção não ensaiado; ambiente descartável não prova reversibilidade operacional.

## Documentação atualizada

[Feature](../../04-features/01-control-settings.md), arquitetura/dados/segurança, módulo configuração, [contrato API](../../05-contratos/api/01-controle.md), entidades/rotas, [operação](../../07-operacao/01-deploy.md), [backlog](../../08-desenvolvimento/09-backlog.md), matriz, handoff, índice e changelog. Nenhum novo capítulo: ordem didática dos capítulos preservada.

## Lessons

Nenhuma lesson nova: aplicados padrões existentes de sessão, fencing, evidência, segregação e fonte única de documentação.

## Uso de IA e custos

Codex nesta sessão, implementador único, sem subagentes. Modelo/provider efetivo e consumo da assinatura não disponíveis como evidência local: unknown. Nenhuma contratação, API de IA ou autenticação paga executada. Fixtures SMTP/Stripe/GitHub e credenciais sintéticas não representam uso comercial.

## Pendências, próximos passos e aceite

Checks/evidências concluídos e commit de código registrado; documentação e evidências constituem commit separado autorizado. Revisão independente e QA não podem ser simulados pelo implementador; integração segue o gate local. Aceite humano da revisão exata pendente; tickets não promovidos a DONE. Configuração operacional real fica com o responsável do sistema, sem ativação implícita por esta entrega.
