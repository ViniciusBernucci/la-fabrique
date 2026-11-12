# ARQ-MEM-001 — Planos e explicação da arquitetura de memória e decisões

Data: 2026-10-09, Europe/Berlin. Base de código: `896ef0d2c804b80711d5ad00ad96695eaaf9f2d7`; branch `feat-arquitetura-contexto`. Status documental: DOCUMENTADO / AWAITING_HUMAN. Implementação das novas capacidades: PLANEJADA, não realizada nesta entrega. ARQ-MEM-001 identifica este registro documental; não é novo ticket FAC ou backlog de implementação.

## Objetivo, problema e escopo

O responsável pediu leitura da transcrição, oito planos arquiteturais separados e documentação didática para compreender as decisões, sem requisitos inferidos. Foram realizadas rodadas de esclarecimento e preservadas pendências de configuração/avaliação. Modelo local, JEV e treino possuem planos futuros; próxima versão reúne capacidades de memória, busca, grafo, contexto/expansão, atualização/conflitos e painel. Backlog detalhado fica para o Tech Lead.

Fonte externa: `/home/vinicius/le-fabrique/transcricao-integral-memoria-contexto-persistente.md`, 2247 linhas, SHA-256 `9badc77b393c45137bc403c5254fe5bacc4385b8e2e29b8c3030793f1ac2579b`. Conversas não serão ingeridas na memória operacional nesta etapa. A fonte foi lida para planejamento e não alterada.

## Solução documental e funcionamento proposto

[Guia didático](../../02-arquitetura/memoria-e-decisoes/00-README.md) oferece visão integrada, vocabulário e oito capítulos por feature, com exemplos e fluxo. [Planos](../../08-desenvolvimento/planos-memoria-e-decisoes/00-README.md) oferecem contratos, alternativas, dependências, riscos, compatibilidade, recuperação e critérios de aceite. [Matriz](../../08-desenvolvimento/planos-memoria-e-decisoes/10-matriz-fluxo.md) conserva as 145 operações, 56 exceções e 20 decisões candidatas JEV. Não atribui verificação individual a software ainda não inspecionado por item.

[Decisões/evidências](../../08-desenvolvimento/planos-memoria-e-decisoes/01-decisoes-e-dependencias.md) separa 20 confirmações do responsável e propostas técnicas. Confirmações incluem main como referência, publicação no aceite, incorporação separada, sobreposições por branch, atualização automática de índices, regra nova sujeita a aceite e validação com auxílio de IA. Os oito planos estão completos como propostas revisáveis; valores operacionais e escolhas técnicas pendentes não foram inventados.

## Arquivos, contratos e decisões

22 capítulos novos: dez guias (introdução, visão integrada e oito features), onze capítulos de planejamento (introdução, decisões, oito planos e matriz), um contrato comum. Dois ADRs novos: [ADR-007](../../06-decisoes/ADR-007-memoria-e-decisoes-versionadas.md) propõe estrutura; [ADR-008](../../06-decisoes/ADR-008-excecao-jev.md) registra exceção JEV confirmada e desenho ainda proposto. Catálogos, navegação, mapa didático, Manual Vivo, Context Builder, handoff, backlog, matriz de cobertura, índice e changelog atualizados.

Nenhuma mudança em código, dependências, schema Prisma, API executável, provider ou infraestrutura. Contratos novos são semântica documental proposta, não software. Política IA recebe exceção específica autorizada para JEV, sem liberar APIs de LLM/extras/fallback pago. Orçamento/limites pelo sistema; integração posterior.

## Evidência do estado observado e limites

Context Builder atual lê fontes indicadas, registra hashes/omissões e não faz indexação/grafo. Workflow atual recompõe contexto em recovery/handoff. main local aponta para bootstrap `15c2198076a6a6afcbf2f15212ae1eaf53822b9f`; precisa ser conferida antes da implementação. Perfil atual permite teto 6 GiB/CPU quota 400%, distinto da VPS declarada de 2 vCPUs/8 GB; benchmark e admissão são planejamento, não medição realizada.

Nenhuma revisão independente ou QA funcional das novas capacidades foi executada. A sessão realizou leitura e autorrevisão documental; não simula aprovação independente. Autoload de políticas/skills permanece NÃO VERIFICADO. Paths/hashes de carregamento explícito estão no registro de decisões.

## Checks e evidências

Baseline: `python3 scripts/validar-documentacao.py` PASS na base, 385 Markdown, 2357 links, 601 snapshots, zero erros. A primeira conferência intermediária após criação dos planos apontou somente link para este relato ainda não criado; ele foi criado antes da validação final.

Resultados finais e teste de fidelidade dos 221 itens constam em [evidências](evidencias/ARQ-MEM-001/verificacao.json). `git diff --check` verifica whitespace; check estrutural verifica links/mapa/fences/hashes, sem provar verdade semântica ou eficácia de software. Testes lint/typecheck/build/unit/integration: NOT_RUN, pois apenas documentação/política foi alterada e scripts de aplicação não foram modificados. Benchmark host e chamada JEV real: NOT_RUN. Renderização de Mermaid: NOT_RUN; fences verificados estruturalmente.

## Diff, segurança e rollback

Workspace inicialmente limpo; somente documentos e mapa de leitura modificados/criados. Lista/hashes finais em [manifesto](evidencias/ARQ-MEM-001/manifesto.json). Não houve commit, merge, push, deploy, leitura de credenciais ou chamada autenticada JEV. Consultas públicas aos documentos oficiais TypeSafe/pgvector sustentam referências técnicas; uma tentativa de leitura da model card Hugging Face falhou, por isso modelo/artefato local permanece candidato histórico não verificado.

Rollback documental: revisar diff desta entrega, retirar somente capítulos/ADRs/seções aqui adicionados e grupos novos do mapa, preservando trabalho posterior e requisitos já confirmados. Reverter publicação de política requer decisão explícita; não apagar evidência da autorização. Sem DB/serviço migrado, não há rollback de dados ou infraestrutura a executar.

## Uso de IA, custos e lessons

Papel: arquiteto via skill local adaptada `skills/arquiteto-de-software/SKILL.md`; sem subagentes. Provider da sessão: OpenAI/Codex; modelo/versão efetivos, billing/uso/custo incremental não observados por ferramenta nesta entrega, permanecem desconhecidos. Nenhuma API paga JEV ou de LLM foi habilitada. Lessons novas: N/A; propostas não foram implementadas e não justificam lição aplicada.

## Pendências e aceite

Responsável avalia propostas técnicas e a revisão documental exata; Tech Lead deriva tickets depois. Retenção, formatos de metadados, override durante execução, parâmetros JEV pelo sistema e benchmark são pendências explícitas de configuração/avaliação. Revisão independente e aceite documental exato pendentes. Nenhum ticket existente mudou para DONE.

## Adendo 2026-10-24 — integração

Trabalho da worktree commitado em `826adb5` e integrado à developer local no merge `05a9877`; conflitos só em índices, changelog, backlog, matriz e handoff, com as entradas dos dois lados mantidas. A developer já continha [ADR-007 da conta do usuário](../../06-decisoes/ADR-007-conta-usuario-e-sessoes.md) (FAC-038); o [ADR-007 desta entrega](../../06-decisoes/ADR-007-memoria-e-decisoes-versionadas.md) mantém número e arquivo para não invalidar o [manifesto](evidencias/ARQ-MEM-001/manifesto.json). Numeração pendente de decisão do responsável. Nenhuma pendência técnica desta entrega depende de execução: avaliação das propostas e aceite continuam com o responsável.
