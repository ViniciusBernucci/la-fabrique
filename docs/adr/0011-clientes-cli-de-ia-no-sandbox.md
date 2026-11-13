# ADR-0011 — Clientes CLI de IA no sandbox (contas e execução)

- **Estado:** ACEITA · **Ticket:** [T09-01](../NEW%20MVP/03-BACKLOG.md#t09-01) · **Épico:** E09 · **Decisões:** D07, D26, D34–35, D80–87

## Decisão
1. **Conta = conexão do cofre do tipo `cli`** (nome, `target` = `claude` ou `codex`). O ID da conexão (UUID) dá nome ao volume `lf-acct-<id>`, que guarda a configuração e o **login oficial** do cliente. Duas contas do mesmo provedor têm volumes (e logins) distintos e são escolhidas por ID.
2. **O login é o oficial de cada CLI**, feito pelo responsável no volume da conta. A fábrica nunca lê, grava nem repassa tokens; o ambiente do processo só leva caminhos (`CLAUDE_CONFIG_DIR` ou `CODEX_HOME` apontando para `/accounts/current`, e `HOME=/tmp`). O segredo que o cofre exige para conexões `cli` é um marcador sem uso: a credencial real vive só no volume.
3. **O sandbox monta apenas o volume da conta daquela execução** (`execution.Mount`: volumes nomeados, nunca caminhos do host; destinos reservados — `/workspace`, `/tmp`, `/etc`, `/proc`… — são recusados). Sandboxes sem o volume não veem a conta.
4. **Modo autorizado por construção** (`BuildCommand`): `read_only` e `workspace_write`. Claude Code: `--permission-mode dontAsk` com lista de ferramentas (somente leitura: `Read,Grep,Glob`) ou `acceptEdits` com `Edit,Write` e extras explícitos (`Bash(make test)`); Codex: `--sandbox read-only|workspace-write`. Os modos que desligam a proteção do cliente (`bypassPermissions`, `danger-full-access`) **não existem** na API. Modelo, ferramentas e diretório são validados por expressão regular. O **prompt vai por stdin**, não por argumento (que apareceria na lista de processos).
5. **Eventos normalizados** (`start`, `message`, `tool`, `usage`, `error`, `done`, `raw`) para os dois formatos de saída JSON; linha desconhecida vira `raw`, linha acima de 1 MiB falha a execução sem travar o processo.
6. **Cancelamento**: cancelar o contexto encerra a **árvore de processos** dentro do container (SIGTERM e, passado o prazo, SIGKILL), e a execução volta marcada como cancelada.

## Limites
- **Os CLIs reais não foram executados** (exigiria a conta do responsável). Flags conferidas no `--help` (2026-11-12); formatos de evento seguem a documentação, sem confirmação por execução. Testes usam um CLI de mentira em Docker real.
- Falta o **fluxo de login** da conta (hoje manual no volume) e a imagem de sandbox com os CLIs instalados. O isolamento sandbox interno do Codex dentro de um container pode não funcionar (aninhamento de namespaces): questão em aberto.
- O volume da conta fica acessível ao código do projeto se o mesmo sandbox rodar os dois; separar o container do cliente do container de scripts do projeto é pendência (E11).
- Capacidade do provedor, cota e orçamento são do T09-03/T09-04.
