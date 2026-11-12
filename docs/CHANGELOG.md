# Changelog

## 2026-11-12 — E05: cadastro de projeto em rascunho (T05-01)
Assistente de cadastro por etapas (Dados, Repositório, Origem, Revisão) com persistência por etapa, relatório de validação com ação por pendência, ativação bloqueada sem preenchimento válido, suspensão/retomada e trilha de alterações. T05-02 e T05-03 seguem pendentes (dependem de E06 a E09). Corrige também o `make check-secrets`, que falhava desde o E04. Detalhes: [entrega](entregas/2026-11-12-E05-cadastro-de-projeto.md) · [ADR-0006](adr/0006-cadastro-de-projeto-em-rascunho.md).

## 2026-11-12 — E04: acesso e credenciais (T04-01, T04-02, T04-03 parcial)
Login com senha + TOTP, códigos de recuperação de uso único, sessões revogáveis, reautenticação, CSRF, HTTPS obrigatório em produção e cofre de conexões cifrado com chave externa ao banco. **Removido** o acesso dev sem senha. Detalhes: [entrega](entregas/2026-11-12-E04-acesso-e-credenciais.md) · [ADR-0005](adr/0005-autenticacao-sessoes-e-cofre-de-conexoes.md).

## 2026-11-12 — Nova stack: fundação (E01, E02 parcial, E03)
Stack Go + React + PostgreSQL + Docker substitui NestJS/Redis. Skills novas instaladas. Front limpo no design "Novo". Detalhes: [entrega](entregas/2026-11-12-E01-E03-fundacao.md).
