> Leitura: [← Anterior](01-POLITICA-IA.md) · [Índice didático](../02-INDEX.md) · [Próximo →](03-GUIA-DE-INTEGRACAO.md)

# Política do Manual Vivo e docs-as-code

## Objetivo e fontes canônicas

Uma pessoa nova deve compreender: o que é, como funciona, por que foi decidido, como operar/modificar e o que mudou. Markdown versionado junto ao código é a fonte editável. PDF é export da mesma revisão, não segunda autoridade. Não existe obrigação de gerar PDF em toda entrega.

| Conhecimento | Casa canônica |
|---|---|
| Introdução/trilha | docs/01-README.md e LEIA-ME-PRIMEIRO |
| Escopo/glossário | docs/01-visao-geral |
| C4/visão transversal | docs/02-arquitetura |
| Responsabilidade e comportamento de módulo | docs/03-modulos/<modulo> |
| Comportamento de produto | docs/04-features |
| Interfaces/entidades/eventos/filas | docs/05-contratos (schema de código quando existir) |
| Razão arquitetural | docs/06-decisoes |
| Procedimento operacional | docs/07-operacao |
| Setup/testes/contribuição/planejamento | docs/08-desenvolvimento; backlog.md nesta área |
| Mudança histórica por entrega | docs/09-entregas/<ano>/AAAA-MM-DD-TICKET-titulo.md |
| Conceito efetivamente aplicado | docs/10-lessons/<conceito>.md |
| Governança/auditoria/templates | docs/00-governanca |
| Origem/superado | docs/99-historico |

[docs/04-CHANGELOG.md](../04-CHANGELOG.md) é o único registro resumido; [backlog](../08-desenvolvimento/09-backlog.md) preserva trabalho e aceites reais. Piloto/plano/controle/matriz ficam em desenvolvimento; especificação em contratos; fontes e guia de integração em governança; prompts em prompts/. A raiz mantém somente README e os três guias de agentes; não recriar atalhos removidos. INDEX é inventário navegável, Manual ensina; não copiar definições entre eles.

## Matriz de impacto por entrega

| Mudança | Atualização de estado atual | Histórico e adicionais |
|---|---|---|
| Nova feature/regra | módulo + feature afetada | entrega; contrato/dados se afetados |
| API/evento/fila/schema | contrato + módulo/feature consumidor | entrega; compatibilidade/versionamento |
| Arquitetura | C4/arquitetura/módulos afetados | ADR relevante + entrega |
| Entidade/persistência | contrato de dados + módulo | migração/rollback + entrega |
| Infra/segurança | arquitetura/runbook/módulo/testes afetados | ADR se decisão estrutural + entrega |
| Ambiente/build/CI | desenvolvimento/contratos pertinentes | entrega |
| Bug com mudança de comportamento | módulo/feature/regra pertinente | entrega |
| Refatoração interna ou bug sem mudança pública | estado atual somente se explicação deixou de corresponder | entrega; justificar não impacto |
| Edição documental | páginas/índices/links afetados | registro documental; nenhum teste de aplicação inventado |

Sempre avaliar índice/changelog/backlog/matriz; atualizar quando afetados, sem churn para reproduzir nada. Toda entrega possui registro; não repetir relatórios por domínio para um mesmo ticket — um registro liga os módulos. Lessons condicionais a aplicação real. ADR somente para decisão relevante, não para registrar todo patch.

## Padrão didático de módulo e feature

Começar com finalidade e exemplo simples; desenvolver responsabilidade/limites, fluxo/estados, regras, entidades, integração/contrato, falhas/recuperação, segurança, código real, ADR/features e evidência/limitações. README é suficiente para módulo pequeno; separar capítulos quando necessário. Linkar contrato em vez de copiar JSON/OpenAPI. Diagramas informam nível, fronteira e status. Exemplos sintéticos são explicitamente exemplos; não viram evidência.

Feature descreve comportamento e critérios atuais; registro de entrega relata mudança. Não usar somente tickets em sequência para explicar funcionamento atual. Não criar documento vazio para cada pasta da árvore desejada.

## Histórico, migração e decisões

Manter snapshot/hash/proveniência e mapa origem/seção/destino/tratamento. Conteúdo útil deve ser consolidado no capítulo atual ou mantido como histórico com justificativa. Arquivar sozinho não fecha migração de uma regra vigente. Duplicações exatas podem virar atalho após comparar; diferenças não são descartadas silenciosamente.

Documentos atuais não repetem stack/rotas/estados contraditórios sem explicar resolução. Intenção aprovada e implementação observada são dimensões separadas; divergência de código é registrada, não corrigida com migração técnica automática. ADR aceito é preservado; nova decisão relaciona substituição. Não renumerar IDs históricos para preencher buraco. Correção factual identificada é permitida no histórico, sem apagar versão original.

## Gate documental

1. Impacto identificado, atual e histórico coerentes; fontes e revisão identificadas.
2. Links locais existentes e anchors válidos; Mermaid/fences legíveis e fechados; índice atualizado.
3. Registro descreve objetivo/problema/solução/funcionamento/arquivos/diff/dados/API/testes/riscos/limites/rollback/docs/lessons/uso/aceite.
4. Checks com critério, revisão, comando/procedimento, resultado e evidência; NOT_RUN explícito; sem placeholders em alegação IMPLEMENTADO.
5. Revisão semântica confirma código/contratos/docs; presença de arquivo não prova correção.
6. Sem secrets/dumps/PII no diff/log/context. Aceite vinculado à revisão/config; alterações invalidam evidência afetada.

O validador deste pacote verifica estrutura e hashes, não veracidade do software ou segurança. Instalar gate de CI real é trabalho de integração, sem alegar já existente. Leia [checklist](04-CHECKLIST-ENTREGA.md) e [templates](templates/00-README.md).

## Ordem didática obrigatória dos arquivos

Toda documentação corrente deve conduzir a leitura como um manual. Organizar por pré-requisitos de entendimento: problema e objetivos → vocabulário e escopo → contexto e estrutura → funcionamento e regras → interfaces e decisões → operação e desenvolvimento. Ordem de implementação, data de criação ou sequência de tickets não determina capítulos do manual.

1. Capítulos correntes usam `NN-titulo.md`, com dois dígitos e posição única na pasta. A introdução de cada área é `00-README.md`; pastas de referência podem começar pelo catálogo ou conceito base numerado `00-`. A entrada global é `00-LEIA-ME-PRIMEIRO.md`, seguida por `01-README.md`.
2. Definir a posição pelo que o leitor precisa conhecer antes. Cada introdução lista os capítulos em ordem, com links; cada capítulo tem anterior/próximo e retorno ao índice. Explicar propostas depois do estado atual; material de referência herdado fica ao final.
3. O índice principal agrupa capítulos por assunto em listas numeradas; registros de tickets/entregas e fontes históricas são apêndices de consulta, fora da leitura inicial. Não gerar o roteiro simplesmente por ordem alfabética ou por implementação.
4. ADRs conservam números/IDs/status originais e são enumerados didaticamente no índice. Tickets conservam IDs; relatos conservam datas/revisões. Não renomear snapshots, PDFs, evidências, logs ou patches para impor uma ordem artificial. Originais precisam permanecer íntegros.
5. Há duas pontes de política com nomes legados para a regra .agents somente leitura. Não são capítulos nem fontes normativas: apontam para as políticas numeradas. Não criar outras exceções sem documentar a necessidade no mapa de ordem.
6. Novo capítulo exige atualizar ORDEM-LEITURA.json, introdução da pasta, índice, vizinhos anterior/próximo, links dos consumidores e guias/prompt quando afetados. Renumeração exige mapa antigo/novo e snapshot/hash; manter IDs de decisões/tickets e evidências.
7. O validador deve exigir prefixos/posições únicos, cobertura do mapa, destinos e navegação. Aprovação semântica também confere pré-requisitos: um check estrutural sozinho não prova que a ordem ensina corretamente.

O [mapa de leitura](ORDEM-LEITURA.json) registra sequência por pasta e realocação íntegra de DOC-MV-003.
