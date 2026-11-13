> Leitura: [← Anterior](06-restore.md) · [Índice didático](../02-INDEX.md) · [Próximo →](08-incidentes.md)

> Integração DOC-MV-001: texto de direção/planejamento do pacote. O [AS-IS atual](../02-arquitetura/01-as-is.md) prevalece para implementação. Não confundir proposta com controle instalado.

# Diagnóstico e recuperação

Revisão documental: 2026-10-06 (America/Sao_Paulo). Arquitetura **PLANEJADA**; implementação e eficácia **NÃO VERIFICADAS** na origem do pacote.

| Sintoma | Tratamento planejado | Evidência antes de retomar |
|---|---|---|
| Worker/rede indisponível | Interromper antes de lease, aguardar/reconciliar | Quiescência, heartbeat/ownership |
| Auth expirada | AUTH_REQUIRED, login oficial administrativo | Preflight por instalação/versão |
| Quota esgotada | Handoff seguro ou WAITING_PROVIDER | Fonte/horário/limite observado |
| Disco/OOM | PAUSED_RESOURCE, preservar e limpar só recursos comprovados | Limites/ownership/controle saudável |
| CLI mudou schema | Bloquear adapter, atualizar em ticket | Fixtures e preflight novo |
| Policy/broker/proxy falhou | Negar operações e admission | Configuração validada; sem bypass |
| RESULT_UNKNOWN | Reconciliar efeito/consumo antes de repetir | Resultado observável/checkpoint |
| Writer antigo não confirmado parado | BLOCKED_RECOVERY | Prova externa de fim da árvore |
| TOOL_DENIED | Verificar escopo/perfil e registrar diagnóstico | Não ampliar privilégio automaticamente |

Procedimentos e comandos reais dependem de instalação. Registrar baseline versus regressão e usar no máximo duas correções antes de checkpoint/diagnóstico.
