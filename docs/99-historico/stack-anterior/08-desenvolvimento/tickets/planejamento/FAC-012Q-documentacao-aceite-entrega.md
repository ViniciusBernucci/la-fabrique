# FAC-012Q — Documentação e aceite exato da entrega

Status: AWAITING_HUMAN. Código `bfd853265360c27f5112063bc7f8ab9ba49b03a9`; 306 testes/checks passaram. Data: 2026-10-03. Baseline: `3ff8bdd`. Branch/worktree: `feat/fac-012q-delivery-gate`, `/home/vinicius/le-fabrique-fac-012q`.

## Objetivo e escopo

Gerar relatório de entrega Markdown determinístico baseado em READY imutável, checks/revisão/chamadas e bundle persistidos; apresentar/download pelo painel. Aceite humano explícito vincula resultado+artefato+documento exatos e só então promove run/ticket a DONE. Relatório documenta a entrega; não modifica documentação técnica dentro do repositório externo nem garante semanticamente cada critério. Esse limite será exposto no documento/painel e no controle do MVP.

Caminhos: contracts, API orchestration + campo approval/migration, painel cliente/DeliveryPanel/tests e docs/lessons. Migration apenas versionada; sem banco/fila/serviço real, writer/provider/piloto/login/push/deploy. Provider nenhum; fixtures. Um writer local; baseline 289 testes. Contas/modelos continuam na interface. Duas rodadas de correção no máximo.

## Critérios

1. Documento usa snapshot READY, não campos mutáveis, e vincula base/patch/artefato/resultado/hashes; desconhecidos explícitos, dados não confiáveis escapados, sem prompts/host paths/segredos.
2. Gate exige última tentativa atual COMPLETED/parada, checkpoint COMPLETED, resultado/revisão aprovados, chamadas Developer/Reviewer concluídas, checks aprovados configurados com baseline/pós sem nova regressão, bundle/digests/snapshot coerentes.
3. API administrativa GET entrega gate/documento ou motivo controlado; POST aceite exige expectedVersion/attempt/deliveryDigest estritos. Transação serializável registra documento/hashes/data sem token, atualiza run/ticket apenas VALIDATING; novo conteúdo/versão invalida aceite. Repetição aceita exata não duplica.
4. Painel exige carregar/revisar evidência e confirmação explícita antes do aceite, aborta respostas antigas, sem aprovação automática ou merge/deploy. DONE só após essa ação humana do software.
5. Checks, testes de mismatch/unknown/idempotência/HTML e docs com SHA/diff/lessons; ticket da fábrica permanece AWAITING_HUMAN.

## Rollback

Revert com writer parado preservando approval/documentação/artefatos. Não aplicar/retirar coluna automaticamente. Relatório em `documentacoes/controle/2026-10-03-FAC-012Q-documentacao-aceite-entrega.md`.
