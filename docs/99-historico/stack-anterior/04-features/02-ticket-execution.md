> Leitura: [← Anterior](01-control-settings.md) · [Índice didático](../02-INDEX.md) · [Próximo →](03-documentation-approval.md)

# Executar ticket configurado

IMPLEMENTADO em a2cc5e0; evidência histórica por FAC-001A/003A/008/012A–L/AC–AH. Verificação de software NOT_RUN nesta migração; aceite por ticket permanece no backlog real.

## Usuário e comportamento

Operador administrativo da fábrica. Cadastro de projeto/definição com SHA e checks aprovados; READY congela intenção; claim/admissão/writer/fence; worker prepara e executa workflow; checks, Reviewer read-only e docs antes de entrega.

## Regras e critério

Critério: payload inválido ou base não resolvida não publica job; reentrega não cria writer concorrente.

## Falhas e segurança

Auth/versão/digest/estado inválido interrompe a operação pertinente; informação do modelo não concede autoridade. Sem API de IA/extras/fallback/recarga. Gates de implantação e UID são independentes de comportamento codificado.

## Relações e evidências

[projects](../03-modulos/projects/00-README.md), [tickets](../03-modulos/tickets/00-README.md), [orchestrator](../03-modulos/orchestrator/00-README.md), [worker](../03-modulos/worker/00-README.md). [Contratos](../05-contratos/00-README.md), [ADRs](../06-decisoes/00-README.md), [entregas](../09-entregas/00-README.md), [aceites reais](../08-desenvolvimento/09-backlog.md).

## Limite

[AS-IS versus alvo](../02-arquitetura/01-as-is.md). Cenários de tenancy e broker dependem das propostas LF-MT; o catálogo FAC não prova seus controles.
