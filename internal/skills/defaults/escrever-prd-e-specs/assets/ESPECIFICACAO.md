# Especificação técnica — <demanda>

- **PRD de origem:** [PRD](PRD.md) · <versão>
- **Versão e estado:** <versão> · <RASCUNHO ou PRONTO_PARA_PLANEJAMENTO>
- **Base consultada:** <repositórios, refs ou SHAs, documentos e contratos>

## Contexto técnico e decisões

<Descreva o estado atual comprovado, o desenho proposto e os limites entre módulos, processos e sistemas. Registre decisões técnicas e trade-offs, com fonte e vínculo a requisitos. Use um diagrama quando houver interação entre serviços ou estados difíceis de entender.>

## Repositórios e estrutura de pastas

| Repositório e base | Componente ou ponto de entrada | Caminho relativo | Ação | Evidência ou justificativa |
|---|---|---|---|---|
| <repo, ref e SHA quando confirmado> | <módulo, rota, serviço, job ou teste> | <caminho> | <reutilizar, alterar ou criar> | <caminho confirmado ou proposto, com fonte> |

<Inclua áreas de código, testes, configuração, contratos, documentação e migrações afetadas. Identifique o proprietário de cada contrato e os caminhos ou recursos em disputa entre áreas. Para um repositório novo, justifique fronteiras, estrutura e dependências. Registre caminhos essenciais que ainda não puderam ser determinados.>

## Comportamento técnico por fluxo

<Para cada fluxo relevante, trace ação → ponto de entrada → validação e autorização → regra de negócio → leitura ou escrita → efeito observável. Inclua transações, erros, efeitos parciais, concorrência, reprocessamento e critérios quando aplicáveis. Explicite estados persistidos e transições.>

## Contratos de API, eventos e integrações

| ID | Interface e mudança | Proprietário e consumidores | Fonte canônica | Compatibilidade |
|---|---|---|---|---|
| CT-01 | <método e rota, evento ou outra interface> | <produtor e consumidores> | <schema, OpenAPI, código ou seção desta spec> | <existente, aditiva, migração ou quebra> |

<Para cada contrato alterado, documente autenticação e autorização por operação, parâmetros, headers, schemas de entrada e saída, tipo e obrigatoriedade por campo, defaults, nulabilidade, formatos e restrições aplicáveis. Inclua exemplos de sucesso e de erros de domínio ou validação, códigos e envelope de erro. Cubra paginação, filtros, ordenação, concorrência ou idempotência onde fizerem diferença. Para eventos e webhooks, inclua gatilho, payload, identidade, versão, entrega, ordenação, retries, duplicidade, reconciliação e segurança conforme aplicável.>

<Quando existir schema canônico, vincule sua revisão e detalhe o delta. Se for necessário criar uma interface HTTP, forneça contrato estruturado compatível com o padrão do projeto, como um trecho OpenAPI quando adotado. Sinalize mudanças incompatíveis e descreva a transição e a validação dos consumidores. Use exemplos sintéticos.>

### <CT-xx> — <operação ou evento>

- **Interface e versão:** <método e rota ou nome do evento, operação e versões afetadas>
- **Autorização:** <identidade, papel, escopo do recurso e resultado quando o acesso for negado>
- **Entrada e saída:** <referência ao schema e seu delta, ou definição estruturada dos campos e formatos>
- **Comportamento:** <pré-condições, efeitos, repetição, concorrência, limites e garantias aplicáveis>
- **Respostas e falhas:** <código ou tipo, condição exata, payload e tratamento do consumidor>
- **Compatibilidade e testes:** <ordem de mudança dos consumidores e testes de contrato associados>

## Dados, segurança e operação

- **Dados e persistência:** <entidades, campos, vínculos, invariantes, constraints, índices, transações, dados antigos e migração ou backfill>
- **Segurança e privacidade:** <matriz ator × ação × recurso, isolamento, validação, dados sensíveis, referências de credenciais e trilha de auditoria>
- **Operação:** <configuração e ambientes, observabilidade, limites, alertas, rollout por componente, recuperação, rollback ou compensação>
- **Integrações:** <dependências, donos, indisponibilidade, versão e ordem de atualização quando aplicável>

## Estratégia de verificação

| ID | Critérios | Nível e interface | Cenário e dados | Resultado esperado | Como executar e comprovar |
|---|---|---|---|---|---|
| TEST-01 | CA-01 | <unitário, integração, contrato ou E2E> | <pré-condição, fixture e ação> | <resultado observável> | <comando existente confirmado ou procedimento proposto, ambiente e evidência> |

<Inclua regressões, permissões e falhas aplicáveis. Para mudanças de interface, indique jornadas, pontos de entrada, papéis e dados de teste. Para serviços, dados ou infraestrutura, especifique as verificações aplicáveis e os riscos que elas devem cobrir. Preserve a distinção entre verificação planejada e executada.>

## Tarefas técnicas propostas

Repita o bloco para cada unidade executável. Use referências locais estáveis, como `T-01` e `T-02`.

### <ID> — <resultado verificável>

- **Área e local:** <área responsável, repositório, módulo, pontos de entrada e caminhos a alterar, criar ou testar>
- **Objetivo e escopo:** <resultado e mudança proposta, incluindo limites>
- **Entradas, saídas e regras:** <requisitos, estados, erros e efeitos que a tarefa precisa respeitar>
- **Contratos e dados:** <CT-xx, campos, compatibilidade, migrações e artefatos afetados>
- **Dependências:** <ID, entrega necessária e etapa afetada, como implementação, revisão, integração ou implantação>
- **Recursos compartilhados:** <contrato, arquivo, esquema ou ambiente, tipo de acesso e restrição de concorrência>
- **Risco e validação:** <risco proposto e justificativa, RF-xx ou RNF-xx, CA-xx, TEST-xx, checks aplicáveis e evidências esperadas>
- **Entregáveis:** <código, testes, documentação, configuração, migração ou artefato exigido>
- **Pendências:** <decisão, impacto e condição de liberação quando aplicável>

## Ordem, dependências e rastreabilidade

<Mostre a ordem de implementação, os pré-requisitos de revisão, o que pode correr em paralelo e a sequência de integração ou implantação. Confira ciclos e condições de integração entre equipes ou repositórios.>

| Requisito ou regra | Critérios | Contratos ou componentes | Tarefas | Testes |
|---|---|---|---|---|
| RF-01 | CA-01 | CT-01 | <ID> | TEST-01 |

## Pendências e handoff

<Informe o que foi verificado, o que continua proposto e quais lacunas bloqueiam a entrega documental ou uma tarefa. Para cada lacuna, indique evidência consultada, ação necessária e alcance do bloqueio. Registre a revisão do pacote entregue para planejamento.>
