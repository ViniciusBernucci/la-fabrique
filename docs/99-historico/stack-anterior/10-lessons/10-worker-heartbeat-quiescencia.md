> Leitura: [← Anterior](09-leases-fencing-checkpoints.md) · [Índice didático](../02-INDEX.md) · [Próximo →](11-baseline-regressao-review.md)

# Heartbeat e parada conservadora

FAC-012L aplica a distinção ao consumer: shutdown aborta a execução antes de fechar a fila; perda de lease usa o mesmo sinal durante checkout/workflow e aguarda `cancelActive()`. Check sem parada confirmada impede checkpoint terminal/complete. O sandbox exige resposta conhecida de systemd: falha de consulta é desconhecida, mesmo quando o comando terminou. Teste Linux aborta comando com descendente e comprova que seu heartbeat deixa de crescer após o retorno.

Heartbeat comprova contato recente; não prova sozinho que um processo anterior morreu. No FAC-004, o worker envia heartbeat a cada cinco segundos e zera o contador quando o controle responde. Após três falhas consecutivas, ele fecha o consumidor BullMQ antes de encerrar com erro.

Essa parada limita o período em que o worker pode continuar ativo sem controle e permite que o supervisor reinicie o processo. Quando jobs reais chegarem, lease e fencing ainda serão necessários: o timestamp ajuda a detectar indisponibilidade, mas apenas a confirmação de término da árvore de processos permite liberar outro writer no mesmo workspace.

## Bootstrap nao e heartbeat

OPS-001 separa tres sinais que antes apareciam juntos no terminal:

- registro inicial: a API pode ainda estar subindo; somente rede/5xx recebe retry limitado a 30 tentativas de um segundo;
- heartbeat: ocorre depois do registro e mantem a parada conservadora apos tres falhas consecutivas;
- resultado de job: um job sintetico antigo pode falhar por revisao-base ausente mesmo com o consumidor pronto.

Retry infinito esconderia indisponibilidade, enquanto retry de 401 apenas atrasaria a descoberta de credencial incorreta. Por isso `registerWithRetry` classifica a falha, mantem o ultimo erro e permite testes sem espera real. O exemplo do repositorio comprova sucesso na terceira tentativa, esgotamento na terceira tentativa do teste reduzido e falha imediata para HTTP 401.

## Lease por tentativa exige renovação e parada comprovada

FAC-012K acrescenta `LeaseGuard` como primitiva independente do heartbeat do processo. A guarda agenda renovações com o fencing token da tentativa e usa a expiração retornada pela API; resposta de erro, expiração ou renovação que não termina a tempo aborta o `AbortSignal` da operação e chama `stopWriter`. O resultado só marca `writerQuiescent` quando esse callback retorna `true`. Mesmo então, a Promise da operação precisa terminar antes de `execute` rejeitar com `LeaseAuthorityLostError`; timeout não é atalho para liberar o writer.

Exemplo de composição futura (não ligado ao consumer): `new LeaseGuard({ fencingToken: claim.fencingToken, leaseDurationMs: 90_000, initialLeaseExpiresAt: claim.leaseExpiresAt, renew: async (token) => control.renew(claim.attemptId, { fencingToken: token, leaseDurationMs: 90_000 }), stopWriter: () => runtime.cancelAndConfirmTree(workflowId) }).execute((signal) => workflow.execute(request, signal))`. O contrato de integração precisa demonstrar que o cancelamento fecha a árvore sandbox/CLI e que `workflow.execute` observa o sinal; testes sintéticos da primitiva não comprovam essas duas capacidades.

FAC-012AD separa heartbeat persistido e prontidão: operation-status.service classifica idade sem alterar o registro, e painel limpa observação antiga durante consulta/falha. Uma tentativa com lease vencida permanece sem stop. global-writer-guard verifica estrutura real do índice; teste PostgreSQL troca-o por índice comum com mesmo nome e exige rejeição. Endpoint administrativo read-only não executa comando nem inventa elegibilidade de IA. [Evidências](../09-entregas/2026/controle/2026-10-03-FAC-012AD-painel-operacao.md).
