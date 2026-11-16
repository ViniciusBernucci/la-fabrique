# ADR-0063 — Guardar uma conexão nova não exige reautenticação

- **Estado:** ACEITA · **Relacionadas:** [ADR-0005](0005-autenticacao-sessoes-e-cofre-de-conexoes.md)

## Contexto
O ADR-0005 listava criar conexão entre as ações sensíveis (janela de reautenticação com senha + TOTP). O responsável pediu que guardar uma conexão não peça autenticação de novo. Pela regra de autoridade do projeto, a decisão atual prevalece e a mudança fica registrada aqui.

## Decisão
1. `POST /api/connections` passa de `sensitive` para `protected`: exige sessão válida e CSRF, mas não a janela de reautenticação.
2. Rotacionar e revogar conexão continuam sensíveis (`reauth_required` sem janela), assim como os demais itens do ADR-0005.
3. O segredo continua só no cofre, nunca devolvido nem logado.

## Limites
- Quem controla uma sessão aberta consegue cadastrar uma credencial nova sem confirmar senha. Não consegue ler as existentes, rotacioná-las nem revogá-las sem reautenticar.
- Para voltar atrás: trocar `protected` por `sensitive` na rota e restaurar o `runSensitive` no formulário.
