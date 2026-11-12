# DOC-ADR-001 — Manual Vivo e fonte canônica

Data: 2026-10-06. Status: PROPOSTA PARA INTEGRAÇÃO. Identificador documental separado; verificar colisão no repo real. Pedido de reorganização atendido neste pacote; adoção no repo ainda não executada.

## Contexto e alternativas

O espelho mistura planejamento, política repetida nos três guias, PDF com capítulos ausentes em disco e pacote multi-tenant posterior. Alternativas: manter tudo em PDF/monólito; manter relatórios por ticket como única documentação; ou organizar estado atual navegável e histórico separado.

## Decisão proposta

Markdown no Git como fonte editável, Manual Vivo didático, C4 de três níveis, módulos/features/contratos/ADRs/operação/desenvolvimento/entregas/lessons. Uma política comum; wrappers curtos por agente. Estado atual aponta fontes especializadas; histórico preserva origem e decisões superadas.

## Justificativa e consequências

Leitura por objetivo, revisão junto ao código e menor divergência. Custo: manutenção de links/mapeamento/gate e revisão semântica. Arquivar não basta para migrar comportamento útil: cada seção deve ter destino atual ou justificativa histórica. Não criar arquivos vazios só para preencher árvore.

## Revisão

Reavaliar após migração no repo e primeiras entregas: tempo de leitura, drift, taxa de links quebrados e custo de manutenção. PDF futuro é export derivado da mesma revisão; não autoridade paralela.
