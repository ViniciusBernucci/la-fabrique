> Leitura: [Índice didático](../02-INDEX.md) · [Próximo assunto →](../07-operacao/00-README.md)

# Decisões e propostas

| Registro | Status original | Tratamento nesta migração |
|---|---|---|
| ADR-001 | NÃO LOCALIZADO no repo/pacote | Não renumerar |
| [ADR-002 VPS única](ADR-002-vps-unica.md) | ACEITA 2026-09-29, implementação incremental descrita no original | Texto local integral, somente links editoriais |
| [ADR-003 TypeScript](ADR-003-stack-typescript.md) | ACEITO 2026-09-29, revisão 2.3 | Fonte da stack vigente; preserva evoluções FAC/OPS |
| [ADR-003 multi-tenant do pacote](ADR-003-multi-tenant-e-stack.md) | PROPOSTA DOCUMENTADA 2026-10-06 | Colisão mantida por nome/origem; não substitui ADR-003 aceito, não recebe novo número sem decisão |
| [ADR-004](ADR-004-provider-runtime-sem-segredos-no-sandbox.md) | PROPOSTA | Auth segregada e broker; sem liberação operacional |
| [ADR-005](ADR-005-sandbox-efemero-e-perfil-de-isolamento.md) | PROPOSTA | Perfil posterior de isolamento |
| [ADR-006](ADR-006-rls-capabilities-e-egress.md) | PROPOSTA | RLS/capabilities/egress não implementados |
| [DOC-ADR-001](DOC-ADR-001-manual-vivo.md) | PROPOSTA PARA INTEGRAÇÃO no pacote | Organização integrada pelo pedido; aceite documental exato pendente |

Não reescrever status/decisão original para descrever HEAD. [AS-IS](../02-arquitetura/01-as-is.md) e [auditoria](../00-governanca/08-AUDITORIA.md) registram evolução e diferenças. Substituição futura exige ADR próprio explícito. Snapshots guardam bytes anteriores aos links editoriais.

## Identidade humana da conta

[ADR-007 — Conta do usuário e sessões individuais](ADR-007-conta-usuario-e-sessoes.md) registra a implementação focal FAC-038, sem atribuir projetos/configuração global a contas novas e sem substituir a direção LF-MT. Status PROPOSTA IMPLEMENTADA; revisão/aceite/implantação pendentes.

## Memória e decisões — avaliação de 2026-10-09

- [ADR-007](ADR-007-memoria-e-decisoes-versionadas.md): estrutura proposta; requisitos confirmados identificados separadamente.
- Colisão de número: este ADR-007 (ARQ-MEM-001) e o [ADR-007 da conta do usuário](ADR-007-conta-usuario-e-sessoes.md) (FAC-038) foram criados em paralelo. Ambos mantidos por nome/origem, como no caso do ADR-003; não recebem novo número sem decisão.
- [ADR-008](ADR-008-excecao-jev.md): exceção específica JEV confirmada, integração futura e desenho técnico proposto.
