# FAC-012T — Retomada segura do trabalho preservado

Estado: IMPLEMENTADO / AWAITING_HUMAN. Data: 2026-10-03. Baseline `6c91be9` (READY sobre `93735b0`). Código `42e7037f81c2fe19cefee85bb64c67373b29795f`. Um writer na worktree `/home/vinicius/le-fabrique-fac-012t`; nenhum provider real, piloto, serviço, banco, fila, migration aplicada, push ou deploy.

## Funcionamento e contratos

POST `/api/runs/:id/resume`, AdminAuthGuard, payload estrito expectedVersion/attemptId/artifactDigest. Transação serializável exige estado PAUSED/PAUSED_LIMIT/FAILED/CANCELLED coerente no run/ticket, última tentativa terminal/fence/parada, checkpoint, resultado e bundle íntegros. Reconfere hashes/manifesto/base/HEAD/patch/stop. Unknown, aceite humano/VALIDATING/DONE e evidência ausente/corrompida recusados.

Outbox `run.resume.v1` congela objetivo/definição READY original e resumeFrom (origem/fence/digests/snapshot), com deduplicação por origem. Run/ticket aguardam worker; pedido não cria processo/fence. Mesmo pedido pendente idempotente. Claim verifica job contra intenção persistida e reconfere evidência parada; só então cria tentativa/fence novo. Reentrega reutiliza tentativa, histórico não é apagado. Eventos/fences antigos não autorizam nova execução.

Consumer lê snapshot privado por UUID/ArtifactReader e confere digest autorizado. Workflow mede baseline na base limpa ANTES de restore; restaura patch/binários/untracked, reconstrói contexto e orienta continuar sem reaplicar patch. Checks posteriores validam estado preservado, não reclassificam regressão como preexistente. Snapshot restaurado permanece referenciado se rota/guard impedir novas chamadas; no máximo três manifestos no relatório, sem apagar artefatos.

SnapshotManager.restore recusa symlink na origem/bytes e ancestrais do alvo, caminhos de controle, base divergente/target dirty; git apply --check antes de aplicar, untracked usa criação exclusiva. Falha não inicia agente; alteração parcial continua conservadoramente fenced. Developer com resultado FAILED conhecido também captura progresso antes de retorno/correção. Não recupera sessão privada; configuração de contas/modelos continua atual e pela interface.

Painel oferece retomada só em tentativa parada com snapshot, exige confirmação por versão e informa que nova tentativa humana reinicia seus limites. Busca bundle/digest sob demanda, requisição autenticada; efeito depende de IDs primitivos, nenhuma retomada automática. Mudança de versão/cleanup aborta resposta; pedido incerto exige atualizar antes de repetir. SHA-256 do browser requer contexto seguro (HTTPS/localhost); falha permanece sem solicitação.

Gate de entrega aceita especificação congelada do evento de retomada; aprovação ainda depende de checks/review/bundle/documento atuais. Sem merge/deploy automático.

## Verificação e diff sanitizado

Checks da revisão final passaram: typecheck completo; 355 testes (launcher 2, contracts 32, runtime 46, API 124, worker 137, web 14); builds completos; lint 187 arquivos; git diff --check. Git/filesystem reais em fixtures comprovam restauração exata e symlinks bloqueados. API/Prisma/filas/clientes de IA simulados; UI verificada por renderização estática, sem ensaio interativo em navegador ou operação real.

Testes incluem intenção/claim/replay/fence, oito recusas sem mutação, job/origem adulterados, baseline pré-restore/contexto pós-restore, regressão preservada não preexistente, restore falho sem agente, Developer falho com snapshot, gate de entrega retomado e confirmação UI. Primeira suíte passou typecheck/testes/build mas lint encontrou chain format que exigiu segunda passagem do formatter; corrigido e suíte completa repetida. Nenhuma dependência nova, zero vulnerabilidades no npm ci.

Diff sanitizado revisável: `git diff 6c91be9 42e7037 -- apps packages`; dados sintéticos, nenhuma credencial ou piloto hardcoded.

## Limites, rollback e aceite

Retomada exige bundle existente até 64 KiB e stop comprovado, não desbloqueia hard crash/unknown. Falha pré-snapshot requer recovery específico, não reinício cego. Cada retomada humana tem orçamento próprio explícito; não é handoff automático nem mudança silenciosa de conta. API não executa builds/clientes. Gate real permanece false; infra/manual/preflight/backup e aceites não foram realizados.

Rollback: com writer parado, reverter código e preservar outbox/evidências; eventos novos não são apagados nem publicados em implementação anterior incompatível. Nenhuma migration nova. Aceite da revisão exata pendente; integração local autorizada não torna DONE.
