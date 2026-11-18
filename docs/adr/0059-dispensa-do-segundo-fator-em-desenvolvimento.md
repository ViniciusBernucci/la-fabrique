# ADR-0059 — Dispensa do segundo fator (TOTP) somente em desenvolvimento local

- **Estado:** ACEITA · **Relacionadas:** [ADR-0005](0005-autenticacao-sessoes-e-cofre-de-conexoes.md)

## Contexto
O ADR-0005 exige senha + TOTP em todo login. O responsável pediu, para o uso local, desligar temporariamente o 2FA. Pela regra de autoridade do projeto, a decisão atual do responsável prevalece, e a mudança fica registrada aqui.

## Decisão
1. Nova variável `AUTH_SKIP_TOTP` (padrão `false`). Ligada, a senha correta abre a sessão direto (`status: "authenticated"` + `session` na resposta de `POST /api/auth/login`), sem desafio, cadastro ou código do 2FA.
2. **Só em `APP_ENV=development`.** A configuração da API recusa subir com a flag ligada em `test` ou `production`, então ela não vale em produção nem por engano.
3. Nada mais muda: senha, bloqueio por tentativas, CSRF, expiração de sessão, reautenticação e auditoria continuam. O login registra `login_success` com `method = senha_sem_2fa_dev`. A sessão fica gravada com `auth_method = totp` porque a coluna só aceita `totp` ou `recovery`; o campo de auditoria é que distingue o caso.
4. Desligar de volta: `AUTH_SKIP_TOTP=false` no `.env` e reiniciar a API (`make up`). O 2FA já cadastrado continua válido.

## Limites
- Acesso sem senha continua proibido; esta decisão dispensa só o segundo fator.
- Quem ligar a flag num ambiente exposto reduz a proteção à senha. Por isso ela só existe em desenvolvimento.
