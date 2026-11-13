# ADR-0005 — Autenticação, sessões e cofre de conexões

- **Estado:** ACEITA · **Tickets:** [T04-01](../NEW%20MVP/03-BACKLOG.md#t04-01), [T04-02](../NEW%20MVP/03-BACKLOG.md#t04-02), [T04-03](../NEW%20MVP/03-BACKLOG.md#t04-03) · **Épico:** E04 · **Decisões:** D01, D36, D37 · **Supera em parte:** [ADR-0004](0004-sessao-no-servidor-e-acesso-dev-provisorio.md)

## Contexto

O painel precisa de um único administrador, 2FA obrigatório, sessões revogáveis e um cofre para credenciais de GitHub, Jira, ClickUp, CLI, API e MCP que nunca vaze para logs, respostas ou prompts.

## Decisão

**Identidade (T04-01)**
1. Conta única criada por `make create-admin` (`cmd/migrate create-admin`; usuário e senha vêm de variáveis de ambiente, nunca de argumentos). Não existe cadastro pela web. Senha: 12–128 caracteres, hash **argon2id** (m=64 MiB, t=3, p=2, formato PHC; parâmetros ficam no hash). Usuário inexistente e senha errada gastam o mesmo custo e dão a mesma resposta.
2. **Login em duas etapas.** `POST /auth/login` valida a senha e abre um *desafio* (cookie `lf_challenge`, HttpOnly, 10 min, só o SHA-256 no banco, 5 tentativas). **Só o segundo fator abre sessão** (`/auth/totp` ou `/auth/recover`). No primeiro acesso o 2FA é cadastrado antes de qualquer sessão (`/auth/totp/setup` + `/auth/totp/enable`).
3. **TOTP** RFC 6238 (SHA-1, 6 dígitos, 30 s, ±1 passo). O passo aceito é gravado (`totp_last_step`) com `UPDATE … WHERE totp_last_step < $passo`: um código vale **uma vez** (anti-replay, atômico). O segredo é guardado **cifrado** (AES-256-GCM, AAD = `admin-totp:<id>`).
4. **Códigos de recuperação:** 10 por conjunto, 80 bits cada, só o SHA-256 no banco, mostrados uma única vez. O consumo é um `UPDATE … WHERE used_at IS NULL` dentro da mesma transação que consome o desafio: requisições concorrentes produzem **exatamente uma** sessão. Entrar por recuperação marca `totp_reset_required`; a substituição do 2FA gera novos códigos (os antigos deixam de valer) e revoga as outras sessões. A recuperação não decifra nada, então **funciona mesmo sem `VAULT_KEY`**.
5. **Limitação de tentativas** (`auth_throttle`): 5 falhas por conta ou 20 por IP em 15 min bloqueiam por 15 min (429 + `Retry-After`). Falhas de senha, TOTP e recuperação contam juntas.

**Sessões (T04-02)**
6. Cookie `lf_session` (256 bits, HttpOnly, SameSite=Strict, `Secure` em produção); só o SHA-256 no banco; validade absoluta de 12 h; revogação vale na próxima requisição. Lista de sessões, revogação individual, "encerrar as outras", logout.
7. **CSRF em duas camadas:** token por sessão (`X-CSRF-Token`, comparação em tempo constante) em todo método que altera estado **e** checagem de `Origin`/`Sec-Fetch-Site` em todo POST/PUT/PATCH/DELETE, inclusive login. Sem `Origin` nem `Sec-Fetch-Site` (cliente não navegador) vale o token.
8. **Reautenticação** (senha + código TOTP, replay-safe) abre uma janela de 5 min (`reauth_at`). Sessão recém-criada já conta. Ações sensíveis: criar/rotacionar/revogar conexão, substituir 2FA, encerrar outra sessão. Sem janela: 403 `reauth_required`.
9. **HTTPS:** com `REQUIRE_HTTPS` (padrão em produção) a API recusa requisição sem `X-Forwarded-Proto: https` (exceto sondas) e envia HSTS. O IP vem de `X-Forwarded-For` só com `TRUST_PROXY`; o nginx do painel só aceita XFF de redes privadas (`set_real_ip_from`).
10. **Trilha de autenticação** (`auth_events`): tipo, IP, user agent, correlação e detalhe mínimo. Nunca senha, código, token nem o nome tentado.

**Cofre (T04-03)**
11. Segredo cifrado com **AES-256-GCM**; chave `VAULT_KEY` (32 bytes) **fora do banco** (variável de ambiente, `.env` ignorado). O AAD é `connection:<id>`: copiar o cifrado para outra linha não abre. `key_id` fica registrado para uma futura rotação da chave mestra (hoje há uma chave só).
12. Conexão = `id` estável + `kind` + `target` + `scopes` + segredo cifrado. **Rotação** troca o segredo mantendo o `id` (vínculos futuros usam `connection:<id>`), com controle otimista de versão. **Revogação** apaga o ciphertext e é idempotente. `Vault.Resolve` consulta o banco a cada uso (sem cache): revogação e rotação valem no próximo uso. A API **nunca** devolve o segredo.
13. **Redação em camadas:** `security.Secret` (todo formato/JSON/slog vira `[REDACTED]`); `Redactor` (valores exatos registrados — inclusive versões antigas após rotação — mais padrões de token) aplicado a (a) logs via `logging.SetRedactor` (mensagem, atributos de texto e erros), (b) toda resposta JSON da API, (c) detalhe de teste de conexão. Na subida, `Prime` carrega os segredos já gravados (erro de banco impede a API de subir; conexões que não decifram — `VAULT_KEY` errada — geram aviso, sem bloquear o login por recuperação).
14. **Teste de conexão:** GitHub, ClickUp e Jira têm teste remoto (sem seguir redirecionamentos; Jira só em `https://*.atlassian.net`, contra SSRF; detalhe nunca contém corpo remoto). `cli`/`api`/`mcp` devolvem `not_run` — sem teste remoto seguro nesta fase; a validação real ocorre no primeiro uso pelos adaptadores (E06/E09/E14).

## Consequências e riscos

- **Perder `VAULT_KEY` perde as conexões e o segredo TOTP.** O login continua possível por código de recuperação (item 4), mas as conexões precisam ser recriadas. Guarde uma cópia da chave **fora** do servidor e do backup do banco ([backup-restauracao.md](../operacao/backup-restauracao.md)).
- O bloqueio por conta permite que um atacante trave o login legítimo por 15 min repetindo falhas. Aceito para um painel pessoal; o IP real (item 9) evita que um só IP bloqueie todos.
- Não há QR code no painel: o 2FA é cadastrado por chave ou URI `otpauth://` (evita dependência extra).
- Rotação da chave mestra (re-cifrar com `VAULT_KEY` nova) **não** está implementada.
- Prova de "nenhum segredo em prompts" depende do pipeline de prompts (E09/E10): quem montar prompts **deve** passar o texto por `Redactor.Redact` e usar `connection:<id>` em vez do valor.

## Fora do escopo

Multiusuário, SSO, WebAuthn/passkeys, rotação automática de credenciais externas, mecanismos oficiais de login das CLIs (E09).
