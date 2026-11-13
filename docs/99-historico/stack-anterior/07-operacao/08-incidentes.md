> Leitura: [← Anterior](07-troubleshooting.md) · [Índice didático](../02-INDEX.md) · [Próximo →](09-economia.md)

> Integração DOC-MV-001: texto de direção/planejamento do pacote. O [AS-IS atual](../02-arquitetura/01-as-is.md) prevalece para implementação. Não confundir proposta com controle instalado.

> AS-IS: pausa global e PAUSE/CANCEL por run implementados. Revogação de capabilities/tenant é proposta posterior sem serviço real localizado. Responsável de plantão não definido nesta migração.

# Resposta a incidente

Revisão documental: 2026-10-06 (America/Sao_Paulo). Arquitetura **PLANEJADA**; implementação e eficácia **NÃO VERIFICADAS** na origem do pacote.

Responsável administrativo de plantão: a definir no ambiente real. Kill switch global/tenant nega novos claims, revoga capabilities e interrompe processos. Congelar artefatos em quarentena e preservar evidências sanitizadas.

Revogar/renovar auth afetada pelo fluxo oficial e credenciais internas pelo mecanismo administrativo. Investigar alcance/ownership e impacto sem afirmar que isolamento permaneceu intacto. Não enviar segredos/logs completos a modelos. Corrigir causa em ticket, repetir testes afetados e reativar somente com gate/aceite. Registrar tempos, impacto, causa confirmada/hipóteses, medidas, limites e ação preventiva; lesson só se aprendizado foi realmente aplicado.

## Proveniência

- [Threat model original v3](../02-arquitetura/seguranca/00-threat-model.md)
