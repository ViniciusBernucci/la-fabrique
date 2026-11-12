# FAC-012AF — Backup criptografado e restauração isolada

Data: 2026-10-03. IMPLEMENTADO / AWAITING_HUMAN. Código `7de9041e700b69f3850a90d4ddf5153886b49735`; baseline `c1aa79a`, READY `89b20b7`. Branch `feat/fac-012af-evidence-backup`, worktree exclusiva `/home/vinicius/le-fabrique-fac-012af`. Writer único; sem piloto/provider, não DONE automático.

## Funcionamento

Módulo offline packages/runtime/evidence-backup e comando npm run backup create/verify/restore. Formato LFBACK01: cabeçalho autenticado, nonce aleatório de 12 bytes, tag de 16 e AES-256-GCM com chave binária externa de exatamente 32 bytes. Sem passphrase embutida, chave ou credencial de banco no arquivo. JSON interno versionado com paths/modes/bytes/SHA-256/base64, validado em runtime. Artefato criptografado 0600, restore/stores 0700 e conteúdo verificado antes de criar destino.

Entradas explícitas: dump PostgreSQL custom PGDMP produzido offline, snapshots e result-journal privados/canônicos do UID atual. Roots disjuntos; key/output fora dos roots de evidência. Allowlist: database.dump; snapshots/UUID/manifest.json, tracked.patch e untracked relativos; result-journal/UUID.json. Rejeita links/special files/hardlinks, controle/NUL/traversal, nomes conhecidos de auth/env, diretórios inesperados, modos/owner inseguros. Leitura por FD NOFOLLOW/NONBLOCK verifica tamanho/mtime/ctime, orçamento total 64 MiB raw/96 MiB arquivo, 4096 files, 8192 entradas de árvore/34 níveis; sem truncamento. Manifesto/patch/untracked completos/hashes/modos e entryHash do journal verificados, sem afirmar que a intenção de journal corresponde a um job específico (ResultJournal faz isso ao consumir).

create exige confirmação explícita de controle/worker fisicamente parados; não prova parada por flag. Dump e filesystem devem pertencer ao mesmo checkpoint offline. verify autentica e confere todo conteúdo. restore exige confirmação de staging isolado e diretório inexistente; não sobrescreve nem executa pg_restore. Falha de autenticação/integridade/path precede escrita. Falha de I/O após escrita pode deixar staging/arquivo incompleto para diagnóstico; não há limpeza automática. Não exporta homes/login/checkout arbitrário ou logs; CLI nunca imprime erro bruto/path/JSON privado.

## Uso operacional preparado (não executado em serviço real)

Executar sob UID que possui arquivos, após parar controle/dispatcher/worker e conferir árvore/evidências. Preparar diretório privado para dump/arquivo e chave aleatória fora do backup (por exemplo openssl rand 32 com umask 077). pg_dump deve usar configuração de serviço PostgreSQL/PGPASSFILE privada externa, sem URL/senha em argv ou no backup. Registrar dump congelado e instante/checkpoint; não copiar arquivos ativos supondo consistência.

```sh
npm run backup -- create --database-dump /privado/database.dump --snapshots /execucao/snapshots --journal /execucao/result-journal --key-file /chaves/backup.key --output /privado/checkpoint.lfb --confirm-services-stopped
npm run backup -- verify --archive /privado/checkpoint.lfb --key-file /chaves/backup.key
npm run backup -- restore --archive /privado/checkpoint.lfb --key-file /chaves/backup.key --destination /restauracoes/checkpoint-novo --confirm-isolated-destination
```

Paths são exemplos absolutos; pais precisam existir/canônicos/privados e staging não pode existir. Armazenamento externo/rotação/agenda são operações autorizadas separadamente. Guardar chave separada do arquivo; perda da chave impede recuperação. Importar dump só em nova instância isolada de PostgreSQL compatível, conferir migrations/counters/evidências antes de qualquer ativação. Tokens oficiais não são restaurados por cópia informal; reautenticação humana permanece independente. Redis/fila é transporte e pode ser reconstruída a partir da autoridade/outbox; nunca liberar writer unknown por ausência de chave BullMQ.

## Evidências e checks

- Baseline ci --ignore-scripts, db:generate e build passaram; nenhum banco existente tocado.
- npm run typecheck, npm run lint, npm run build, git diff --check passaram na revisão final. Runtime build adicional inclui novo teste Git. Lint 227 files, warning optional-chain anterior único.
- Suíte raiz final: 500 passou (scripts 6, contracts 45, runtime 64, API 163, worker 190, web 32); teste Git adicional validado com runtime completo final 65 passou. Total único 501 testes internos. Nenhum check anterior ficou invalidado por mudança de implementação não verificada: runtime final/compile/lint cobre core após ajustes.
- npm run test:postgres: 8 passou/0 skipped, banco/container efêmero network none/tmpfs/512 MiB/1 CPU, todas migrations reais. pg_dump -Fc real → arquivo criptografado → stage byte-equivalente → pg_restore em outra database sintética do mesmo container preservou contagem de project_events, cursor e proteção global. Nenhum restore existente/deploy.
- Snapshot Git real: capture patch/untracked → encrypt → stage → SnapshotManager.restore em worktree limpa do SHA exato preservou texto e binário. Demais testes: crypto corruption/wrong key, authenticated traversal/duplicate/NUL, symlink/hardlink/secret/path extra, snapshot inválido, sparse file oversized, confirmações e overwrite; CLI negativo não vaza argumentos privados.
- Duas rodadas de correção: CLI top-level await incompatível CommonJS/declarations ainda sem rebuild, corrigida com main async/build; import opendir ausente após leitura incremental, corrigido e suíte revalidada. Ajuste de formatação evita regex de controles via charCode. Falhas de incremento, não regressão de baseline.

Logs efêmeros /tmp/fac-012af-{tests-final,runtime-tests-last,typecheck-final,lint-last,build-final,runtime-build-last,postgres-final}.log. Diff sanitizado: git show 7de9041 -- package.json packages/contracts/src/index.ts packages/runtime scripts/evidence-backup.ts scripts/evidence-backup-cli.test.mjs scripts/global-writer.postgres.test.mjs (8 files, 877 inserções/4 remoções). Somente chaves/dados sintéticos efêmeros. Containers/dados de teste removidos, evidências de serviço preservadas.

## Limites, rollback e aceite

Não comprova storage externo/RPO/RTO/reboot ou consistência enquanto processos escrevem. Assinatura PGDMP é verificação de formato inicial; validade integral SQL vem de pg_restore isolado, não apenas AES/hash. Segredos eventualmente presentes no dump/código continuam dados privados criptografados: whitelist não é scan universal de conteúdo. Nenhuma chave de serviço/provider real lida ou exportada. Modelo/uso reais não observados; zero IA/API/extras/login/push/merge/deploy.

Rollback reverte tooling mantendo arquivos, chave separada, formato e relatórios. Não apagar evidência/keys por cleanup. Aceite humano exato pendente. Software de backup independe de piloto; próxima lacuna interna auditada: pausa global/kill switch de agendamento pela interface, hoje controles são por run e gate de execução é env.

## Docs/lessons

Runtime/operação/infraestrutura/planejamento, README, INDEX, backlog/changelog, CONTROLE/handoff/ADR-003 e lesson sandbox/snapshot atualizados. Conceito aplicado: cifrar não basta; cópia congelada, autenticidade/hash/allowlist, staging exclusivo e restore real são fronteiras distintas.
