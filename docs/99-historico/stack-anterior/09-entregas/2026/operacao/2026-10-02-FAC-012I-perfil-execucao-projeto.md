# FAC-012I — Perfil de execução por projeto

Status: implementado em branch isolada; aguardando aceite humano.

## Mudança

- Contrato compartilhado aceita `executionProfile: null` para definições antigas; perfil preenchido valida fontes únicas, caminhos permitidos/proibidos e checks candidatos exatos.
- Painel lista arquivos de contexto com papéis e exige aprovação explícita individual de cada check; editar identidade/comando/argv do check desmarca sua aprovação.
- API recusa `DRAFT -> READY` antes de mutar ticket/outbox quando não há fontes e checks aprovados. O snapshot READY captura o perfil.
- Compilador do worker exige igualdade entre fontes/checks aprovados do snapshot e a resolução confiável local; comandos cadastrados mas não aprovados não entram no workflow.

## Verificações

`npm test`: passou — 21 contracts, 40 runtime, 55 API, 67 worker, 5 web e 2 launcher (190 total). `npm run typecheck`, `npm run lint`, `npm run build` e `git diff --check`: passaram. A primeira rodada de lint identificou apenas formatação; arquivos foram formatados e nova rodada passou.

## Limites e rollback

Sem migration; não foi necessário alterar dados persistidos. Sem banco/fila real, provider, consumer, checkout ou piloto. Reverter o commit de implementação/documentação restaura a definição anterior; perfis JSON escritos por esta versão precisariam ser preservados/exportados antes de downgrade da aplicação. Mantém consumer em probe: checkout operacional, lease/fencing/writer e identidade efetiva continuam pendentes.

## Revisão

Revisão exata aguarda aceite humano. Não houve merge, push ou deploy.
