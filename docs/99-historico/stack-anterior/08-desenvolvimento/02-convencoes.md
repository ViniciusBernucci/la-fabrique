> Leitura: [← Anterior](01-ambiente-local.md) · [Índice didático](../02-INDEX.md) · [Próximo →](03-como-contribuir.md)

> Integração DOC-MV-001: texto de direção/planejamento do pacote. O [AS-IS atual](../02-arquitetura/01-as-is.md) prevalece para implementação. Não confundir proposta com controle instalado.

# Convenções de engenharia e documentação

Revisão documental: 2026-10-06 (America/Sao_Paulo). Arquitetura **PLANEJADA**; implementação e eficácia **NÃO VERIFICADAS** na origem do pacote.

Aplicar stack aprovada da fábrica; preservar stack somente de projeto externo, escopo pequeno e caminhos autorizados. Um writer por workspace. Não concatenar prompt em shell; adapters usam argv/stdin e binário/versão administrados. Identidade e política derivam de canal confiável, não input do modelo.

Markdown em português didático, nomes kebab-case; conceitos técnicos mantêm nome exato da entidade. Links relativos dentro do pacote, fonte canônica em vez de cópia. Mermaid descreve nível/estado/fronteira. Docs atuais evoluem; entregas/histórico são append-only com correções identificadas.

ADR aceito não é reescrito materialmente; criar relação de substituição. IDs/data/SHA reais, sem hash circular. Não criar testes que só espelham edição editorial; validar links/proveniência/cobertura para documentação. Regras detalhadas na [política](../00-governanca/01-POLITICA-IA.md).
