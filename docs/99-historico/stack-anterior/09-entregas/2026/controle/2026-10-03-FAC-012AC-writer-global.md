# FAC-012AC — Exclusão global do writer e independência do piloto

Data: 2026-10-03. IMPLEMENTADO / AWAITING_HUMAN. Código exato `143f3d6bae888942050571f5dc04c5ab8d773b0b`; baseline `a54876f`, READY `76c6d61`. Branch `fix/fac-012ac-global-writer`, worktree exclusiva. Nenhum outro writer nessa worktree.

## Problema e funcionamento

Claim só consultava tentativa da mesma run. Dois tickets diferentes podiam obter autoridade simultânea, apesar de concurrency=1 em cada processo worker. Antes de criar tentativa, API agora consulta globalmente tentativas com stoppedConfirmed=false, sem filtrar lease, status ou worker. Replay e recuperação anteriores permanecem. Índice PostgreSQL `attempts_single_unconfirmed_writer`, único parcial na expressão constante 1, fecha a disputa entre transações concorrentes. P2002 no insert vira HTTP conflict sem conceder claim. Prova persistida de parada libera o índice; status terminal sozinho não libera.

Migration aditiva não inventa prova nem corrige dados: múltiplas tentativas sem parada tornam sua aplicação impossível até diagnóstico autorizado. Comentário no schema lembra que Prisma não representa este índice parcial e futuras migrations devem preservá-lo. Nenhum banco existente alterado.

A instrução reiterada do responsável também foi aplicada ao estado atual de PILOTO, PLANO-MVP, CONTROLE-MVP e operação: piloto externo é experimento posterior opcional e nunca requisito para concluir o software ou preparar/verificar operação sintética. Históricos de entregas anteriores foram preservados. Autenticação real/financeiro/implantação têm requisitos próprios, sem vínculo com escolher projeto externo.

## Verificação e diff sanitizado

- `npm ci --ignore-scripts`, geração Prisma e build de baseline: passaram; sem migration no ambiente existente.
- `npm test`: 445 passaram (scripts 5, contracts 34, runtime 50, API 148, worker 190, web 18).
- `npm run test:postgres`: 3 testes adicionais passaram, zero skipped, PostgreSQL 18.1 em container exclusivo/efêmero com network none, tmpfs, limite 512 MiB/1 CPU. Todas as migrations reais foram aplicadas nesse banco. Sem porta, bind mount, volumes antigos ou DATABASE_URL externo.
- Corrida entre tickets/workers em transações reais: exatamente 1 insert confirmado e 1 rejeitado pelo índice; lease expirada/FAILED continuam excluindo; stopped=true permite novo writer e tentativa de reabrir o anterior falha; aplicação sobre 2 writers recusa sem mudar evidências.
- `npm run typecheck`, `npm run build` e `git diff --check`: passaram. `npm run lint`: 211 arquivos, único warning anterior optional chaining em run-delivery.service.ts.
- API ganhou 7 testes; resume/replay anteriores passaram. Uma rodada de formatação, sem falha/correção funcional nova. Logs efêmeros em `/tmp/fac-012ac-{tests,postgres,typecheck,lint,build}.log`; prova reproduzível versionada.

Diff: `git show 143f3d6bae888942050571f5dc04c5ab8d773b0b -- apps/api/src/orchestration apps/api/prisma/migrations/20261003060000_fac_012ac_global_writer package.json scripts/global-writer.postgres.test.mjs` (6 arquivos, 275 inserções/11 remoções). Somente fixtures sintéticas, sem tokens/contas. Container privado removido ao término, serviços existentes preservados.

## Limites, rollback e aceite

Índice é requisito da revisão da API: aplicar em janela autorizada, com writer parado e diagnóstico de qualquer conflito; SQL failure não autoriza marcar stopped=true. A consulta sozinha não substitui índice na concorrência. Não prova autenticação/identidade/isolamento nativo, implantação, reboot ou disponibilidade externa. Sem cliente/modelo oficial, consumo, API de IA/extras/login/push/merge/deploy. Piloto não bloqueia a continuação.

Rollback: preservar índice e evidências enquanto execução estiver possível; reverter software somente com writer parado. Remover proteção do banco exige ticket autorizado e zero tentativas sem stop; não há remoção automática. Aceite humano da revisão exata pendente, não DONE.

## Docs e lessons

Atualizados controle/infraestrutura/operação/planejamento, ADR-003 (persistência do limite já aprovado), README, INDEX, changelog/backlog, PILOTO/PLANO/CONTROLE e lesson de leases/fencing. Conceito aplicado: exclusão global persistida é independente do lock da fila e da lease de cada run.
