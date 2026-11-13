> Leitura: [← Anterior](03-documentation-approval.md) · [Índice didático](../02-INDEX.md) · [Próximo →](05-provider-handoff.md)

# Pausar e recuperar execução

IMPLEMENTADO em a2cc5e0; evidência histórica por FAC-012K/N–U/AF/AG. Verificação de software NOT_RUN nesta migração; aceite por ticket permanece no backlog real.

## Usuário e comportamento

Operador administrativo da fábrica. PAUSE/CANCEL são intenção administrativa; worker confirma stop e preserva journal/snapshot. Resume usa bundle íntegro e novo fence; recover-finalization só reenvia resultado parado sem IA.

## Regras e critério

Critério: crash unknown mantém bloqueio; lease vencida não libera exclusão. Backup restaura em staging exclusivo, sem overwrite de destino.

## Falhas e segurança

Auth/versão/digest/estado inválido interrompe a operação pertinente; informação do modelo não concede autoridade. Sem API de IA/extras/fallback/recarga. Gates de implantação e UID são independentes de comportamento codificado.

## Relações e evidências

[runs](../03-modulos/runs/00-README.md), [sandbox](../03-modulos/sandbox/00-README.md), [worker](../03-modulos/worker/00-README.md). [Contratos](../05-contratos/00-README.md), [ADRs](../06-decisoes/00-README.md), [entregas](../09-entregas/00-README.md), [aceites reais](../08-desenvolvimento/09-backlog.md).

## Limite

[AS-IS versus alvo](../02-arquitetura/01-as-is.md). Cenários de tenancy e broker dependem das propostas LF-MT; o catálogo FAC não prova seus controles.
