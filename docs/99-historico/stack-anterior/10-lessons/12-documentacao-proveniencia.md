> Leitura: [← Anterior](11-baseline-regressao-review.md) · [Índice didático](../02-INDEX.md) · [Próximo →](13-modal-edicao-configuracao.md)

# Proveniência preservada antes da conciliação

Aplicado documentalmente em DOC-MV-001, base a2cc5e0, 2026-10-07. Sem aprendizado de software inventado.

## O que é e por que usamos

Documentação atual e histórico respondem perguntas diferentes. Uma cópia mais nova de planejamento não prova implementação nem pode apagar decisão aceita; hashes demonstram quais bytes foram preservados, não veracidade técnica.

## Exemplo aplicado

ADR-003-stack-typescript ACEITO local e ADR-003-multi-tenant-e-stack PROPOSTO do pacote foram preservados com número/status/origem, colisão explícita no índice e snapshots integrais. Controllers mostram READY/claim, enquanto ESPEC histórica propunha POST runs; contrato atual explica implementação sem inventar alias nem reescrever o original.

MANIFESTO-FONTES conserva SHA-256/bytes/origem; MAPA-SECOES-REPO mapeia cada trecho integral e destino/tratamento. Paths antigos viraram bridges após capítulo canônico e cópia imutável; o pedido posterior removeu os individuais redundantes, preservando entradas essenciais. Validador estrutural confere hash/cobertura/link, revisão semântica usa código real.

## Cuidados e limites

Hash não prova que o desenho funciona, extração PDF não preserva layout, autoload não segue de um link. Template não é evidência. Alteração de código/byte invalida verificação pertinente; snapshots nunca se alteram para esconder conflito. Documentar intenção PLANEJADA, código IMPLEMENTADO e ensaio NOT_RUN separadamente.

[Evidências/entrega](../09-entregas/2026/2026-10-07-DOC-MV-001-migracao-manual-vivo.md), [ADRs](../06-decisoes/00-README.md), [contratos](../05-contratos/api/01-controle.md).
