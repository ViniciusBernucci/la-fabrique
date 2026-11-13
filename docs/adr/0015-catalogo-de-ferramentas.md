# ADR-0015 — Catálogo de ferramentas e ordem de verificação da chamada

- **Estado:** ACEITA · **Ticket:** [T08-03](../NEW%20MVP/03-BACKLOG.md#t08-03) · **Épico:** E08 · **Decisões:** D10–13, D74, D78–80

## Decisão
1. **Ferramenta = contrato**: nome, tipo (`native`, `script`, `integration`), parâmetros e saída com **esquema tipado** (`string`, `int`, `bool`, `path`, `enum`; `list` só na saída), **permissões** (`read`, `write`, `exec`, `git`, `api`) e **limites** (tempo e bytes de saída). Nativas no código: `read_file`, `write_file`, `edit_file`, `list_dir`, `search`, `run_command`, `git_status`, `git_diff`. Personalizadas no banco (migration 11).
2. **Ordem fixa antes de qualquer efeito** (`Registry.Check`/`Call`): a ferramenta existe → foi **concedida** ao agente → as **permissões** exigidas estão no escopo concedido → os argumentos **batem com o esquema** (desconhecido, ausente, tipo errado, fora da faixa, fora do formato ou caminho que escapa do workspace falham) → a **política de risco** autoriza (`Guard`, ADR-0012) → executa com limites. Falhas das quatro primeiras etapas **nem chegam à política**.
3. **Ferramenta nova não herda acesso:** o `Grant` lista nomes explicitamente; registrar ou descobrir uma ferramenta não a concede a ninguém.
4. **Sem shell para argumentos:** as nativas rodam por `argv` (sem interpretação do shell); `write_file`/`edit_file` entregam o conteúdo por stdin. Ferramentas de script recebem os parâmetros em `PARAM_<nome>` (variáveis de ambiente), **nunca** substituídos no texto do comando, o que impede injeção. Caminhos são confinados lexicalmente a `/workspace` (qualquer `..` explícito é recusado).
5. **Script e integração sob política:** o comando de uma ferramenta de script é avaliado no cadastro (proibido ou fora da lista → recusa; sensível → aceito) e de novo em cada chamada; a operação de uma ferramenta de integração é avaliada como ação de conector. As nativas de arquivo têm regras `fs.*`.
6. **Saída tipada:** o resultado é conferido contra o esquema (campo ausente ou de tipo errado é erro; campos fora do esquema são descartados). Ferramentas de script devolvem um objeto JSON no stdout; sem esquema, `exit_code` e `stdout`. Saída acima do limite é truncada e sinalizada; estouro do tempo falha a chamada.

## Limites
- Links simbólicos dentro do sandbox e acesso ao socket são do T11-02; aqui a proteção é lexical.
- `git_status`/`git_diff` exigem `git` na imagem do perfil, que hoje não existe: a falha é clara, mas a execução com git **NOT_RUN**.
- O cadastro de ferramentas personalizadas é por API (a tela lista, sem formulário); ferramentas MCP são do T08-04.
- O tempo estourado não encerra os processos dentro do container (T11-03).
