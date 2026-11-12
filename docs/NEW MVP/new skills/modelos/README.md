# Modelos mínimos de instruções

`CLAUDE.md` e `AGENTS.md` contêm o mesmo bloco essencial para o fluxo destas skills. São modelos para incorporar às regras de outro projeto, não configurações completas de aplicação.

## Como usar

1. Instale as skills conforme o README da distribuição, preservando `_shared/`, referências e scripts.
2. Para Claude, use o conteúdo de `CLAUDE.md`. Para um agente que lê `AGENTS.md`, use o conteúdo desse arquivo. Se o projeto usa ambos, mantenha o bloco consistente nos dois ou estabeleça uma fonte comum explicitamente referenciada.
3. Se o arquivo de destino já existe, acrescente o bloco e resolva conflitos com suas regras. **Não substitua o arquivo inteiro nem apague instruções existentes.** Os modelos ficam em `modelos/` para não se tornarem regras ativas na raiz desta distribuição.
4. Acrescente os fatos específicos do seu projeto descritos abaixo. A configuração muda no projeto; o código das skills permanece reutilizável.

## O que configurar em cada projeto

| Informação | O que declarar nas regras locais ou na configuração existente |
| --- | --- |
| Repositórios | Raízes e camadas; não presuma que backend e frontend estão na mesma raiz Git |
| Validação | Comandos reais de lint, build e testes; critérios e isolamento exigido para banco/serviços |
| Documentos e worktrees | Diretórios de planos, relatórios, entregas e worktrees; fuso operacional |
| Políticas locais | Regras de dependências, Git, dados e limites adicionais; permissões continuam sendo as do cliente |
| Contexto, quando usado | Instalação real da CLI, runtime compatível e configuração da raiz selecionada |

Com Contexto, o objeto `workflow` de `.contexto/project.json` descreve caminhos, fuso e repositórios; não concede autorização nem substitui regras de segurança. Sem Contexto, as skills usam as convenções locais/defaults documentados nelas. Não é necessário instalar o Contexto para descobrir ou ler as skills.

Se a ferramenta estiver separada das skills, configure `CONTEXTO_CLI` no ambiente do executor para o arquivo real `contexto.mjs`, ou declare o caminho ao coordenador. Copiar os modelos não instala Node, integrações MCP, ferramentas externas ou sessões independentes. Gates que exigem esses recursos permanecem bloqueados se não estiverem disponíveis.

Os modelos não incluem caminhos de máquina, nomes de empresa, framework obrigatório, banco específico, credenciais ou vínculo obrigatório com gerenciador de tarefas. Também não repetem os contratos completos de cada skill: eles continuam nos respectivos `SKILL.md` e referências.
