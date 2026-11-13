# FAC-012P — Diff e artefatos no painel

Data: 2026-10-03. Status: AWAITING_HUMAN. Baseline `2d5a9a3`; ticket READY `d00fe7c`; código `739cff001ad7c2ae53ca6719e2cbf92a11522683`.

## Funcionamento e contrato

Worker exporta último snapshot normal via ArtifactReader sob root privado `WORKER_EXECUTION_ROOT/snapshots`. UUID deriva do relatório, nunca de caminho do browser. Leitor recusa symlink/escape/tipo/owner/tamanho e verifica manifesto, patch Git binário e bytes untracked. DTO contém manifesto relativo, patchBase64 e files com path/dataBase64; sem paths host. Base64 canônico, limite JSON 64 KiB; leitor impõe teto de bytes originais 48.000 e 1.000 arquivos. Bundle maior bloqueia finalização, preservando journal/snapshots; não trunca.

Schemas são estritos em runtime; verificador neutro recebe ports de decode/hash para não importar servidor no painel. Patch/file/manifesto/contagem/bytes totais precisam corresponder. Padrões conhecidos de chave/Bearer/private key e arquivos sensíveis bloqueiam exportação, inclusive alguns headers de patch; heurística não garante detectar todo segredo arbitrário. Não se altera silenciosamente o diff/hashes. Conteúdo arbitrário de projeto continua não confiável.

Sequência: journal → result → artifact → checkpoint → complete. Falha de leitura/upload impede concluir; recuperação do journal repete com mesmo fence sem IA. Snapshot ausente não produz bundle fictício. API só persiste/consulta bytes, não lê filesystem nem aplica patch. POST interno `/api/internal/orchestration/attempts/:id/artifact` usa WorkerAuthGuard e worker/fence atuais, relatório correspondente obrigatório, digest imutável/idempotente em transação serializável, sem liberar writer.

GET administrativo `/api/runs/:runId/attempts/:attemptId/artifact` usa AdminAuthGuard e exige attempt pertencente ao run; retorna artifact nullable e verifica bytes armazenados. Painel carrega sob demanda, AbortSignal descarta resposta antiga, renderiza patch como texto escapado e oferece download JSON com bytes base64 após clique. Não executa/restaura bundle nem interpreta HTML.

Migration aditiva `20261003020000_fac_012p_execution_artifacts` adiciona artifact JSONB/artifact_digest TEXT; apenas versionada, não aplicada. Retenção sem limpeza automática.

## Checks e diff sanitizado

Typecheck, todos os testes, build, lint, Prisma validate (URL sintética, sem conexão) e git diff --check passaram. 289 testes: launcher 2, contracts 32, runtime 44, API 85, worker 117, web 9. Lint 172 arquivos. Primeira rodada lint recusou regex de controle; substituída por checagem de códigos dos caracteres. Sem outra correção funcional.

Testes reais Git/filesystem produzem patch/untracked via SnapshotManager e leem bytes exatos; corrupção e symlink são recusados. Contratos testam traversal/base64/limite/segredos; API testes simulam fence/owner/digest/snapshot/run; processor comprova ordem e falhas sem complete; renderização React escapa script. Não há E2E de navegador, DB/HTTP/fila real, reboot ou credencial de serviço.

Diff `git diff d00fe7c 739cff001ad7c2ae53ca6719e2cbf92a11522683`: 20 arquivos, 796 inserções/10 remoções; contracts/verificador/tests, campos/migration/API, leitor/protocolo/processor worker, painel/cliente/CSS/tests. Sem segredo real nem conta/piloto hardcoded.

## Limitações, rollback e aceite

64 KiB é teto inicial para tickets pequenos, não suporte a qualquer repositório/artefato. Bundle completo dentro do limite; artefatos grandes precisam transporte dedicado futuro. Documentação da entrega ainda não é gerada/gated. Retenção/backup operacional, snapshots de interrupção, retry de FAILED, providers/handoff/identidades e controles de runs permanecem pendentes. Gate real false; nenhum serviço/banco/provider/piloto/login/push/deploy usado.

Rollback: revert de código com writer parado, preservar journal/snapshots e dados artifact. Migration não aplicada não exige rollback DB; se aplicada futuramente não remover dados automaticamente. READMEs/domínios, índices/backlog/changelog/controle e lesson contratos atualizados. Integração local autorizada após checks/árvore limpa, branch preservada na limpeza. AWAITING_HUMAN até aceite do SHA exato; MVP não concluído.
