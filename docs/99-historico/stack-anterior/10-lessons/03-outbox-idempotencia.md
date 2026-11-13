> Leitura: [← Anterior](02-contratos-runtime-monorepo.md) · [Índice didático](../02-INDEX.md) · [Próximo →](04-contexto-deterministico-limites.md)

# Outbox idempotente e versão otimista

Atualizar um agregado e publicar trabalho são efeitos distintos. No FAC-003, a passagem do ticket para `READY` e a criação de `ticket.ready.v1` ocorrem na mesma transação Prisma. Assim, uma falha não deixa o ticket pronto sem a intenção persistida de despacho.

Retries também precisam de identidade estável. A outbox usa `deduplication_key = ticket:{ticketId}:ready` com índice único. Se o cliente repetir a requisição depois de perder a resposta, a API encontra o ticket já `READY` e o evento existente, retornando o estado atual sem criar outro evento.

O campo `version` protege contra edição baseada em leitura antiga. A atualização exige `id`, estado `DRAFT` e `expectedVersion`; apenas uma transação consegue incrementar para a próxima versão. A chave de outbox evita duplicação da intenção, enquanto a versão evita sobrescrever uma mudança concorrente. O futuro consumidor ainda precisará de sua própria idempotência: outbox única não prova processamento único.

O FAC-011 reutiliza a parte de versao otimista sem criar outbox: `PUT /api/settings` atualiza o singleton somente quando `expectedVersion` ainda corresponde. Configuracao nao dispara trabalho neste incremento, portanto inventar um evento seria efeito adicional sem consumidor. A licao e separar protecao contra sobrescrita de necessidade real de despacho; os dois mecanismos podem coexistir, mas nao sao sinonimos.

No FAC-011D, preparar um PR tambem nao cria outbox: e apenas um artefato revisavel com versao e digest. Somente a segunda requisicao `approve`, autenticada e vinculada ao payload exato, muda o estado e grava a intencao na mesma transacao. No consumidor, `gh pr list` reconcilia head/base antes e depois de um create incerto. Assim, idempotencia local, gate humano e reconciliacao do side effect remoto cobrem falhas diferentes.

FAC-003A acrescenta uma quarta fronteira: deduplicar uma intencao impossivel ainda produz uma falha unica, mas inutil. Por isso a promocao valida o SHA-base antes de atualizar ticket/outbox, e o dispatcher recusa eventos legados nulos antes da fila. A mudanca de `baseRef` usa comparacao otimista pelo valor previamente lido. Elegibilidade, concorrencia, deduplicacao e processamento idempotente sao protecoes complementares, nao substitutas.

FAC-012AE aplica outra necessidade além de publicação eventual: ordem de commit para cursor. append_project_event incrementa linha por projeto sob lock transacional; BIGSERIAL poderia alocar N antes de N+1 e publicar N depois que cliente já avançou. Eventos minimizados invalidam consultas HTTP atuais, não autorizam trabalho. Teste PostgreSQL confirma rollback e concorrência; project-events.ts valida projeto/ID e ignora duplicatas sem regredir, guarda token só no cabeçalho. [Evidências](../09-entregas/2026/controle/2026-10-03-FAC-012AE-eventos-projeto.md).
