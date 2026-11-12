# ADR-0004 — Sessão validada no servidor e acesso de dev provisório

- **Estado:** ACEITA, **PROVISÓRIA** até o [T04-01](../NEW%20MVP/03-BACKLOG.md#t04-01) · **Ticket:** [T03-03](../NEW%20MVP/03-BACKLOG.md#t03-03) · **Épico:** E03

## Contexto

O T03-03 exige que "acesso sem sessão seja recusado pelo servidor", mas login/TOTP/recuperação são do T04-01, fora do escopo pedido. Sem alguma forma de obter sessão, o painel não poderia ser usado nem testado.

## Decisão

1. **Definitivo:** todas as rotas de dados passam por `requireSession`. O cookie `lf_session` (HttpOnly, SameSite=Strict, `Secure` em produção) carrega um token aleatório de 256 bits; o banco guarda **só o SHA-256**. Uma sessão vale se existir, não estiver expirada nem revogada — revogar tem efeito imediato.
2. **Provisório:** `POST /api/dev/session` emite uma sessão para um admin `dev-admin` **sem senha**. Só é registrado com `DEV_AUTH_ENABLED=true`, e a configuração **recusa essa variável fora de `APP_ENV=development`** (a API nem sobe). O front só mostra o botão "Entrar (somente dev)" em build de desenvolvimento (`import.meta.env.DEV`).

## Riscos e mitigação

- Se o endpoint existir em um ambiente exposto, qualquer pessoa vira administrador. Mitigação: trava na configuração (testada em `config_test.go`), em produção o endpoint responde 404 (verificado no ensaio de produção), e `.env.production.example` fixa `DEV_AUTH_ENABLED=false`.
- Antes de expor o painel na internet, **concluir o T04-01** e remover o endpoint, a variável e o botão.

## Fora do escopo

Senha, TOTP, códigos de recuperação, limitação de tentativas, CSRF, reautenticação e trilha de autenticação (T04-01/T04-02).
