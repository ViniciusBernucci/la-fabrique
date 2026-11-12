> Leitura: [← Anterior](04-handoff.md) · [Índice didático](../02-INDEX.md) · [Próximo →](06-restore.md)

# Backup offline de evidências

Tooling IMPLEMENTADO em FAC-012AF, código presente em a2cc5e0; nenhum backup real executado nesta migração. Fonte: packages/runtime/src/evidence-backup.ts e scripts/evidence-backup.ts.

LFBACK01, AES-256-GCM com nonce aleatório 12 bytes/tag 16; chave externa binária de 32 bytes. Arquivo 0600, roots/staging 0700, dados/schema/hashes autenticados. Dump custom PGDMP, snapshots e result-journal congelados no mesmo checkpoint offline. Allowlist database.dump, snapshots/UUID e journal/UUID; recusa links/hardlinks/special/secret/path/owner/mode inseguros. Limites 64 MiB raw/96 MiB arquivo, 4096 files, 8192 entradas e 34 níveis; sem truncar. Whitelist não prova ausência universal de segredo no dump/código.

## Pré-condições e procedimento

Parar controle/dispatcher/worker e conferir árvore/stop fisicamente; flag de confirmação não detecta parada. Dump com pg_dump -Fc em ambiente autorizado e configuração privada externa, sem DSN/senha em argv. Paths abaixo são exemplos, não destinos reais.

```sh
npm run backup -- create --database-dump /privado/database.dump --snapshots /execucao/snapshots --journal /execucao/result-journal --key-file /chaves/backup.key --output /privado/checkpoint.lfb --confirm-services-stopped
npm run backup -- verify --archive /privado/checkpoint.lfb --key-file /chaves/backup.key
```

Chave fora do backup/roots; pais canônicos/privados. Guardar chave separada; perda impede recuperar. Sem copiar auth/home/checkout arbitrário/logs. Erro de I/O pode deixar material parcial para diagnóstico; não limpar automaticamente.

## Agenda, retention e evidência

Proposta: diário externo, 7 diários/4 semanais; artifacts 30 dias/auditoria 90 dias, RPO 24h/RTO 4h como metas. Não há scheduler/limpeza/serviço externo comprovados. Testes FAC-012AF históricos fizeram pg_dump/crypto/staging/pg_restore em DB sintética e restore Git real; não reexecutados aqui. Externalização/agenda/chaves/operabilidade continuam pendentes.

[Relato integral com limites e comandos](../09-entregas/2026/operacao/2026-10-03-FAC-012AF-backup-restauracao.md), [restore](06-restore.md), [referência operacional local](operacao/00-referencia-v2.3.md).
