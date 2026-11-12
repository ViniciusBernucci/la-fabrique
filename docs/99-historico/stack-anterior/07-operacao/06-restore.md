> Leitura: [← Anterior](05-backup.md) · [Índice didático](../02-INDEX.md) · [Próximo →](07-troubleshooting.md)

# Restaurar em staging novo

Tooling presente; ensaio real de serviço NOT_RUN nesta migração. Bloquear claims, confirmar parada física, selecionar backup/checkpoint/hash/versão compatíveis e chave externa. Destino não pode existir; nunca overwrite.

```sh
npm run backup -- restore --archive /privado/checkpoint.lfb --key-file /chaves/backup.key --destination /restauracoes/checkpoint-novo --confirm-isolated-destination
```

Exemplos de paths privados. CLI autentica/verifica bytes e restaura staging; não executa pg_restore. Importar dump só em nova instância PostgreSQL compatível/isolada, conferir migrations/índice/triggers/check constraints/cursors/outbox/evidências. SnapshotManager restaura patch/untracked apenas na base Git limpa exata, sem symlink/controle/traversal. Journal consumido precisa corresponder ao job. Falta de stop conhecido mantém BLOCKED_RECOVERY.

Reautenticar provider pelo fluxo oficial, sem cópia informal de auth. Token interno rotacionado por procedimento administrativo autorizado. RPO/RTO só após medir perda/tempo reais; roundtrip sintético histórico não prova storage externo/reboot ou recuperação após comprometimento. Não apagar staging parcial/evidências/chave por limpeza automática.

[Relato FAC-012AF](../09-entregas/2026/operacao/2026-10-03-FAC-012AF-backup-restauracao.md), [backup](05-backup.md), [sandbox/snapshot](../03-modulos/sandbox/00-README.md).
