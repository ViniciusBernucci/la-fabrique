# Configuração por projeto

As skills mantêm papéis, risco, independência, evidências, limites de correção e autorização. Convenções de infraestrutura/documentação vêm do projeto, não de um caminho fixo no SKILL.md.

## Descoberta

1. Resolva cwd, repositório(s), regras AGENTS.md/CLAUDE.md/GEMINI.md aplicáveis e contexto da tarefa. Instruções vigentes prevalecem sobre configuração; não trate memória como regra.
2. Se houver `.contexto/project.json` na raiz explicitamente selecionada, leia `workflow`. Em worktree use sua configuração ou a raiz de configuração explicitada pelo coordenador, sem procurar automaticamente em outro projeto. Paths configurados são relativos à raiz que contém `.contexto`; roots de repositórios adicionais devem ser confirmadas. Nunca copie segredos entre checkouts.
3. Sem configuração, descubra convenções existentes; pergunte somente lacuna essencial no planejamento. Defaults portáveis: um repositório, `docs/planejamento/ia/` para plano/estado, `docs/historico/` para relatórios, fuso informado pelo usuário (senão UTC explicitado), branch de tarefa por unidade lógica, integração local em branch de revisão. Não crie pastas operacionais da Intranet em outro projeto.
4. Intranet identificada pelas regras centrais explícitas: use o perfil Intranet das skills e sua fonte `.claude` real. Preserve SQL Server, dois repositórios quando pertinentes, índice diário e ClickUp opcional. Não classifique um projeto como Intranet apenas pelo nome de uma pasta.

## Campos workflow

`schema: 1`; `preset: "generic" | "intranet"`; `timezone`; `plansDir`; `reportsDir`; `deliveryDir`; `reviewBranchPrefix`; `defaultProfile`; `rules` (lista de arquivos); `repositories` (lista `{name, root, layer}`); `worktreeDir`; `publish: "manual"`.

Paths não podem substituir proteção de dados nem autorizar comandos. `root` dos repositórios pode ser relativo (`../frontend`), mas confirme a topologia e permissão antes de agir. Diretório externo não vira permissão de escrita. A ferramenta só acessa fontes da raiz selecionada; múltiplos repositórios usam instâncias/configurações separadas e coordenação por referências.

`publish` permanece manual; flags não autorizam push/PR. Campos desconhecidos devem ser esclarecidos ou ignorados com registro; não invente efeitos. A classificação de risco pode elevar `defaultProfile`, nunca ser reduzida por ele. Exemplos e instalação estão no README/INSTALL da ferramenta Contexto; configure o projeto, não edite skills.

## Limites invariantes

Review independente; auditoria/QA conforme perfil; evidência essencial bloqueada não é PASS; até três rodadas técnicas e três de QA; preservação de trabalho alheio; sem publicação externa implícita. Banco, Git, dependências, registros e criação de worktrees seguem regras locais e autorização da chamada. Sem política local, preserve o padrão conservador: sem escrita em main/master, sem push/PR automático, sem operação destrutiva e sem dependência nova silenciosa.
