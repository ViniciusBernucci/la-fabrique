# Entrega FAC-004 — identidade e heartbeat do worker

Data: 2026-09-30

Status: DONE

Revisão de código: `6d73041bdcbd50215b6a018c9937475c29f542b1`

## Resultado

O worker recebe UUID e credencial próprios pelo ambiente, registra nome/capacidades/OS/arquitetura/Node na API e envia heartbeat a cada cinco segundos. A API persiste a identidade e expõe a listagem apenas ao administrador. Rotas de serviço usam prefixo interno bloqueado no Nginx público. Após três falhas consecutivas, o worker fecha o consumidor BullMQ e encerra com erro para recuperação pelo supervisor.

## Evidências

- `npm run lint`: 52 arquivos aprovados.
- `npm run typecheck`: todos os workspaces aprovados.
- `npm test`: 16 testes aprovados em 8 arquivos.
- `npm run build`: aprovado; bundle web 310,73 kB, 94,47 kB gzip.
- Migration `20260930000200_fac_004_worker_identity`: aplicada no banco de desenvolvimento e pela API na composição completa.
- Registro integrado: worker `00000000-0000-4000-8000-000000000001`, estado `ONLINE`, capacidades `probe` e `subscription-client-preflight`.
- Heartbeat avançou entre duas leituras administrativas; token inválido retornou `401`.
- Proxy público retornou `404` para `/api/internal/workers/register`.
- Composição publicou somente `8080`; worker, API, PostgreSQL e Redis ficaram sem portas públicas.
- Ensaio de falha: API parada; worker registrou falhas 1, 2 e 3, fechou com `control-unavailable` e voltou após restauração.

## Limites

O estado `ONLINE` representa o último heartbeat persistido; consumidores devem considerar também sua idade. Há uma única credencial de worker no MVP e sua rotação ainda é manual. Não existem claim, lease, fencing ou execução de clientes. O encerramento conservador prepara esses mecanismos, mas não os substitui.

## Rollback

Reverter o commit restaura o probe sem registro. A tabela aditiva `worker_identities` pode permanecer sem afetar o código anterior. Para rollback operacional, restaurar as imagens anteriores e manter os volumes; não usar `down -v`.

## Documentação e lessons

Foram atualizados operação, planejamento, README, índice, changelog e backlog. `lessons/worker-heartbeat-quiescencia.md` registra a parada conservadora aplicada.

## Aceite

O responsável aceitou explicitamente o código `6d73041bdcbd50215b6a018c9937475c29f542b1` e a documentação `c2c211c` em 30/09/2026.
