> Leitura: [← Anterior](05-matriz-cobertura.md) · [Índice didático](../02-INDEX.md) · [Próximo →](07-piloto.md)

> Nota editorial DOC-MV-001, 2026-10-07: conteúdo integral local preservado como referência da evolução v2.3. Afirmações por incremento têm a revisão/data original; trechos sobre consumer ausente, recuperação pendente e caminhos antigos não descrevem necessariamente HEAD a2cc5e0. O [AS-IS conferido](../02-arquitetura/01-as-is.md) e a [auditoria](../00-governanca/08-AUDITORIA.md) delimitam o estado atual; números de testes abaixo são evidência histórica, não reexecução desta migração.

# Controle do MVP — La fabrique

Atualizado em 2026-10-03 pelo FAC-012AH. Estado: MVP DE SOFTWARE IMPLEMENTADO/VERIFICADO / ACEITE E ATIVAÇÃO OPERACIONAL PENDENTES. Software e instalação têm verificações distintas; piloto externo não é requisito de conclusão do MVP. [BACKLOG.md](09-backlog.md) registra aceite por ticket; [documentacoes/INDEX.md](../02-INDEX.md) reúne evidências.

Escopo autorizado: concluir código/verificações internas sem piloto externo, integrar trabalho local na `developer` e preservar configuração pela interface. Aceite humano, push e implantação são distintos de implementação/merge local.

## O que já existe

| Área | Estado comprovado | Evidência |
|---|---|---|
| Stack | React/Vite, NestJS, worker Node/TS, PostgreSQL, Redis/BullMQ | ADR-003; manifests/checks |
| Projeto/objetivo | Cadastro, definição versionada, caminhos, contexto/checks aprovados e tickets READY | FAC-001A/003A/012I |
| Configuração IA | Interface para instalações, modelos, funções e proteções financeiras sem segredos/hardcode | FAC-011/012E–G |
| Snapshot da intenção | Evento READY congela projeto/definição/ticket com invariantes runtime | FAC-012A/B aceitos; OPS-003 |
| Execução | Consumer compõe checkout, workflow, rotas atuais, lease/fencing/checkpoint | FAC-012L `54bf483`; gate desligado |
| Checks/revisão | Baseline, regressões, Reviewer separado e correções limitadas | FAC-009/012G/L |
| Parada | AbortSignal em CLI/checks; systemd desconhecido não confirma parada | FAC-012K/L; testes Linux |
| Artefatos locais | Patch/untracked, manifesto/hash e restauração em bibliotecas | FAC-007; SnapshotManager |
| Resultados no painel | Histórico/checks/revisão/metadados/chamadas | FAC-012M `14ae5ba`; migration não aplicada |
| Integrações oficiais | Verificação/login Codex/GitHub, adapter Claude/handoff e PR sob gate | FAC-010A/B/C e FAC-011A/B/C/D; operação parcial |
| Infraestrutura | Compose controle/bancos e template host worker dedicado | OPS-005; serviço não instalado |
| Git local | D–J consolidados; K/L e OPS-005 integrados pelo OPS-006 após checks | ancestry/relatório OPS-006 |

FAC-012A/B estão DONE com aceites registrados. FAC-012C–Z/AA/AB/AC/AD/AE/AF/AG/AH e OPS-003/004/005/006 aguardam revisão humana; merge local não altera aceite. APIs de IA, extras, recarga e fallback pago permanecem proibidos.

## Estado do software e gates de operação

1. Transporte ampliado IMPLEMENTADO no W: diff completo/bundle até 8 MiB JSON/6 MiB raw com hashes/tamanhos, HTTP compatível e leituras bounded; não entrega arquivo ilimitado/repo inteiro nem trunca. Retenção/backup são operação manual, sem limpeza automática de evidências.
2. Recuperação de finalização IMPLEMENTADA no U: pedido UI/outbox FINALIZATION_ONLY reenvia journal parado/reconcile sem IA/novo writer. N/O/R/S/T preservam/reconciliam/retomam estado conhecido. Hard crash/writer desconhecido permanece bloqueado, sem reexecução por suposição; confirmação física pós-crash pertence operação manual, nunca liberação cega pela interface.
3. Auth/cota/WAITING_PROVIDER IMPLEMENTADOS no X; handoff IMPLEMENTADO no Y: alternativas conta/modelo explícitas na UI, stop/snapshot/restauração/contexto em outra worktree antes de próxima chamada. Reviewer read-only, budgets/lease/fence preservados; zero alternativas por padrão, máximo duas trocas sem gasto extra. Preflight real é manual; erro de segurança não vira fallback, unknown não libera writer.
4. Identidade por instalação IMPLEMENTADA no V: login/status/runtime usam stores privados e factory da rota UI atual, sem adapter global por provider/segredos do controle no ambiente. Autenticação/preflight de serviço manual. Perfil granular Claude e preflight IMPLEMENTADOS no AA; escrita exige prova nativa privada atual de UID/instalação/CLI/hash/política. Prova real não executada; nenhuma conta real desbloqueada.
5. Comandos de interface IMPLEMENTADOS para evidência conhecida: S solicita PAUSE/CANCEL e T retoma snapshot íntegro/parado, com confirmação explícita de orçamento da nova tentativa. Q exige aceite exato antes de DONE. Unknown/crash não oferecem retomada cega; recuperação de finalização segue item 2.
6. Gate documental técnico IMPLEMENTADO no Z: operador configura arquivos/seções, Developer atualiza documentos do projeto, gate estrutural exige mudanças/referências/hashes e Reviewer verifica conteúdo/critério. Snapshot posterior deve ser byte-equivalente; controle cruza evidência com manifesto final. Não substitui aceite humano ou prova operacional.

Evidência da implementação: `apps/api/src/control/control.controller.ts`, `apps/api/src/orchestration/orchestration.controller.ts`, `apps/worker/src/main.ts`, `execution.processor.ts`, `developer-workflow.ts` e `apps/web/src/App.tsx`. O escopo interno auditado está implementado e verificado; estes arquivos não representam pendências de desenvolvimento. Operação real e aceite exato permanecem distintos.

## Passos manuais do responsável

Para ativação real: autenticar clientes sob identidade de serviço, confirmar extras desligados, provisionar/ativar host worker/serviços, aplicar migrations, publicar Git/deploy e validar operação/backup/restauração. Não executados neste incremento. Piloto ausente não bloqueia desenvolvimento, conclusão do software, preparação ou validação da operação com dados sintéticos.

## Conclusão interna e operação posterior

FAC-012AB verificou a composição com Git/sandbox/snapshot/journal reais e portas externas sintéticas. A auditoria seguinte fechou exclusão global AC, observação sem projeto AD, SSE AE, backup/restauração AF, pausa global AG e jobs aguardando capacidade AH. Última revisão de código `f2d80f1671ac8b0c76a5c76f35be4ce626fcab6d`: **528 testes + 10 PostgreSQL + 3 Redis passaram**, typecheck/lint/build/diff verificados; único warning de lint anterior. [Evidências finais](../09-entregas/2026/controle/2026-10-03-FAC-012AH-admissao-capacidade.md).

A fábrica recebe projetos pela interface posteriormente. Desenvolver um projeto externo não é requisito para construir, concluir ou validar internamente o software que fará isso. Nenhuma lacuna interna concreta permanece aberta nesta auditoria; novo problema deve ganhar ticket verificável, sem transformar piloto ou login em impedimento ao desenvolvimento.

Retomada está em `fix/fac-012ah-admission-capacity`, `/home/vinicius/le-fabrique-fac-012ah`, com Z/AA/AB/AC/AD/AE/AF/AG por ancestry. Root `developer` permanece limpo em `745ce2d`; não houve merge/push/deploy. Outros processos no root não permitem comprovar writer parado para integração segura; worktree atual tem writer único e todos os commits disponíveis para revisão. Não inferir aceite humano desta revisão.

Próxima fase operacional: confirmar usuário Linux de serviço/clientes oficiais, autenticação/prova nativa/financeiro sob esse UID; integrar/implantar em janela autorizada e validar reboot/backup externo/operabilidade. Pergunta sobre usuário/clientes já enviada enquanto o código avançava; sem resposta ainda. Nenhuma conta real desbloqueada ou cobrança habilitada. Piloto externo continua opcional posterior.

## Incrementos de confiabilidade atuais

- AC: claim exige ausência global de tentativa sem stop; índice único parcial fecha corrida entre runs. Lease/status não liberam exclusão.
- AD: heartbeat, writer sem stop e índice em painel sem projeto; falta de índice impede claim.
- AE: eventos minimizados persistidos por commit, SSE autenticado/retomável e HTTP fallback.
- AF: backup criptografado e staging exclusivo, roundtrip PostgreSQL e restauração de snapshot Git comprovados internamente.
- AG: pausa global default true/versionada, claim protegido por lock, PAUSE no renew e queue conserva intents; pedido não afirma stop.
- AH: recusa conhecida anterior à autoridade aguarda capacidade sem perder job/consumir tentativa; falhas posteriores/ambíguas propagam.

Migrations desses incrementos só foram aplicadas em PostgreSQL efêmero de teste. Serviço dedicado/ativação e dados existentes permanecem como antes. Rollback preserva índice/pausa/eventos/evidências e exige escritor parado; não remover dados automaticamente.
