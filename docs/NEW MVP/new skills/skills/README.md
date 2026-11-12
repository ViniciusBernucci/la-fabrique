# Fluxo de desenvolvimento com IA (Intranet / ORCA)

Este diretório (`D:\Documentos\00 - Dev\docker-intranet\.claude\skills`) é a **fonte única de manutenção** das skills do fluxo Intranet para **Claude**, **Codex** e **Antigravity**. Os repositórios não carregam cópias: `~/.claude/skills` e `~/.agents/skills` têm links para cada skill daqui, criados e verificados por [`planejador-de-lote/scripts/sincronizar-fluxo.py`](planejador-de-lote/scripts/sincronizar-fluxo.py), que também espelha o `CLAUDE.md` central em `intranetNova\CLAUDE.md`.

Leia também [`_shared/execucao-portavel.md`](_shared/execucao-portavel.md) e [`../CLAUDE.md`](../CLAUDE.md) (espelhado em `AGENTS.md` e `GEMINI.md`).

---

## Visão geral — duas chamadas, zero checkpoints intermediários

O fluxo completo é dividido em **duas entradas**:

1. **Planejamento (`planejador-de-lote`)**: encadeia automaticamente [`grill-me`](grill-me/SKILL.md) → [`tech-lead`](tech-lead/SKILL.md) → [`worktree-planner`](worktree-planner/SKILL.md) para uma ou mais demandas e consolida o manifesto `lote-AAAA-MM-DD-{slug}.md` em `.claude/worktrees/AAAA-MM-DD/`. Termina com os planos prontos, sem iniciar implementação (salvo pedido já explícito do usuário para planejar e executar).
2. **Execução (`orquestrador-fluxo-ia`)**: consome o manifesto e conduz **todo o lote até o fim sem nenhum checkpoint humano intermediário** — cria worktrees, delega implementação, exige commit local e registro de entrega, executa `revisor-de-codigo`, audita os achados com `auditor-de-review`, aplica automaticamente apenas os achados `APLICAR` via `review-loop-driver` + especialista da camada (`implementador-de-ticket`), revalida, executa `revisor-de-qa` (com correções e revalidação técnica + QA), integra nas branches `revisao/AAAA-MM-DD-{slug}`, executa revisão/QA final de interação, remove worktrees órfãs com segurança e gera o pacote `revisao-humana-AAAA-MM-DD-{slug}.md`.

### O que permanece com o humano

- Responder às perguntas essenciais de produto durante o `planejador-de-lote` (`grill-me`);
- Revisar telas no **checkout principal**, onde o orquestrador deixa ativa a branch `revisao/AAAA-MM-DD-{slug}`, e os diffs, usando o pacote `revisao-humana-AAAA-MM-DD-{slug}.md`;
- Executar `git push` e abrir os Pull Requests quando decidir.

Não existem mais paradas intermediárias para criar worktrees manualmente no ORCA, escolher IDs do code review ou destravar etapas no meio do lote: qualquer tarefa com bloqueio real (requisito ausente, conflito semântico, risco destrutivo ou falha que persista após 3 ciclos) é marcada como `ISOLADA` com evidência, e as demais tarefas independentes do lote seguem até o fim.

---

## Sequência completa do fluxo

### Fase A — Planejamento (`planejador-de-lote`)

1. **Seleção e leitura inicial**: identifica as demandas do lote (com ID/link do ClickUp quando existir, ou chave local estável com `id=nenhum` / `não possui ID vinculado`).
2. **Entrevista (`grill-me` / `grill-with-docs`)**: esclarece objetivo, regras de negócio, casos de borda e fora do escopo antes da especificação técnica.
3. **Especificação técnica (`tech-lead`)**: acionado automaticamente assim que a entrevista permite critérios observáveis. Decompõe em tickets executáveis, contratos afetados e grafo de dependências.
4. **Planejamento operacional (`worktree-planner`)**: acionado automaticamente para cada feature pronta. Gera em `.claude/worktrees/AAAA-MM-DD/`:
   - `00-plano-{chave}-{assunto}.md` — visão consolidada, contratos e ordem de integração;
   - `01-{chave}-backend-{assunto}.md` e/ou `02-{chave}-frontend-{assunto}.md` — prompts autocontidos para o executor (Claude, Antigravity ou Codex);
   - `99-merge-e-limpeza-{chave}-{assunto}.md` — plano de integração em `revisao/AAAA-MM-DD-{slug}` e limpeza;
   - `estado-{chave}-{assunto}.md` — estado persistente da tarefa (`PRONTA_PARA_ORQUESTRAR` ou `BLOQUEADA_NO_PLANEJAMENTO`).
5. **Manifesto do lote**: consolida `lote-AAAA-MM-DD-{slug}.md` com todas as tarefas, grafo sem ciclos, SHAs-base por repositório e branches de revisão.

### Fase B — Execução autônoma (`orquestrador-fluxo-ia`)

Para cada tarefa `PRONTA_PARA_ORQUESTRAR` (independentes em paralelo quando houver recursos; dependentes após aprovação da origem):

1. **Criação das worktrees (`CRIACAO_WORKTREES`)**: cria as worktrees pelo Git CLI (ou `orca-cli` quando gerenciado), já nas branches finais `{feat|fix|refactor|chore}/{chave}-{assunto}` a partir do SHA-base fixado.
2. **Implementação e fechamento (`IMPLEMENTACAO`)**: delega a sessão executora na worktree com `implementador-de-ticket` e `fechar-entrega`. Grava `docs/historico/entregas/AAAA-MM-DD-{chave}-{assunto}.md`, faz commit local sem push e devolve `FLUXO_EVENTO: IMPLEMENTACAO_CONCLUIDA`. O coordenador atualiza `.claude/entregas/AAAA-MM-DD.md` de forma serializada e idempotente.
3. **Gate técnico único (`CODE_REVIEW`)**: em sessão independente, executa [`revisor-de-codigo`](revisor-de-codigo/SKILL.md) uma vez sobre o diff da tarefa, consolidando correção/performance (`code-reviewer`), qualidade (`clean-code-reviewer`) e segurança (`security-reviewer`).
4. **Auditoria do review (`AUDITORIA_REVIEW`)**: em sessão independente, [`auditor-de-review`](auditor-de-review/SKILL.md) confronta cada achado com código, escopo e regras de negócio, classificando como `APLICAR`, `ADIAR` ou `REJEITAR` (ou `AUDITORIA_BLOQUEADA` se faltar definição essencial).
5. **Correções automáticas (`CORRECOES_REVIEW`)**: [`review-loop-driver`](review-loop-driver/SKILL.md) registra a fila rastreável e delega **somente** os achados `APLICAR` ao especialista da camada (`implementador-de-ticket`), que corrige, roda validações, atualiza o registro de entrega e commita (`fix: ...`, sem push).
6. **Revalidação técnica (`REVALIDACAO_REVIEW`)**: `revisor-de-codigo` revalida em sessão independente os IDs aplicados e os novos commits. `AINDA PRESENTE` ou `REGRESSÃO` voltam ao auditor (limite de 3 ciclos por tarefa; persistindo bloqueio, a tarefa vai para `ISOLADA`).
7. **QA e correções (`QA` → `CORRECOES_QA` → `REVALIDACAO_QA`)**: [`revisor-de-qa`](revisor-de-qa/SKILL.md) valida os critérios de aceite em sessão independente. Falhas dentro do escopo são corrigidas pelo `implementador-de-ticket`, passam por revalidação técnica focal e revalidação de QA (limite de 3 ciclos; persistindo, `ISOLADA`).
8. **Commit documental (`PRONTA_PARA_INTEGRAR`)**: commita explicitamente na branch da tarefa os relatórios de review, auditoria, fila e QA (`docs: ...`, sem push).
9. **Integração e validação integrada (`INTEGRACAO` → `VALIDACAO_INTEGRADA`)**: cria worktree de integração na branch `revisao/AAAA-MM-DD-{slug}` (com sufixos `-2`, `-3` se houver grupos incompatíveis), faz `git merge --no-ff` por tarefa na ordem de dependências, executa validações pós-merge, `revisor-de-codigo` + `auditor-de-review` de interação entre tarefas, `revisor-de-qa` dos fluxos combinados e `review-guide`, e commita os relatórios finais.
10. **Branch de revisão no checkout principal**: com tudo aprovado e commitado, remove a worktree de integração e faz `git switch revisao/AAAA-MM-DD-{slug}` no checkout principal de cada repositório — é lá que o usuário testa as telas. Só troca com o checkout principal limpo (nunca stash/reset); senão, deixa os comandos no pacote. Avisa se `composer.lock`/`package-lock.json` mudaram e exigem reinstalar dependências.
11. **Limpeza segura e pacote final (`PRONTO_PARA_REVISAO_HUMANA`)**: remove worktrees de integração e worktrees órfãs comprovadamente limpas, sem sessão ativa e com commits preservados em branches nomeadas (nunca apaga branches nem usa `--force`). Grava `.claude/worktrees/AAAA-MM-DD/revisao-humana-AAAA-MM-DD-{slug}.md`.

---

## Estados da tarefa

Contrato detalhado em [`orquestrador-fluxo-ia/references/estado-e-gates.md`](orquestrador-fluxo-ia/references/estado-e-gates.md):

```text
BLOQUEADA_NO_PLANEJAMENTO | PRONTA_PARA_ORQUESTRAR
→ CRIACAO_WORKTREES
→ IMPLEMENTACAO
→ CODE_REVIEW
→ AUDITORIA_REVIEW
→ CORRECOES_REVIEW
→ REVALIDACAO_REVIEW
→ QA
→ CORRECOES_QA
→ REVALIDACAO_QA
→ PRONTA_PARA_INTEGRAR
→ INTEGRACAO
→ VALIDACAO_INTEGRADA
→ PRONTO_PARA_REVISAO_HUMANA
(ou ISOLADA em qualquer ponto com bloqueio real documentado)
```

---

## Catálogo de skills

| Skill | Papel principal | Não faz |
| --- | --- | --- |
| [`planejador-de-lote`](planejador-de-lote/SKILL.md) | Encadear `grill-me → tech-lead → worktree-planner` e gerar `lote-AAAA-MM-DD-{slug}.md` | Implementar código ou disparar execução sem pedido |
| [`grill-me`](grill-me/SKILL.md) | Entrevistar requisitos e decisões de produto antes da especificação | Inventar requisitos ou implementar |
| [`grill-with-docs`](grill-with-docs/SKILL.md) | Entrevistar demanda registrando decisões e glossário | Substituir `tech-lead` |
| [`tech-lead`](tech-lead/SKILL.md) | Transformar requisitos em tickets executáveis e dependências | Implementar ou aprovar entregas |
| [`worktree-planner`](worktree-planner/SKILL.md) | Gerar planos `00/01/02/99` e `estado-*` por feature | Criar worktrees ou rodar review |
| [`orquestrador-fluxo-ia`](orquestrador-fluxo-ia/SKILL.md) | Executar o lote sem checkpoints até integração, limpeza e pacote final | Fazer `git push`, abrir PR ou autoaprovar sem evidência |
| [`implementador-de-ticket`](implementador-de-ticket/SKILL.md) | Implementar escopo autorizado e correções delegadas (`APLICAR` / QA) | Auditar ou aprovar o próprio código |
| [`fechar-entrega`](fechar-entrega/SKILL.md) | Gravar registro de entrega com `clickup-resumo`, commitar sem push e emitir `FLUXO_EVENTO` | Fazer `git push`, abrir PR ou publicar no ClickUp |
| [`revisor-de-codigo`](revisor-de-codigo/SKILL.md) | Gate técnico único (correção/performance, qualidade e segurança) e revalidação focal | Editar código |
| [`auditor-de-review`](auditor-de-review/SKILL.md) | Auditar achados do review contra código, escopo e negócio (`APLICAR`, `ADIAR`, `REJEITAR`) | Editar código ou adiar bloqueante real confirmado |
| [`review-loop-driver`](review-loop-driver/SKILL.md) | Consolidar fila e delegar correções (`APLICAR` no lote; indicadas pelo usuário fora dele) | Marcar `VERIFICADO` sem revisor independente |
| [`revisor-de-qa`](revisor-de-qa/SKILL.md) | Validar critérios de aceite e regressões em sessão independente | Aprovar por leitura sem execução |
| [`arquiteto-de-software`](arquiteto-de-software/SKILL.md) | Definir/revisar arquitetura estrutural e ADRs | Planejar sprints ou implementar rotineiramente |
| [`security-audit`](security-audit/SKILL.md) | Auditoria de segurança ampla / modelo de ameaças | Aplicar correções automaticamente |
| [`simplify`](simplify/SKILL.md) | Refatorar legibilidade sob pedido sem alterar comportamento | Substituir `revisor-de-codigo` |

---

## Portabilidade e sincronização

Para (re)criar os links globais das skills e do diretório de agentes e espelhar o `CLAUDE.md` central em `intranetNova\CLAUDE.md`. Um diretório real no lugar de um link (instalação antiga) é movido para `D:\Documentos\00 - Dev\_backup-fluxo-ia\{data}\` antes de virar link; nada é apagado:

```bash
python "D:/Documentos/00 - Dev/docker-intranet/.claude/skills/planejador-de-lote/scripts/sincronizar-fluxo.py"
```

Para apenas verificar se há divergências (modo check):

```bash
python "D:/Documentos/00 - Dev/docker-intranet/.claude/skills/planejador-de-lote/scripts/sincronizar-fluxo.py" --check
```
