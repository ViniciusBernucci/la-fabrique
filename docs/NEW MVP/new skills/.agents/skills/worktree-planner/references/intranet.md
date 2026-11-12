# Perfil Intranet — planejamento operacional do lote

Leia [execução portátil](../../_shared/execucao-portavel.md), o ticket do tech-lead e o manifesto do planejador-de-lote. Não repita a entrevista já concluída; uma lacuna essencial volta ao planejador. Esta etapa grava planos, sem criar worktrees, implementar ou integrar. O usuário inicia a execução chamando orquestrador-fluxo-ia, salvo pedido anterior para planejar e executar.

## Topologia

| Camada | Repositório | Raiz das worktrees |
| --- | --- | --- |
| Backend | {RAIZ_BACKEND_CONFIRMADA} | {RAIZ_WORKTREES_BACKEND_CONFIRMADA} |
| Frontend | {RAIZ_FRONTEND_CONFIRMADA} | {RAIZ_WORKTREES_FRONTEND_CONFIRMADA} |

Backend termina em /app. docker-intranet é ambiente Docker, sem Git. Confirme topologia com Git de leitura. Worktree tem a estrutura do repositório da camada, sem o prefixo do checkout principal.

Por feature: no máximo uma worktree por camada; tickets internos da mesma feature ficam no mesmo prompt. Features independentes do lote usam worktrees próprias. Uma separação diferente pedida pelo usuário prevalece. Branch final {feat|fix|refactor|chore}/{chave}-{assunto}, igual nos dois repositórios quando cruzar camadas. Chave é ID externo ou chave local estável; camada aparece no nome da pasta, não na branch final.

O orquestrador cria as worktrees automaticamente, já na branch final. Quando a criação for gerenciada pelo ORCA, use orca-cli e confirme o nome real gerado; se for necessário renomear, faça apenas renomeio simples da branch da tarefa, nunca -M. Não exija criação manual no ORCA nem nome temporário de branch em planos Git puros.

Leia ../../_shared/perfis-de-risco.md e contexto-e-automacao.md. Simples usa ticket curto, estado e manifesto; 00/99 só quando houver decisão operacional específica. Os nomes abaixo continuam aceitos em planos existentes.

## Arquivos

Na pasta central {FONTE_CLAUDE}/worktrees/AAAA-MM-DD/, usando data/fuso e identificador do manifesto:

- 00-plano-{chave}-{assunto}.md: contexto, resultado, escopo/fora do escopo, critérios Dado/Quando/Então, regras de negócio, contratos de API campo a campo, dependências e ordem de deploy/merge. Reutilize o backlog, sem duplicar especificação divergente.
- 01-{chave}-backend-{assunto}.md e/ou 02-{chave}-frontend-{assunto}.md: prompt mínimo por camada, referenciando especificação e regras.
- 99-merge-e-limpeza-{chave}-{assunto}.md: bases, branch de revisão do lote, ordem e verificações de integração e limpeza, seguindo as referências do orquestrador. Nunca merge em main/master, push ou exclusão de branches.
- estado-{chave}-{assunto}.md: contrato de estado-e-gates do orquestrador, fase PRONTA_PARA_ORQUESTRAR ou BLOQUEADA_NO_PLANEJAMENTO com motivo real.

Liste a pasta antes de escrever; retome somente arquivos da mesma tarefa/lote. Em colisão, use slug distinto e atualize os links. Não sobrescreva planos anteriores. Vários lotes no mesmo dia têm manifestos distintos lote-AAAA-MM-DD-{slug}.md.

## Bases e dependências

Registre ref e SHA completos de cada repositório. O ponto inicial fica fixo no manifesto; não inclua pull origin main nos prompts, pois mudaria o escopo revisado. Atualização posterior da base é decisão operacional registrada e invalida evidências afetadas.

Dependência entre tarefas do mesmo repositório: a dependente nasce do commit aprovado da tarefa de origem, depois de seus gates; registre quais commits herda. Sem dependência de símbolo/código, cada tarefa nasce da base do lote. Identifique colisões de arquivos e contratos antes de sugerir paralelismo. Dependências são um grafo sem ciclos; uma tarefa isolada bloqueia suas dependentes.

## Prompt para o executor

A seção se chama **## 3. Prompt para o executor (Claude, Antigravity ou Codex)**. Gere texto pronto, com caminhos, branches, IDs e contratos resolvidos, sem placeholders pendentes. O cabeçalho informa repositório real, worktree planejada, base ref/SHA, branch, camada, ticket e manifesto.

Inclua nesta ordem:

1. **Âncora e regras:** confirme cwd, git rev-parse --show-toplevel, git rev-parse --git-common-dir, branch ativa e status. A âncora é parte do prompt/estado, não acréscimos repetidos em CLAUDE.local.md rastreado. Nunca identificar worktree só pelo path ou pela presença de âncora. Conflito de branch/diretório vira evento de bloqueio, não escrita no checkout errado.
2. **Leituras:** AGENTS.md/CLAUDE.md da camada, regras centrais, especificação e plano geral por caminho absoluto, ADRs e contratos pertinentes. Permita leitura desses recursos fora da worktree; escrita de código/docs fica nesta camada. Não proíba ler skills externas necessárias. Se recurso local não existir, carregue o equivalente central confirmado, sem inventar arquivo.
3. **Especificação e plano:** caminho da especificação canônica e IDs dos critérios; acrescente somente decisões operacionais da camada, perfil, dependências, sequência e arquivos prováveis. Não peça uma nova aprovação do plano durante execução já autorizada.
4. **Preparação da camada:** frontend precisa proxy.conf.json válido, obtido do checkout principal quando existir e sem copiar segredos. Use dependências já declaradas/lockfile. Se faltarem, restaure pelo gerenciador do projeto sem alterar manifests/lockfile, respeitando permissões do host; não adicione pacote nem use npm install para resolver versões por tentativa. Backend: php -l e verificações permitidas; qualquer comando que toque banco depende das provas de isolamento das regras centrais. Não instale dependências novas no lote.
5. **Implementação:** use implementador-de-ticket. Não execute code review, auditoria ou aprovação de QA no papel de implementador. A validação local não substitui gates independentes.
6. **REGISTRO DE ENTREGA:** carregue fechar-entrega e seu perfil Intranet pelo caminho absoluto confirmado, mesmo quando a skill não estiver descoberta no cliente. Referencie o perfil por caminho absoluto. Não transcreva o contrato se o arquivo estiver acessível; use o resumo abaixo apenas como fallback de ambiente sem acesso, registrando o motivo.
7. **COMMIT DESTA ENTREGA:** na branch da tarefa, confira índice completo, adicione arquivos próprios explicitamente, registro junto do código, mensagem de uma linha em português, sem co-autoria. Sem git add . / -A, sem amend após hash indexado, sem push/PR nem merge pelo implementador. Sem autorização de host, devolva bloqueio real.
8. **HANDOFF AO ORQUESTRADOR:** devolva evento com hashes e evidências. Encerrar a sessão executora não interrompe a coordenação do lote.

### Contrato mínimo de fechamento embutido

Registro docs/historico/entregas/AAAA-MM-DD-{chave}-{assunto}.md no repositório desta worktree. Atualize o registro da mesma tarefa/sessão, não o de outra sessão parecida. Campos: data/fuso, ID externo ou “não possui ID vinculado”, camada, branch, Contexto, O que foi feito e por quê, Arquivos alterados, Referências de diff/validação, Impacto e pontos de atenção e Resumo para o ClickUp.

Registre arquivos próprios e comandos de diff com base/SHA conhecidos; não copie diffs completos nem novos arquivos inteiros para o registro. Antes do commit, use a base e o inventário; o evento posterior registra o commit, evitando autorreferência impossível. Backend e frontend têm registros separados com links cruzados. Migração não executada e teste não executado são limitações explícitas.

Resumo termina com um único bloco, marcadores em linhas próprias:

<!-- clickup-resumo:inicio id={id-externo-ou-nenhum} camada={backend-ou-frontend} -->
Título de produto, o que mudou, como testar, pontos de atenção e nome do repositório. Sem diff ou detalhes internos.
<!-- clickup-resumo:fim -->

No prompt final, resolva os valores dos marcadores. Resumo fica dentro do registro, além do chat; nenhuma publicação externa é implícita.

Depois do commit, o executor devolve os campos para o orquestrador atualizar UMA linha por registro no índice central .claude/entregas/AAAA-MM-DD.md. Só o coordenador escreve no índice durante o lote, serializando e verificando duplicação por tarefa/camada/registro. Commit vem antes da linha; sem commit, escreva “sem commit” e bloqueie o gate. Links partem da pasta do índice: ../../app/ para backend e ../../../intranetNova/ para frontend; antes de integrar, o conteúdo pode ser lido por git show {hash}:{registro}.

### Evento de conclusão

FLUXO_EVENTO: IMPLEMENTACAO_CONCLUIDA
TAREFA: chave real
CAMADA: backend ou frontend
WORKTREE: caminho absoluto
BRANCH: branch final
BASE: SHA
COMMIT: SHA completo
REGISTRO: caminho
VALIDACOES: comandos e resultados observados
LIMITACOES: lista objetiva

Em falha: FLUXO_EVENTO: BLOQUEIO com TAREFA, CAMADA, EVIDENCIA e MOTIVO. Não declare implementação concluída sem commit. Correções futuras são delegadas pelo review-loop-driver somente para os IDs APLICAR do auditor, ou por pedido expresso do usuário fora do lote.

## Conferência antes do handoff

Confira que todos os planos têm caminhos reais, branch e base, regras de negócio/aceite verificáveis, contratos entre camadas, validação viável, registro/commit/evento e fora do escopo. Nenhum prompt pede confirmação humana durante execução, depende exclusivamente de Claude CLI, manda push/PR, remove branch ou atualiza a base silenciosamente. O plano referencia o manifesto correto e o estado começa na fase correta. Devolva ao planejador-de-lote, que consolida o lote; se chamado diretamente, termine nos planos sem iniciar execução implicitamente.
