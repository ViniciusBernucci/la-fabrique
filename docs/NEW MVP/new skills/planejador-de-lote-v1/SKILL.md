---
name: planejador-de-lote-v1
description: Planeja uma feature ou um lote de features da Intranet encadeando grill-me → tech-lead → worktree-planner e entrega especificação, tickets, prompts por worktree e plano de integração em Markdown. Use para planejar antes de implementar; não cria worktrees, não implementa, não commita.
---

# Planejador de lote (v1)

Coordena o planejamento de uma ou mais demandas até planos prontos para outra sessão executar.
**Planejar não implementa.** Esta skill não cria worktrees, não altera código, não commita, não faz
merge e não roda testes. Pode ler o repositório e rodar `git fetch` para registrar a base.

## Antes de começar

- Leia o `CLAUDE.md` e o `.claude/rules/` dos repositórios que a demanda toca:
  backend `docker-intranet/app`, frontend `intranetNova`. Confirme cada raiz com
  `git rev-parse --show-toplevel`.
- Carregar uma etapa significa ler o `SKILL.md` dela e seguir suas instruções **nesta mesma
  sessão**. As etapas são as skills irmãs `../grill-me/SKILL.md`, `../tech-lead/SKILL.md` e
  `../worktree-planner/SKILL.md`. Não abra outros agentes para repetir o planejamento.

## Pasta do lote

Tudo é gravado em `docker-intranet/.claude/worktrees/AAAA-MM-DD/` (fuso `America/Sao_Paulo`). Do
backend o caminho é `../.claude/worktrees/`; do frontend, `../docker-intranet/.claude/worktrees/`.
Nos arquivos gerados, escreva sempre o **caminho absoluto**, porque as worktrees do ORCA ficam em
outro lugar.

- Liste a pasta antes de escrever. **Nunca sobrescreva outro lote.** Reutilize arquivos só quando
  for a retomada do mesmo lote; em colisão de nome, acrescente um sufixo e corrija os links.
- Arquivos do lote:
  - `lote-AAAA-MM-DD-{slug}.md`: manifesto (este planejador).
  - `00-plano-{chave}-{assunto}.md`: especificação canônica de cada demanda (tech-lead).
  - `01-{chave}-backend-{assunto}.md` / `02-{chave}-frontend-{assunto}.md`: prompt por worktree
    (worktree-planner).
  - `99-merge-e-limpeza-{slug}.md`: integração e limpeza do lote (worktree-planner).

## Encadeamento

1. **Demandas.** Identifique as demandas que o usuário escolheu. Com ID do ClickUp e conector
   disponível, leia a tarefa **somente para leitura**. Sem ID, gere uma chave local estável em
   kebab-case e registre "não possui ID vinculado". Leia o código e a documentação pertinentes antes
   de perguntar.
2. **grill-me.** Esclareça objetivo, comportamento atual, regras de negócio, bordas e fora do
   escopo. Reaproveite respostas entre demandas do mesmo lote. Falta de requisito não é permissão
   para inventá-lo.
3. **tech-lead.** Assim que a entrevista permitir critérios observáveis, carregue-o com todas as
   respostas. Ele grava o `00-plano`, com tickets, critérios, contratos e perfil de risco. Questão
   essencial sem resposta deixa a demanda `BLOQUEADA_NO_PLANEJAMENTO`; continue as demais.
4. **worktree-planner.** Com os tickets prontos, carregue-o. Ele registra as bases e grava os
   prompts 01/02 e o 99, sem repetir a entrevista.
5. **Manifesto.** Consolide o `lote-AAAA-MM-DD-{slug}.md`.

Não peça ao usuário para invocar as etapas: encadeie sozinho. Pergunte só o que só ele pode decidir.

## Manifesto

- Identificador do lote, data/fuso, branch de integração `revisao/AAAA-MM-DD-{slug}`.
- Demandas incluídas e excluídas, com motivo.
- Base de cada repositório: `origin/main` + SHA completo.
- Por demanda: chave, ID externo (ou "não possui"), camadas, perfil de risco e justificativa, link
  para o `00-plano`, IDs dos critérios, worktrees/prompts, dependências e status.
- Ordem de execução e de integração. Dependências formam um grafo sem ciclos.
- Riscos e pendências.

Referencie em vez de copiar: critérios e contratos já estão no `00-plano`.

## Conclusão

- Marque `PRONTA_PARA_IMPLEMENTAR` só quando critérios essenciais, contratos, prompts e validação
  viável estiverem definidos. Hipótese não é aprovação: registre a origem de cada decisão
  (usuário, evidência no código/documento, hipótese).
- Entregue no chat: caminho do manifesto, resumo do lote, pendências e próximos passos:
  1. criar no ORCA as worktrees listadas no manifesto (repositório, branch e base);
  2. entregar a cada sessão executora o prompt `01`/`02` correspondente;
  3. depois do commit de cada frente, abrir uma sessão nova na mesma worktree com o "Prompt de
     revisão";
  4. pedir a integração do `99` quando as frentes estiverem revisadas.
