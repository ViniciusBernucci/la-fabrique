> Leitura: [← Decisão anterior](ADR-007-memoria-e-decisoes-versionadas.md) · [Catálogo de decisões](00-README.md) · [Índice didático](../02-INDEX.md)

# ADR-008 — Exceção específica para API JEV

Data: 2026-10-09, Europe/Berlin. **Exceção de política CONFIRMADA pelo responsável nesta sessão; desenho técnico PROPOSTO PARA AVALIAÇÃO.** Integração PLANEJADA para tarefa posterior; operação NÃO VERIFICADA. Não altera a stack aceita.

## Contexto e decisão confirmada

A política subscription-only da fábrica mantém APIs de IA e extras desligados. O responsável decidiu abrir exceção específica para JEV, documentando o necessário à integração e definindo limites de gasto/chamadas pelo sistema. A exceção modifica apenas essa restrição para o componente JEV; não libera APIs de LLM, compras, recarga ou gastos sem limites configurados.

JEV será candidato para ambiguidades semânticas entre alternativas permitidas. Ausência ou resposta inconclusiva encaminha para LLM elegível por cliente oficial e depois responsável. Bloqueios de política não são resolvidos por modelo. Local ML não é pré-condição inicial para JEV.

## Desenho proposto e alternativas

Configuração separada, adapter no worker, credencial privada por referência, payload sanitizado, perguntas/versões/candidatos auditados, reserva de orçamento e resultado desconhecido conservador. HTTP TypeScript versus SDK compatível fica para avaliação. Manter JEV desligado evita custo mas não atende disponibilidade futura desejada; chamar diretamente pelo agente exporia autoridade/credencial e não é recomendado.

## Consequências e gates

O responsável preencherá orçamento/moeda/janelas/chamadas e parâmetros de operação no sistema. Não há valores aceitos nesta sessão. Sem configuração válida e mecanismo de controle comprovado, nenhuma chamada paga. Não alterar guards subscription-only dos clientes oficiais para acomodar JEV.

Fonte técnica pública verificada em 2026-10-09: [quick start TypeSafe](https://docs.typesafe.ai/introduction/quickstart). Contrato/tarifa/modelo/condições de idempotência precisam ser revalidados antes da integração. Esta entrega não cria conta, lê credencial ou chama API autenticada.

[Plano específico](../08-desenvolvimento/planos-memoria-e-decisoes/07-jev.md). Rollback operacional futuro desativa novas chamadas e conserva contabilidade/auditoria, sem presumir recuperação de custo desconhecido. Aceite dos detalhes técnicos e da entrega documental permanece pendente.
