# FAC-012O — Journal de resultados

Data: 2026-10-03. Status: AWAITING_HUMAN. Baseline `0f96dd3`; ticket READY `76fb26e`; código `016a5ef088e5e46c178b4558ffdf4a6bec0d8bea`.

## Objetivo e funcionamento

Preserva resultado de workflow retornado/parado antes de qualquer envio final à API. `ResultJournal` fica em `WORKER_EXECUTION_ROOT/result-journal`, irmão de workspaces/snapshots, fora do namespace do código. Relatório público minimizado + IDs/fence + intenção checkpoint/outcome, schema estrito, até 128 KiB. Não persiste job completo, prompt, workspace, session ou credenciais; SHA do job vincula a intenção exata.

Arquivo 0600 em root 0700 owned pelo worker, sem symlinks nos caminhos canônicos; abertura NOFOLLOW/NONBLOCK rejeita symlink/tipo/tamanho/permissão. Publicação usa arquivo exclusivo temporário, fsync, hardlink sem sobrescrita e fsync do diretório. Apenas temporário criado pela operação é removido; relatório publicado permanece. Checksum detecta corrupção, não autentica alteração feita por invasor com acesso à identidade do worker. Esse root deve ser protegido operacionalmente.

Consumer lê journal antes do claim; entrada válida reenvia result/checkpoint/complete do mesmo attempt/fence autenticado, sem checkout/IA ou renovação de lease. Isso recupera redelivery após reinício mesmo com lease vencida, mas apenas porque o journal foi criado após workflow retornar e todos os checks confirmarem parada. API continua recusando owner/fence obsoletos. Falha ao gravar localmente impede upload/conclusão; falha ao enviar mantém evidência. Repetição igual é idempotente; alteração e corrupção falham fechadas. Não há limpeza automática.

## Checks e diff sanitizado

npm ci e db:generate locais, sem banco. Typecheck, testes, build, lint e git diff --check passaram: 266 testes (launcher 2, contracts 24, runtime 44, API 76, worker 112, web 8), lint 165 arquivos. Primeira rodada lint encontrou variável de FileHandle sem tipo; corrigida explicitamente, sem falha funcional posterior.

Teste real de filesystem cria journal, simula falha no upload, instancia journal/consumer novos e recupera o mesmo attempt sem claim/checkout/IA. Negativos cobrem disco indisponível, stop desconhecido, conteúdo divergente, job alterado, corrupção, symlink, tamanho/permissões e campos privados/outcome inventados.

Diff `git diff 76fb26e 016a5ef088e5e46c178b4558ffdf4a6bec0d8bea`: 5 arquivos worker, 408 inserções/14 remoções (journal/tests, processor/tests, composição main). Sem schema de banco, autenticação ou serviços alterados. Docs/lessons em commit separado.

## Limitações e próximo passo

Não preserva progresso quando workflow lança antes de retornar, não confirma writer desconhecido e não agenda automaticamente retry de job já FAILED no BullMQ. Recuperação foi comprovada por redelivery sintética, não por serviço/fila real ou reboot da VPS. Reinício só recupera se ocorrer redelivery; varredura de pendências/retry administrativo e snapshots de interrupção seguem pendentes. Artefatos completos/painel, provider/handoff, isolamento de contas, controles de run e gate documental continuam necessários.

Nenhum piloto/provider/login/API paga, serviço, banco, fila, push ou deploy usado. Gate permanece false.

## Rollback, docs e aceite

Reverter código com writer parado preservando journal/snapshots; nenhum banco alterado. Retenção/limpeza requer autorização e evidência preservada. README operação/runtime, índices/backlog/changelog/controle e lesson leases atualizados. Integração local autorizada após checks e árvore limpa, branch preservada na remoção de worktree. Aceite humano da revisão exata pendente; MVP não concluído.
