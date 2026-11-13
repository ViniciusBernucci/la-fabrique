> Leitura: [Índice didático](../../02-INDEX.md) · [Próximo assunto →](../infraestrutura/00-dimensionamento-vps.md)

> Instruções de criação em documentacoes/ ou lessons/ abaixo são histórico da fonte; novas entregas/lessons seguem exclusivamente docs/00-governanca/POLITICA-DOCUMENTACAO.md.

> Nota editorial DOC-MV-001, 2026-10-07: conteúdo integral local preservado como referência da evolução v2.3. Afirmações por incremento têm a revisão/data original; trechos sobre consumer ausente, recuperação pendente e caminhos antigos não descrevem necessariamente HEAD a2cc5e0. O [AS-IS conferido](../../02-arquitetura/01-as-is.md) e a [auditoria](../../00-governanca/08-AUDITORIA.md) delimitam o estado atual; números de testes abaixo são evidência histórica, não reexecução desta migração.

# Checkpoint e troca de provider

FAC-012Y integra handoff automático de cliente dentro da tentativa por alternativas explícitas na UI: stop → snapshot → worktree/restore/contexto → rota atual. Máximo dois handoffs sem reset de budgets, Reviewer read-only; mesmo worker mantém lease/fence e exclusão global. Ausência de alternativa aguarda X; unknown bloqueia. [Relatório](../../09-entregas/2026/handoff/2026-10-03-FAC-012Y-handoff-configurado.md).

FAC-012T continua a retomada HUMANA: última tentativa parada com hashes/bundle/checkpoint, nova intenção/outbox/fence e worktree, baseline pré-restore e contexto pós-restore. Configuração atual resolve contas/modelos; sem sessão privada. Crash unknown permanece bloqueado. [Relatório](../../09-entregas/2026/controle/2026-10-03-FAC-012T-retomada-snapshot.md).

## Decisão obrigatória da stack - revisão 2.3
A stack da própria La fabrique está APROVADA: React + TypeScript + Vite no painel; NestJS + TypeScript na API; worker Node.js + TypeScript em processo separado; PostgreSQL; Redis + BullMQ; Docker Compose na mesma VPS. Não solicitar nova escolha ou confirmação da stack. Não iniciar a fábrica em PHP/Laravel, Angular ou .NET. Esta decisão substitui propostas anteriores.
Monorepo: `apps/web`, `apps/api`, `apps/worker`, `packages/contracts`. Contratos compartilhados precisam de validação em runtime. API não executa clientes, builds ou testes; o worker executa esses trabalhos com isolamento, limites e um writer inicial.
Ao trabalhar na própria fábrica, aplicar esta stack. A regra de preservar a stack existente aplica-se somente a projetos EXTERNOS cadastrados para desenvolvimento pela fábrica; ela não altera a stack da La fabrique. Se o repositório da fábrica contiver implementação anterior incompatível, registrar a divergência e planejar a adaptação por etapas; não apagar código existente nem reabrir a escolha tecnológica.
Versões exatas e comandos devem ser fixados conforme compatibilidade no bootstrap; isso não é uma nova decisão de stack. Repositório e funcionalidade do piloto externo permanecem pendentes quando não fornecidos.

## Regra central
Um writer ativo por workspace. Estado externo é a memória compartilhada: PostgreSQL para workflow, Git para código/docs, armazenamento de artefatos para patch/untracked/checks e checkpoint para próximos passos. Não transferir sessão privada de um fornecedor a outro.
## Procedimento
1. Solicitar pausa e bloquear novas ferramentas/etapas. Se cliente não pode pausar, cancelar árvore e aguardar término comprovado.
2. Supervisor coleta status Git, base/code SHA, diff binário quando aplicável e lista de untracked; exclui segredos e dependências. Fazer snapshot verificável mesmo que o agente não consiga gerar resumo após esgotar cota.
3. Persistir artefatos com hashes e checkpoint. Commit WIP só se permitido no repo e conteúdo revisado; alternativamente guardar patch + untracked. Commit sozinho não cobre arquivos não rastreados.
4. Quando transferir autoridade para novo job/writer, confirmar envio/recuperabilidade, liberar lock anterior e emitir novo fencing_token. Novo job rejeita eventos do token antigo. No handoff interno Y não se libera lock nem se cria writer: o mesmo worker conserva lease/fence, troca apenas cliente após stop e restaura artefato local verificado. Crash antes de publicar continua sem recuperação automática desse snapshot intermediário.
5. Novo provider lê checkpoint, confere revisão e patch, reexecuta checks pertinentes e continua somente próximos passos. Não reaplicar patch já presente.
6. Registrar handoff, motivo, origem/destino e evidência de snapshot/tempo/resultado. Y limita a dois handoffs e também aos budgets por função/globais; sem destino elegível retorna WAITING_PROVIDER, budgets interrompem com PAUSED_LIMIT. Histórico de preparação não prova chamada/sucesso do destino; consultar observações efetivas.
## Corrida de recuperação
Lease expirado na VPS não prova que processo local morreu. Durante partição de rede, o worker deve se auto-interromper antes de vencer sua janela de lease. Controle não envia outro writer ao mesmo workspace até confirmação de quiescência; sem confirmação, manter BLOCKED_RECOVERY. Não considerar fencing_token suficiente para impedir dois processos editando arquivos locais.
Heartbeat proposto 15s, lease 90s; worker precisa parar antes de expirar, incluindo margem de encerramento. Valores devem ser ensaiados. Supervisor sem lease não inicia comandos. Cancelamento registra resultado desconhecido caso não possa confirmar término.
## Conteúdo mínimo
Ticket/run/attempt, objetivo/aceite, escopo, base SHA/code SHA, branch/workspace, provider/versão/modelo conhecido, arquivos modificados/untracked, patch hash, decisões e evidências, testes reais e revisão, falhas/limitações, próximos passos, docs pendentes, motivo da pausa e autenticação/cota observada sem segredos.
Checkpoint não afirma aceite nem transforma checks antigos em atuais. Cancelamento pode ter consumido cota; guardar uso desconhecido como desconhecido.

FAC-007 implementa a camada local de artefato: patch binario, arquivos untracked regulares, modos, tamanhos e hashes, com restauracao somente no mesmo SHA-base limpo. FAC-008 persiste base/code SHA, snapshot/patch hash e parada no checkpoint associado ao attempt; fencing impede mutacao por tentativa antiga. FAC-009 passa a capturar snapshot depois dos checks e antes do Reviewer ou de nova correcao, mas ainda nao envia esse manifesto ao protocolo persistido. Handoff entre providers e persistencia do resultado coordenado permanecem no FAC-010 e na integracao operacional.

FAC-010C implementa o coordenador local de handoff sem ativar o consumidor real. Ele resolve a atribuicao do funcionario, exige destino `AVAILABLE`, parada confirmada e snapshot capturado depois do termino na mesma base. RuntimeGuard limita tentativa/troca; outra worktree recebe o snapshot verificado. A evidencia de origem contem apenas provider, ID de execucao, status, erro normalizado e horario: sessao, mensagem bruta e raciocinio nao entram no contrato.

Claude continua deslogado no ambiente real, portanto o resultado operacional permanece `WAITING_PROVIDER`. Ativacao ainda exige login Claude App por assinatura, gate financeiro, modelo observado e integracao com claim/checkpoint/fencing persistidos; a biblioteca nao autoriza contornar esses gates.
