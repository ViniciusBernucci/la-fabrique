# ADR-0060 — Conexão do GitHub pela conta da máquina; Jira e ClickUp por token de API

- **Estado:** ACEITA · **Relacionadas:** [ADR-0005](0005-autenticacao-sessoes-e-cofre-de-conexoes.md) (cofre), [ADR-0059](0059-dispensa-do-segundo-fator-em-desenvolvimento.md)

## Contexto
O responsável não quer digitar senhas ou tokens no painel: a conexão com GitHub (e demais) deve usar o acesso do próprio usuário, autorizado no site do serviço, como faz o Orca. A decisão atual do responsável prevalece sobre o item 14 do ADR-0005 (credencial colada para esses três tipos).

## Revisão (2026-11-15) — decisão final do responsável
Registrar OAuth Apps (e client secrets) é atrito sem ganho. Vale esta forma, igual ao Orca/GitKraken:
- **GitHub:** `make connect-github` usa a conta já logada no GitHub CLI (`gh auth login`, que abre o navegador; o token fica no Keychain do `gh`) e guarda o token no cofre. A fábrica age com as permissões dessa conta. É feito no host porque a API roda em container e não enxerga o Keychain. Re-executar atualiza o token.
- **Jira:** formulário como o do Orca: URL do site, e-mail e token de API (segredo `email:token`). **ClickUp:** token pessoal da API. Tokens de API com escopo e sem e-mail (Bearer) NÃO são suportados.
- Os fluxos OAuth abaixo (device flow do GitHub, redirecionamento de ClickUp/Jira) continuam no código, mas **só aparecem na tela se o app OAuth estiver no `.env`**; são opcionais.

## Decisão original (opcional, desligada por padrão)
1. **GitHub: device flow** (RFC 8628). A API pede o código ao GitHub, a tela mostra `user_code` + link, e a API consulta o token respeitando `interval` (e `slow_down`). Só `GITHUB_OAUTH_CLIENT_ID` (OAuth App com Device Flow). Escopos: `repo read:org workflow` (commits, push e PRs).
2. **ClickUp e Jira: fluxo com redirecionamento**, pois não oferecem device flow. Callback único `OAUTH_REDIRECT_BASE/api/oauth/callback`. Exigem `client_id` e `client_secret` no `.env`.
3. **O token vai direto ao cofre** (`Vault.Create`), nunca à tela nem a logs; segue `connection:<id>`, redação e demais regras do ADR-0005. Iniciar um fluxo é ação sensível (reautenticação + CSRF).
4. **Callback público.** O cookie `SameSite=Strict` não acompanha o retorno do provedor, então a rota não usa sessão: o `state` (256 bits, uso único, 10 min, só em memória) é a prova de origem; sem `state` válido nada é trocado. O resultado vai por 303 para `/connections?oauth=ok|failed`.
5. **Jira (3LO):** o segredo guardado é JSON (`access_token`, `refresh_token`, `expires_at`, `cloud_id`); as chamadas vão ao gateway `https://api.atlassian.com/ex/jira/<cloudid>` com `Bearer`. O token vale 1 h e o refresh token é rotativo: `Vault.SetRefresher` renova dentro de `Resolve` (API e worker) e regrava o segredo sem mexer em `version`. Segredos `email:token` antigos continuam funcionando.
6. **ClickUp:** o token OAuth não expira e usa o mesmo cabeçalho do token pessoal, sem mudança nos adaptadores.
7. A tela deixa de aceitar colar credencial de GitHub/Jira/ClickUp; `cli`, `api` e `mcp` continuam manuais (sem OAuth aplicável).

## Limites
- Fluxos pendentes ficam em memória: reiniciar a API cancela, e o usuário repete.
- Revogar no cofre apaga o token local; revogar no provedor (Settings do GitHub etc.) é manual.
- Não usamos o `client_id` público de CLIs de terceiros: cada instalação registra o seu OAuth App.
- Autorização real contra os provedores NOT_RUN até os apps serem criados.
