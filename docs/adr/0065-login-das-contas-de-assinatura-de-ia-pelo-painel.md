# ADR-0065 — Login das contas de assinatura de IA (Claude e Codex) pelo painel

- **Estado:** ACEITA · **Relacionadas:** [ADR-0011](0011-clientes-cli-de-ia-no-sandbox.md), [ADR-0060](0060-conexoes-por-oauth.md), [ADR-0063](0063-guardar-conexao-sem-reautenticacao.md)

## Contexto
O ADR-0011 fez da conta de assinatura uma conexão `cli` (destino `claude` ou `codex`) com o **login oficial do cliente** num volume Docker por conta, mas deixou o login manual. O responsável quer o mesmo jeito do GitHub: um botão, autorizar na própria conta e pronto. Restrições do projeto: usar o cliente oficial, não alterar o fluxo de autenticação dele e nunca usar o token do Claude Code como credencial própria. Por isso a fábrica **não** copia credenciais da máquina e **não** lê o token.

## Decisão
1. **O login oficial roda num container do worker.** O worker ganha acesso ao Docker (`docker.sock`) e uma imagem com os clientes (`infra/images/Dockerfile.cli`, versões fixadas: Claude Code 2.1.325, Codex 0.164.0). Comandos: `claude auth login --claudeai` e `codex login --device-auth`, com `CLAUDE_CONFIG_DIR`/`CODEX_HOME` apontando para o volume `lf-acct-<id da conexão>`.
2. **API e worker conversam pelo banco** (`cli_logins`, migration 00068). A API só registra a intenção e responde ao polling; o worker pega o login, roda o cliente, publica o que a pessoa precisa e confirma o resultado.
3. **Dois jeitos de autorizar, ambos observados nos clientes reais:**
   - **Codex:** device flow. O cliente imprime link (`auth.openai.com/codex/device`) e um código de uso único; a pessoa digita o código no site.
   - **Claude:** o cliente imprime um link e pede para **colar de volta** o código que o site mostra. A tela tem um campo para isso; a API guarda o código por instantes e o worker o entrega ao cliente por stdin, **uma vez**, e o apaga.
4. **Só links dos hosts oficiais chegam à tela** (`claude.com`, `claude.ai`, `platform.claude.com`, `console.anthropic.com`; `auth.openai.com`, `chatgpt.com`). A saída do cliente é tratada como dado.
5. **Concluir = o cliente terminou e o status oficial confirma** (`claude auth status` com `loggedIn: true`; `codex login status` com saída 0). Só então a conexão `cli` é criada, **com o mesmo id do volume** e um marcador no cofre (a credencial real nunca passa pelo cofre).
6. **Nada de curta duração sobra:** link, código e código colado são apagados ao terminar (sucesso, falha, cancelamento ou prazo). Prazo de 15 minutos. Um login aberto por provedor. Falha, cancelamento e prazo derrubam o container e apagam o volume. Ao subir, o worker encerra e limpa os logins que ficaram abertos.
7. **Container sem privilégios:** `--cap-drop ALL`, `no-new-privileges`, limites de processos e memória, usuário não root, volume só da conta.
8. **Rotas** (sessão e CSRF, sem reautenticação, como o ADR-0063): `POST /api/cli-logins`, `GET /api/cli-logins/{id}`, `POST /api/cli-logins/{id}/code`, `POST /api/cli-logins/{id}/cancel`.
9. **Opt-in:** `CLI_LOGIN=true` no worker (ligado no overlay de desenvolvimento), com `VAULT_KEY`. Em desenvolvimento o worker constrói a imagem sozinho se ela faltar; `make cli-image` faz o mesmo à mão.

## Limites
- **O `docker.sock` equivale a acesso root à máquina.** Por isso fica só no overlay de desenvolvimento. A VPS (`compose.prod.yaml`) não o recebe: lá o recurso fica desligado até haver decisão sobre como dar acesso ao Docker (por exemplo, um proxy de socket restrito).
- **Verificado de verdade:** o ciclo completo com Docker real e os clientes reais até o ponto da autorização (Codex mostra link e código; Claude mostra link e pede o código), o cancelamento e a limpeza, e que o status oficial recusa um volume sem login. **NOT_RUN:** a autorização concluída com uma conta real (exige a conta do responsável): a troca do código do Claude e a criação da conexão ao final foram cobertas com um `docker` de mentira que imita a saída real.
- **Antigravity** não entra: o `agy` não tem comando de login conhecido e não é um provedor de execução da fábrica. Fica para uma etapa própria, depois de investigar como ele autentica.
- Reiniciar o worker (inclusive o reload automático do desenvolvimento) no meio de um login o encerra; a pessoa repete.
- O volume da conta fica acessível ao código do projeto se o mesmo sandbox rodar os dois (pendência do ADR-0011, E11).
