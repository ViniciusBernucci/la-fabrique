---
name: worktree-planner
description: Planeja o isolamento de tickets já especificados em worktrees da Intranet (backend e frontend), com base fixa, branches, prompts prontos por frente e plano de integração na branch de revisão. Use depois do tech-lead; não cria worktrees, não implementa, não faz merge.
---

# Planejador de worktrees

Transforme tickets já especificados num plano operacional. **Não crie worktrees, não implemente,
não rode Git de escrita.** O usuário cria as worktrees no ORCA; outra sessão executa cada prompt.
Só `git fetch` é permitido, para registrar a base.

Reutilize o `00-plano` do tech-lead. Não refaça a entrevista. Lacuna essencial impede marcar o
prompt como pronto: registre o bloqueio.

## Topologia

| Camada | Repositório Git (checkout principal) |
| --- | --- |
| Backend (Laravel) | `docker-intranet/app` (o `docker-intranet` sozinho não é Git) |
| Frontend (Angular) | `intranetNova` |

Confirme os dois com `git rev-parse --show-toplevel`. A worktree do ORCA tem a estrutura do
repositório da camada, sem o prefixo do checkout principal.

## Base

- Para cada repositório envolvido: `git fetch origin` e `git rev-parse origin/main`. Registre no
  manifesto e nos prompts o ref `origin/main` e o **SHA completo**.
- Tarefa independente nasce desse SHA. Tarefa que depende do código de outra nasce do commit final
  da tarefa de origem, que só existe depois que ela termina. Registre a dependência e diga que o
  prompt só pode começar então.
- Não inclua `git pull` nos prompts: mudaria a base que foi planejada.

## Frentes

- **No máximo uma worktree por camada por feature.** Tickets da mesma feature e da mesma camada
  ficam no mesmo prompt. Features independentes do lote têm worktrees próprias. Separação diferente
  pedida pelo usuário prevalece.
- Separe frentes só com independência real. Antes de sugerir paralelismo, identifique arquivos,
  símbolos e contratos disputados. Nada de paralelismo fictício.
- Branch: `feat|fix|refactor|chore/{chave}-{assunto}`, **igual nos dois repositórios** quando a
  feature cruza as camadas. A camada aparece no nome do arquivo do prompt, não na branch.

## Prompt por frente (`01-…-backend-…md` / `02-…-frontend-…md`)

Texto pronto, sem placeholders pendentes. Cabeçalho: repositório, camada, branch, base (ref + SHA),
tickets, perfil de risco, caminho absoluto do manifesto e do `00-plano`, dependências.

Seção **"Prompt para o executor (Claude, Codex ou Antigravity)"**, nesta ordem:

1. **Conferência da worktree.** Você está numa worktree criada pelo ORCA. Antes de qualquer
   escrita, confira:
   - `git rev-parse --show-toplevel` (a worktree) e `git rev-parse --git-common-dir` (aponta para o
     repositório da camada);
   - `git branch --show-current` igual à branch do plano. Se o ORCA gerou outro nome, renomeie com
     `git branch -m {branch-do-plano}`, só se esse nome ainda não existir (nunca `-M`);
   - `git status --porcelain` vazio;
   - `git merge-base --is-ancestor {SHA-base} HEAD` com sucesso.

   Qualquer divergência: **pare e reporte**. Nunca escreva em outro checkout.
2. **Leituras.** `CLAUDE.md` e `.claude/rules/` desta worktree, o `00-plano` (caminho absoluto) e
   os contratos de API pertinentes. Ler fora da worktree é permitido; escrever, só nela.
3. **Preparação da camada.**
   - Frontend: `proxy.conf.json` copiado do checkout principal quando existir (não é versionado).
     `node_modules` restaurado pelo lockfile (`npm ci`), sem alterar `package.json` nem o lockfile.
   - Backend: validação e testes conforme o `CLAUDE.md` do backend (PHPUnit só no container; banco
     só depois de `banco.md`).
4. **Implementação.** Tickets e critérios `CA-xx` desta frente, por ID, com os arquivos prováveis e
   o que está fora do escopo. Acrescente só decisões operacionais desta camada; não copie a
   especificação.
5. **Registro de entrega e commit.** Conforme `documentacao.md` e `git.md`: registro em
   `docs/historico/entregas/`, commit na branch da tarefa com o registro junto. Sem push, sem PR,
   sem merge.
6. **Relatório final** no chat: SHA do commit, caminho do registro, validações executadas com
   resultado, e o que ficou sem validar e por quê. Para cada critério `CA-xx` da frente: ATENDIDO,
   NÃO ATENDIDO ou NÃO VALIDADO, com a evidência.

Depois, uma seção **"Prompt de revisão (sessão nova, mesma worktree)"**, em poucas linhas: use a
skill `revisor-de-codigo` na branch da frente, com a base `{SHA}` e a spec `{caminho do 00-plano}`.

## Integração (`99-merge-e-limpeza-{slug}.md`)

Ações futuras, executadas só quando o usuário pedir:

1. Para cada repositório, no checkout principal com `git status` limpo: criar
   `revisao/AAAA-MM-DD-{slug}` a partir do SHA base do lote. Nunca `main`/`master`.
2. Merge `--no-ff` das branches das tarefas, na ordem de dependência do manifesto. Conflito:
   **pare** e reporte. Não resolva por conta própria se a resolução mudar comportamento.
3. Validação após a integração: `php -l` nos PHP alterados / `npx --no-install ng build`, mais os
   testes pertinentes conforme as regras.
4. A branch `revisao/…` fica ativa no checkout principal para o usuário testar as telas.
5. Limpeza: remover uma worktree (pelo ORCA ou com `git worktree remove`) só depois de confirmar
   que a branch dela está integrada (`git branch --merged revisao/…`) e que a worktree está limpa.
   Não apague branches.

Sem push e sem PR em nenhuma etapa.

## Conferência antes de devolver

Todo prompt tem caminhos reais, branch, base com SHA, critérios verificáveis, contratos entre as
camadas, validação viável, registro, commit e fora do escopo. Nenhum prompt pede push/PR, apaga
branch, atualiza a base em silêncio ou depende de ferramenta exclusiva de um cliente.

## Saída

- **Chamado pelo planejador:** devolva os caminhos dos prompts e do 99, a ordem e os bloqueios.
- **Chamado direto:** entregue o mesmo no chat, a partir da especificação indicada, e termine no
  plano. Não inicie a execução.
