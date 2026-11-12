> Leitura: [← Anterior](04-recovery-operation.md) · [Índice didático](../02-INDEX.md) · [Próximo →](06-usage-dashboard.md)

# Selecionar cliente e transferir progresso

IMPLEMENTADO em a2cc5e0; evidência histórica por FAC-010C/012V/X/Y/AA. Verificação de software NOT_RUN nesta migração; aceite por ticket permanece no backlog real.

## Usuário e comportamento

Operador administrativo da fábrica. Configuração atual por função, instalação/modelo elegíveis; indisponibilidade conhecida após stop preserva snapshot. Até duas alternativas explícitas, outra worktree, restore/contexto; sem provider, WAITING_PROVIDER.

## Regras e critério

Critério: unknown não troca writer; conta/modelo não configurados não viram fallback. Handoff interno conserva lease/fence da tentativa; resume abre outra.

## Falhas e segurança

Auth/versão/digest/estado inválido interrompe a operação pertinente; informação do modelo não concede autoridade. Sem API de IA/extras/fallback/recarga. Gates de implantação e UID são independentes de comportamento codificado.

## Relações e evidências

[providers](../03-modulos/providers/00-README.md), [runtime](../03-modulos/runtime/00-README.md), [runtime-guard](../03-modulos/runtime-guard/00-README.md). [Contratos](../05-contratos/00-README.md), [ADRs](../06-decisoes/00-README.md), [entregas](../09-entregas/00-README.md), [aceites reais](../08-desenvolvimento/09-backlog.md).

## Limite

[AS-IS versus alvo](../02-arquitetura/01-as-is.md). Cenários de tenancy e broker dependem das propostas LF-MT; o catálogo FAC não prova seus controles.
