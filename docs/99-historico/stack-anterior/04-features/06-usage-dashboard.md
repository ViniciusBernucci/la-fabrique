> Leitura: [← Anterior](05-provider-handoff.md) · [Índice didático](../02-INDEX.md) · [Próximo →](07-tenant-isolation.md)

# Observar estado e uso

IMPLEMENTADO em a2cc5e0; evidência histórica por FAC-012M/AD/AE/AG/020. Verificação de software NOT_RUN nesta migração; aceite por ticket permanece no backlog real.

## Usuário e comportamento

Operador administrativo da fábrica. Painel Estado da fábrica consulta heartbeat, exclusão global e pausa; SSE de projeto retoma cursor com fallback HTTP; histórico mostra checks/chamadas/modelo/uso nullable.

## Regras e critério

Critério: uso desconhecido não vira zero/estimativa; home demonstrativa não representa observação operacional.

## Falhas e segurança

Auth/versão/digest/estado inválido interrompe a operação pertinente; informação do modelo não concede autoridade. Sem API de IA/extras/fallback/recarga. Gates de implantação e UID são independentes de comportamento codificado.

## Relações e evidências

[controle](../03-modulos/controle/00-README.md), [observabilidade-economia](../03-modulos/observabilidade-economia/00-README.md). [Contratos](../05-contratos/00-README.md), [ADRs](../06-decisoes/00-README.md), [entregas](../09-entregas/00-README.md), [aceites reais](../08-desenvolvimento/09-backlog.md).

## Limite

[AS-IS versus alvo](../02-arquitetura/01-as-is.md). Cenários de tenancy e broker dependem das propostas LF-MT; o catálogo FAC não prova seus controles.
