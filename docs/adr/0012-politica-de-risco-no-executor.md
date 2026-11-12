# ADR-0012 — Política de risco aplicada no executor

- **Estado:** ACEITA · **Ticket:** [T13-01](../NEW%20MVP/03-BACKLOG.md#t13-01) · **Épico:** E13 · **Decisões:** D07, D26, D34–35, D80–87, D93

## Decisão
1. **Três efeitos:** `allow` (baixo risco, segue sozinho), `approve` (sensível: exige aprovação humana) e `deny` (proibido). A ação é descrita por **ferramenta** (`script`, `cli`, `api`, `mcp`, `connector`, `git`), **operação**, **recurso**, **destino** e **parâmetros reais**.
2. **O efeito mais restritivo vence** entre as regras que casam (`deny > approve > allow`), **independente da ordem**; sem regra, o padrão é **negar** (falha fechada). Nada de agente, comentário externo ou anotação de ferramenta concede permissão: em MCP, `readOnlyHint` é dado, não autorização; ferramentas MCP só são liberadas por nome na configuração, e as que alteram estado (nome começando por `delete`, `write`, `send`, `exec`…) exigem aprovação mesmo se listadas.
3. **Proibições embutidas** (não dependem de regra): ferramenta desconhecida, operação vazia e **destino interno** (loopback, redes privadas, link-local, metadados de nuvem, `localhost`, `*.internal`, `*.local`, `host.docker.internal`, formas numéricas de IP) para qualquer ferramenta. A decisão olha o texto do destino; resolução de DNS e redirecionamentos precisam ser barrados por quem faz a chamada (camada de rede).
4. **Scripts:** cada comando simples (separado por `;`, `&&`, `||`, `|`, quebra de linha) é avaliado e vale o mais restritivo. Substituição de comando (`$(…)`, crases) e shells aninhados não são analisáveis e viram aprovação. Proibidos: ferramentas de rede, elevação de privilégio, runtime de containers, Git remoto, `rm -rf` da raiz/home. Sensíveis: mudança de dependências, `rm`/`mv`/`find -delete`, `git config`. Baixo risco: `make`/`go`/`npm` de build, teste e lint, leitura e Git local.
5. **Git** pelo broker: push só a branch declarada e que não seja `main`/`master`/`production`/`release`; push forçado e merge direto proibidos; commit, branch e Draft PR permitidos.
6. **Parâmetros canônicos:** JSON de chaves ordenadas e SHA-256 (`params_hash`), a base para vincular autorizações a argumentos (T13-03).
7. **Trilha** (`policy_decisions`): ferramenta, operação, recurso, host de destino, efeito, risco, regra, motivo e hash; os parâmetros entram **redigidos** (chaves sensíveis, padrões de token e os **valores exatos do cofre** via `Redactor`). Se a trilha não puder ser gravada, a ação **não executa**.
8. **Ponto de aplicação:** `Guard.Do` (avalia, registra e só então executa) e `GuardedExecutor`, que protege um `workflows.Executor`: ação sensível devolve `ErrApprovalRequired` e proibida `ErrForbidden`, e o motor segue a rota de erro do nó.

## Limites
- O pedido de aprovação na Central, a retomada e as autorizações com escopo, validade e uso único são do T13-02/T13-03; hoje "sensível" apenas **não executa** e é sinalizado.
- A política padrão é inicial e não foi calibrada com projetos reais; a configuração por projeto/agente (e a allowlist de MCP por agente, T08-04) ainda não existe na interface.
- Segredo arbitrário **dentro de texto livre** só é redigido se for um valor conhecido do cofre ou tiver formato de token.
