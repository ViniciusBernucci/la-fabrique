# ADR-0064 — Conexão revogada libera o nome e pode ser apagada

- **Estado:** ACEITA · **Relacionadas:** [ADR-0005](0005-autenticacao-sessoes-e-cofre-de-conexoes.md), [ADR-0063](0063-guardar-conexao-sem-reautenticacao.md)

## Contexto
O nome da conexão era único entre todas as linhas, inclusive as revogadas, e não havia como apagar. Reconectar uma conta (ex.: GitHub com token recusado) obrigava a inventar outro nome e deixava a conexão morta na lista. O responsável pediu poder apagar a antiga ou, ao revogar e criar outra, tê-la substituída.

## Decisão
1. **Migration 00067:** a unicidade do nome passa a valer só entre conexões **ativas** (índice parcial `WHERE state = 'active'`). Revogar libera o nome; a nova conexão pode usar o mesmo.
2. **`DELETE /api/connections/{id}`** (sessão + CSRF, sem reautenticação, como o ADR-0063) apaga uma conexão **já revogada**. Conexão ativa é recusada (409 `connection_active`: revogue antes). Conexão ainda referenciada por projeto, agente ou MCP é recusada (409 `connection_in_use`), pelas chaves estrangeiras `RESTRICT`. O segredo já foi apagado na revogação. A ação é auditada (`connection_deleted`).
3. **Tela de Conexões:** a conexão revogada mostra **Apagar**, com confirmação.
4. A nova conexão **não** assume sozinha os vínculos da antiga: projetos e agentes escolhem a nova conexão. Para manter vínculos com o mesmo destino, use *Substituir segredo* (mantém o id).

## Limites
- Apagar não tem desfazer. Registros que referenciam por `ON DELETE SET NULL` ou `CASCADE` (uso/orçamento e disponibilidade da conexão) são afetados pela exclusão.
- Duas linhas com o mesmo nome podem coexistir (uma revogada, outra ativa) até a antiga ser apagada.
