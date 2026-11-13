# FAC-012I — Perfil de execução do projeto

Status: implementado, aguardando aceite humano.

`ProjectDefinition.executionProfile` guarda fontes explícitas de contexto (caminho relativo e papel) e checks autorizados para execução autônoma. O painel separa checks cadastrados de checks aprovados; editar nome, executável ou argv remove a aprovação. Fontes devem estar sob `allowedPaths` e não sobrepor `forbiddenPaths`; checks aprovados devem corresponder exatamente aos candidatos. Definições antigas são lidas como perfil nulo.

O endpoint de READY exige perfil não nulo antes de qualquer mutação. O snapshot já autocontido em `executionSpecification` inclui o perfil junto da versão imutável. No worker, o perfil confiável de contexto/checks é comparado ao snapshot; divergência falha fechada e somente checks aprovados são compilados. Contas, credenciais, providers e modelos continuam exclusivamente no Centro de Configurações.

Não há migration: a definição é persistida em JSON. Não houve alteração de banco/fila real nem execução de checkout, consumer, cliente IA ou piloto. O consumer segue em probe porque ainda faltam resolução confiável do checkout local e integração real de lease/fencing/writer.

Verificação: `npm test` (190 testes, incluindo 2 do launcher); `npm run typecheck`; `npm run lint`; `npm run build`; `git diff --check` — todos passaram. O ticket aguarda revisão humana da revisão exata.
