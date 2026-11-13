# ADR-003 - Stack TypeScript da fábrica
Data: 2026-09-29. Status: ACEITO pelo usuário. Revisão 2.3.
## Decisão e motivo
React + TypeScript + Vite para o painel, NestJS + TypeScript para API/orquestração e worker Node.js + TypeScript separado para execução. PostgreSQL guarda o estado; Redis + BullMQ transportam jobs. Reduzir linguagens mantidas e compartilhar contratos motiva a escolha. Substitui a proposta Angular/Laravel da fábrica; preserva a stack dos projetos desenvolvidos por ela. Tudo continua na mesma VPS.
## Organização proposta do monorepo
| Caminho | Responsabilidade |
|---|---|
| apps/web | Painel React, build estático servido pelo proxy HTTPS |
| apps/api | NestJS: autenticação, projetos, tickets, runs, providers, contexto, auditoria e orquestração |
| apps/worker | Node.js: adapters, claim/heartbeat, processos, snapshots e checks; sem porta pública |
| packages/contracts | Esquemas e tipos de DTOs, eventos e checkpoints |
| packages/runtime | Interfaces/componentes de runtime sem dependência do frontend |
Backend modular inicialmente; separar microserviços quando houver necessidade medida. Fixar versões e lockfile no bootstrap após verificar compatibilidade. ORM, gerenciador de pacotes e biblioteca de UI serão registrados nessa etapa.
## Contratos e comunicação
Tipos TypeScript não validam entrada em runtime. Compartilhar esquemas, por exemplo Zod, e validar HTTP, fila e saída dos clientes. Não exportar entidades de persistência, segredos ou código servidor ao bundle do painel. Configuração validada no startup.
O painel usa HTTP para ações e SSE autenticado para eventos sanitizados, com sequência persistida/retomada e consulta HTTP de fallback. Aprovação segue vinculada à revisão exata.
## Persistência e fila
Transações PostgreSQL e outbox registram mudanças/dispatch. Dispatcher publica jobs BullMQ com IDs estáveis. Consumidores tratam duplicação/reentrega; não prometer exactly-once. Manter leases, fencing e confirmação de término independentemente do lock BullMQ. Nenhum evento do agente autoriza gasto, merge ou deploy.
## Execução e segurança
API não executa clientes ou builds dentro da requisição. Worker usa spawn/execFile assíncronos com argv explícito e shell desativado; prompts por stdin quando suportado. Implementar backpressure, limites de logs, timeout e término da árvore/grupo de processos. Novo writer exige confirmação de quiescência do anterior.
Separação de processos não basta para isolamento: preservar usuários/redes/volumes distintos, sandboxes, credenciais inacessíveis ao código e ausência de Docker socket no piloto. Clientes oficiais autenticados na VPS continuam sujeitos ao preflight.
## Verificação e operação
FAC-012Y distingue handoff interno de cliente e transferência de writer: o mesmo worker mantém lease/fence e exclusão global durante stop/snapshot/restore em outra worktree, sem liberar claim. Novo job/retomada exige confirmação de stop e fence novo. Alternativas são configuração explícita, limitadas e revalidadas; não há sessão privada compartilhada/reset de budgets/fallback pago. Snapshot intermediário local não promete recuperação após crash anterior à publicação; unknown continua BLOCKED_RECOVERY.

Checks: typecheck, lint, builds de web/API/worker e testes relevantes. Integração cobre contratos inválidos, jobs duplicados, crash/reinício, timeout, handoff, gate documental e aprovação obsoleta; UI conforme critérios do ticket.
CPU intensa e builds ficam fora do event loop da API. Medir RSS, heap, cgroup, OOM, disco e latência sob carga; limitar retenção/logs. Toolchains PHP/.NET/outras entram somente quando o piloto exigir.
Mantidos mínimo 4 vCPU/8 GB/120 GB, bom 8 vCPU/16 GB/200 GB e ideal 8 vCPU/32 GB/300 GB, um executor inicial e sem GPU. Revisar capacidade apenas com medições.
## Estado da entrega
Atualização documental de planejamento; nenhum serviço implementado ou implantado. Mantidos assinaturas oficiais, API/extras desligados, handoff, documentação por domínio, diffs, lessons e aceite humano.

FAC-012Z: política documental versionada por projeto segue contratos runtime/UI/READY. Worker executa gate bounded após stop/checks e exige revisão semântica/snapshot inalterado; API apenas cruza hashes/conjunto com evidência persistida, sem executar código. Compatibilidade de leitura dos registros antigos preservada, novos jobs exigem política.

FAC-012AA implementa perfil Claude por caminho e preflight oficial opt-in sob UID do serviço; factory exige prova privada current/fingerprint, preservada fora de credenciais/workspaces. Não alterou topologia ou ativou provider/serviço. Prova real continua requisito operacional; fixtures não substituem isolamento/financeiro/identidade verificados.

FAC-012AC materializa o limite global já aprovado de um writer: consulta de admissão e índice PostgreSQL único parcial em stopped_confirmed=false. Concorrência de consumidores e lease por run não substituem essa exclusão. Stop comprovado libera, expiração/status não; conflitos antigos impedem migration sem auto-repair. [Evidências](../09-entregas/2026/controle/2026-10-03-FAC-012AC-writer-global.md).

FAC-012AD expõe observações operacionais administrativas sem projeto, em snapshot read-only RepeatableRead. Guard estrutural do índice global é compartilhado entre observação e claim; ausência bloqueia nova execução. Heartbeat é sinal recente, não prova de autenticação ou stop. [Evidências](../09-entregas/2026/controle/2026-10-03-FAC-012AD-painel-operacao.md).

FAC-012AE implementa o SSE previsto: eventos de invalidação sanitizados por projeto, sequência bigint decimal persistida em linha transacional bloqueada até commit; não substituir por nextval/ordem de alocação. Triggers registram mudanças atomically, Bearer no cabeçalho e Last-Event-ID na retomada, HTTP permanece fallback. [Evidências](../09-entregas/2026/controle/2026-10-03-FAC-012AE-eventos-projeto.md).

FAC-012AF implementa tooling offline de backup, não execução na API: dump congelado + evidências privadas, AES-256-GCM/chave externa e staging novo. PostgreSQL continua autoridade; prova sintética real restaura events/cursor/índice e Git snapshot. Não altera topologia nem storage externo. [Evidências](../09-entregas/2026/operacao/2026-10-03-FAC-012AF-backup-restauracao.md).

FAC-012AG materializa kill switch conservador de agendamento: estado PostgreSQL versionado/default paused, share lock na admissão e PAUSE em renew; BullMQ entrega pausada/reconciliada sem apagar intent. Pedido não comprova stop; deferred job ainda sem writer não consome execução. [Evidências](../09-entregas/2026/controle/2026-10-03-FAC-012AG-pausa-global.md).

FAC-012AH: PostgreSQL mantém exclusão global; 429 pré-claim por capacidade/rollback conhecido produz delayed BullMQ sem consumo de tentativa. Não converter falha de lease/execução/commit ambíguo em retry. Nenhum cliente/check executado pela API. [Evidências](../09-entregas/2026/controle/2026-10-03-FAC-012AH-admissao-capacidade.md).
