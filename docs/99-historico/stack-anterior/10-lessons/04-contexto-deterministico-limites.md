> Leitura: [← Anterior](03-outbox-idempotencia.md) · [Índice didático](../02-INDEX.md) · [Próximo →](05-checkout-confiavel.md)

# Contexto deterministico e limites conservadores

Contexto reproduzivel precisa guardar identidade dos bytes, nao apenas uma lista de nomes. No FAC-006, `ContextBuilder` ordena caminhos, calcula SHA-256 por fonte e deriva o hash do manifesto da revisao-base, fontes aceitas e omissoes. Assim, trocar um arquivo ou ultrapassar um limite invalida o manifesto de forma observavel.

Omissao deve ser dado de primeira classe. `context-builder.ts` registra motivos como `SECRET_DETECTED`, `BINARY` e `TOTAL_BYTES_LIMIT` e marca `truncated`; ele nao apresenta um pacote parcial como completo. A selecao semantica continua responsabilidade do chamador porque hash prova identidade, nao suficiencia.

Limites tambem precisam parar ciclos, nao apenas contar recursos. `RuntimeGuard` pausa a segunda falha consecutiva com a mesma impressao e conserva a decisao no estado validado. Esse comportamento evita repetir um diagnostico identico ate esgotar tentativas ou cota.
