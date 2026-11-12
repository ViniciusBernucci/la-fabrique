---
name: planejador-de-lote-v1
description: Planeja um ticket, uma feature ou um lote de tickets da La fabrique encadeando grill-me → tech-lead → worktree-planner e entrega especificação, tickets, prompts por frente e plano de integração em Markdown. Use para planejar antes de implementar; não cria worktrees, não implementa, não commita código.
---

# Planejador de lote (v1)

Coordena o planejamento de uma ou mais demandas até planos prontos para outra sessão executar.
**Planejar não implementa.** Esta skill não cria worktrees, não altera código, não faz merge e não
roda testes. Pode ler o repositório e rodar `git fetch` para registrar a base. Os arquivos do
planejamento são documentação: o commit deles segue a regra de commits do projeto (abaixo).

## Antes de começar

- O projeto é um **monorepo** (Go na API/worker, React/TypeScript em `apps/web`, PostgreSQL,
  Docker). Confirme a raiz com `git rev-parse --show-toplevel`.
- Leia `CLAUDE.md` (ou `AGENTS.md`), `docs/README.md`, `docs/desenvolvimento/01-convencoes.md`, o
  backlog `docs/NEW MVP/03-BACKLOG.md` e o estado em `docs/NEW MVP/07-CHECKLIST.md`.
- Carregar uma etapa significa ler o `SKILL.md` dela e seguir suas instruções **nesta mesma
  sessão**. As etapas são as skills irmãs `../grill-me/SKILL.md`, `../tech-lead/SKILL.md` e
  `../worktree-planner/SKILL.md`. Não abra outros agentes para repetir o planejamento.

## Regra de commits do projeto

Commits sempre na branch **`developer`**, **sem** `Co-Authored-By` nem qualquer atribuição ao
agente, feitos **ao final de cada tarefa**. Branches/worktrees por frente só existem quando o
responsável pede paralelismo real; nesse caso a integração termina em `developer`. Sem push, PR,
merge em `main` ou deploy sem autorização explícita.

## Pasta do lote

Tudo é gravado em `docs/planejamento/AAAA-MM-DD/` (fuso `America/Sao_Paulo`). Nos arquivos
gerados, escreva sempre o **caminho absoluto**, porque worktrees ficam fora do checkout principal.

- Liste a pasta antes de escrever. **Nunca sobrescreva outro lote.** Reutilize arquivos só quando
  for a retomada do mesmo lote; em colisão de nome, acrescente um sufixo e corrija os links.
- Arquivos do lote:
  - `lote-AAAA-MM-DD-{slug}.md`: manifesto (este planejador).
  - `00-plano-{chave}-{assunto}.md`: especificação canônica de cada demanda (tech-lead).
  - `01-{chave}-{camada}-{assunto}.md`: prompt por frente, com camada `api`, `web` ou `infra`
    (worktree-planner).
  - `99-integracao-{slug}.md`: integração e limpeza do lote (worktree-planner).

## Encadeamento

1. **Demandas.** Identifique as demandas que o usuário escolheu. Tickets do backlog usam o ID
   `Txx-yy` como chave; leia o ticket, o épico e as decisões `Dxx` ligadas
   (`docs/NEW MVP/04-RASTREABILIDADE.md`). Demanda sem ticket recebe uma chave local estável em
   kebab-case e é registrada como "fora do backlog". Leia o código e a documentação pertinentes
   antes de perguntar.
2. **grill-me.** Esclareça objetivo, comportamento atual, regras de negócio, bordas e fora do
   escopo. Reaproveite respostas entre demandas do mesmo lote. Falta de requisito não é permissão
   para inventá-lo.
3. **tech-lead.** Assim que a entrevista permitir critérios observáveis, carregue-o com todas as
   respostas. Ele grava o `00-plano`, com tickets, critérios, contratos e perfil de risco. Questão
   essencial sem resposta deixa a demanda `BLOQUEADA_NO_PLANEJAMENTO`; continue as demais.
4. **worktree-planner.** Com os tickets prontos, carregue-o. Ele registra a base e grava os
   prompts por frente e o de integração, sem repetir a entrevista. Para uma única frente
   sequencial no `developer` (o caso comum), ele devolve um único prompt sem worktree.
5. **Manifesto.** Consolide o `lote-AAAA-MM-DD-{slug}.md`.

Não peça ao usuário para invocar as etapas: encadeie sozinho. Pergunte só o que só ele pode decidir.

## Manifesto

- Identificador do lote, data/fuso e branch de integração (`developer`, ou `revisao/AAAA-MM-DD-{slug}`
  quando houver worktrees paralelas).
- Demandas incluídas e excluídas, com motivo.
- Base: branch + SHA completo (`git rev-parse developer`, e `origin/developer` se existir).
- Por demanda: chave/ticket, épico, camadas, perfil de risco e justificativa, link para o
  `00-plano`, IDs dos critérios, prompts, dependências e status.
- Ordem de execução e de integração. Dependências formam um grafo sem ciclos.
- Riscos e pendências.

Referencie em vez de copiar: critérios e contratos já estão no `00-plano`.

## Conclusão

- Marque `PRONTA_PARA_IMPLEMENTAR` só quando critérios essenciais, contratos, prompts e validação
  viável estiverem definidos. Hipótese não é aprovação: registre a origem de cada decisão
  (usuário, evidência no código/documento, hipótese).
- Entregue no chat: caminho do manifesto, resumo do lote, pendências e próximos passos:
  1. se houver worktrees, criá-las conforme o manifesto (branch e base);
  2. entregar a cada sessão executora o prompt correspondente;
  3. depois do commit de cada frente, abrir uma sessão nova com o "Prompt de revisão"
     (`revisor-de-codigo`);
  4. integrar em `developer` quando as frentes estiverem revisadas.
