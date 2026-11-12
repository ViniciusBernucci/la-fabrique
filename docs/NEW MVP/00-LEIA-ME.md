# Le Fabrique — especificação do novo MVP

**Estado da entrega documental:** especificado, com implementação e aceite operacional pendentes.  
**Escopo de lançamento:** MVP completo na primeira entrega (D42).  
**Responsável pelo desenvolvimento e pelo aceite:** o usuário.

## Documentos

| Documento | Uso |
|---|---|
| [Arquitetura e contratos](01-ARQUITETURA.md) | Estrutura do repositório, ambientes, serviços, dados, estados e protocolos entre componentes |
| [Épicos detalhados](02-EPICOS.md) | Escopo, requisitos, regras, dependências, entregáveis, critérios de aceite e testes |
| [Backlog de implementação](03-BACKLOG.md) | Features e tickets com ordem técnica, dependências e critérios executáveis |
| [Matriz das decisões D01–D95](04-RASTREABILIDADE.md) | Decisão, vigência e vínculo com épico e ticket |
| [Aceite integral e piloto real](05-ACEITE-E-PILOTO.md) | Preparação, roteiros, evidências e condição de lançamento |
| [Fontes e precedência](06-FONTES.md) | Referência à entrevista e inventário dos 14 arquivos históricos |

## Produto

A Le Fabrique é um painel pessoal para gerir a execução de desenvolvimento assistido por IA em projetos de software. Opera continuamente em uma VPS existente. Em cada projeto, Jira ou ClickUp fornece as features marcadas, os requisitos e as prioridades. A fábrica decompõe o trabalho, agenda tarefas respeitando dependências, executa workflows configurados pelo administrador e produz código, testes, documentação e uma Draft PR no GitHub. O merge cabe a uma pessoa. O estado detalhado da execução, as decisões, a memória técnica e a recuperação pertencem à fábrica.

O cadastro orienta o administrador a validar conexões e escolher recursos manualmente. O produto oferece editor visual com workflows publicados e subfluxos, catálogos de agentes, skills, ferramentas e servidores MCP, conexões CLI e API configuráveis por agente, fallback autorizado, controles de orçamento, isolamento, aprovações e recuperação. Um projeto só inicia uma tarefa técnica por vez. Projetos distintos podem concorrer, conforme o limite global.

## Condições de vigência

- **D42 e D95.** A primeira entrega inclui todos os épicos, integrados e validados com evidências. Fundações, ordem de trabalho e testes intermediários organizam a implementação. O aceite exige projeto real cadastrado do zero, feature ponta a ponta e cenários de recuperação, aprovações, isolamento e sincronização.
- **D32 → D55–57.** A trava única global foi substituída por uma tarefa técnica por projeto, limite global de projetos ativos e prioridade com envelhecimento da fila. Referências anteriores a uma única feature global cedem a essa regra.
- **D64 → D65.** A revisão inicial é independente da implementação. Depois da avaliação, o mesmo revisor pode corrigir em etapa própria, seguida de revalidação real.
- **D89.** A descoberta de mudanças no Jira, ClickUp e GitHub acontece em sincronização manual ou na consulta autorizada que inicia uma execução, inclusive retomada. Ao detectar uma mudança por esses meios, o tratamento pode ser automático. Escritas de andamento, consultas para reconciliar uma operação conhecida e retentativas de disponibilidade de IA obedecem a contratos próprios.
- **D74, D75 e D94.** Conta escolhida explicitamente por agente ou nó, modelo cadastrado manualmente e workflows/agentes associados manualmente por projeto. Catálogo global e fallback previamente configurado continuam disponíveis.
- **Limites de escopo.** Uso pessoal com um administrador. GitHub é o único provedor Git integrado. A Central de Decisões fica no painel. Webhooks de descoberta, polling periódico de mudanças externas, catálogo automático de modelos, rodízio de contas, templates iniciais de projeto e merge automático ficam fora do lançamento.

## Convenção de leitura

`Dxx` remete à entrevista. `Exx` identifica épico e `Txx-yy` identifica ticket. Todos os épicos e tickets deste pacote compõem o lançamento. **Fundação**, **produto**, **integração** e **aceite** indicam sequência técnica. Escolhas marcadas como **proposta de engenharia** concretizam decisões delegadas ou parâmetros em aberto. Registre alterações dessas escolhas em ADR, com teste equivalente e preservação das regras da entrevista. O responsável define a ordem efetiva do desenvolvimento.

A identificação do repositório da fábrica, a VPS, o projeto piloto, os escopos reais e as conexões serão preenchidos durante o desenvolvimento. Este pacote é composto por documentos Markdown para copiar para o repositório ou para o Obsidian, mantendo os arquivos juntos e os links relativos.
