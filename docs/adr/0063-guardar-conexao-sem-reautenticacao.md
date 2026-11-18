# ADR-0063 — Criar, rotacionar e revogar conexão não exigem reautenticação

- **Estado:** ACEITA · **Relacionadas:** [ADR-0005](0005-autenticacao-sessoes-e-cofre-de-conexoes.md)

## Contexto
O ADR-0005 listava criar conexão entre as ações sensíveis (janela de reautenticação com senha + TOTP). O responsável pediu que guardar uma conexão não peça autenticação de novo e, em seguida, o mesmo para revogar e substituir o segredo. Pela regra de autoridade do projeto, a decisão atual prevalece e a mudança fica registrada aqui.

## Decisão
1. `POST /api/connections`, `.../{id}/rotate` e `.../{id}/revoke` passam de `sensitive` para `protected`: exige sessão válida e CSRF, mas não a janela de reautenticação.
2. Continuam sensíveis os demais itens do ADR-0005 (substituir 2FA, encerrar outra sessão) e as outras ações do painel.
3. O segredo continua só no cofre, nunca devolvido nem logado.

## Limites
- Quem controla uma sessão aberta consegue cadastrar uma credencial nova sem confirmar senha. Também consegue rotacionar ou revogar as existentes. Não consegue ler o valor de nenhum segredo, que nunca volta pela API.
- Para voltar atrás: trocar `protected` por `sensitive` nas três rotas e restaurar o `runSensitive` em `ConnectionsPage.tsx`.
