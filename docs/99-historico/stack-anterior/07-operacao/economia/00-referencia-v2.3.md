> Leitura: [Índice didático](../../02-INDEX.md) · [Próximo assunto →](../handoff/00-referencia-v2.3.md)

> Instruções de criação em documentacoes/ ou lessons/ abaixo são histórico da fonte; novas entregas/lessons seguem exclusivamente docs/00-governanca/POLITICA-DOCUMENTACAO.md.

> Nota editorial DOC-MV-001, 2026-10-07: conteúdo integral local preservado como referência da evolução v2.3. Afirmações por incremento têm a revisão/data original; trechos sobre consumer ausente, recuperação pendente e caminhos antigos não descrevem necessariamente HEAD a2cc5e0. O [AS-IS conferido](../../02-arquitetura/01-as-is.md) e a [auditoria](../../00-governanca/08-AUDITORIA.md) delimitam o estado atual; números de testes abaixo são evidência histórica, não reexecução desta migração.

# Política de economia

## Decisão obrigatória da stack - revisão 2.3
A stack da própria La fabrique está APROVADA: React + TypeScript + Vite no painel; NestJS + TypeScript na API; worker Node.js + TypeScript em processo separado; PostgreSQL; Redis + BullMQ; Docker Compose na mesma VPS. Não solicitar nova escolha ou confirmação da stack. Não iniciar a fábrica em PHP/Laravel, Angular ou .NET. Esta decisão substitui propostas anteriores.
Monorepo: `apps/web`, `apps/api`, `apps/worker`, `packages/contracts`. Contratos compartilhados precisam de validação em runtime. API não executa clientes, builds ou testes; o worker executa esses trabalhos com isolamento, limites e um writer inicial.
Ao trabalhar na própria fábrica, aplicar esta stack. A regra de preservar a stack existente aplica-se somente a projetos EXTERNOS cadastrados para desenvolvimento pela fábrica; ela não altera a stack da La fabrique. Se o repositório da fábrica contiver implementação anterior incompatível, registrar a divergência e planejar a adaptação por etapas; não apagar código existente nem reabrir a escolha tecnológica.
Versões exatas e comandos devem ser fixados conforme compatibilidade no bootstrap; isso não é uma nova decisão de stack. Repositório e funcionalidade do piloto externo permanecem pendentes quando não fornecidos.

A prioridade é economizar capacidade e evitar retrabalho. Assinaturas substituem cobrança por chamada na arquitetura principal, dentro da elegibilidade e limites reais de cada plano.
1. Ticket pequeno, critérios claros e baseline antes da IA.
2. rg/imports/testes para selecionar fontes; excluir vendor/node_modules/dumps/segredos.
3. Apenas Developer e Reviewer no fluxo comum; QA automatizado e documentação integrada. Papéis adicionais por necessidade.
4. Sessão independente para review, contexto resumido mas com fontes acessíveis.
5. Limitar correções, timeout, handoffs e concorrência; pausar se diagnóstico repetido.
6. Escolher modelo exposto/elegível adequado e medir qualidade. Contextos curtos ajudam a cota, mas limites não são conversão fixa tokens/tickets.
7. Reusar caches de dependências e outputs determinísticos por revisão. Checks antigos não validam diff novo.
8. Handoff objetivo com artefatos; não colar conversas inteiras entre providers.
9. API, extra usage e créditos automáticos desligados. Aguardar cota se todos indisponíveis.
10. Revisar resultados após dez tickets; custo atribuído, incremental, espera por cota e retrabalho separados.
Não prometer gasto total R$0: assinaturas, VPS e energia têm custo. Não contratar plano adicional antes de medir gargalo. Preços não foram fixados no kit; preencher com cobrança real e data na configuração financeira.

O FAC-006 implementa os limites locais: no maximo duas tentativas, 30 minutos de janela configurada, duas trocas de provider e pausa na segunda falha consecutiva identica. A politica validada exige assinatura, budget de API zero, fallback e extras desligados. A conta do fornecedor continua exigindo verificacao humana; configuracao local nao altera cobranca externa.

No FAC-010, os clientes candidatos estavam deslogados. A documentacao oficial diferencia Claude App Pro/Max de Anthropic Console com billing de API; somente a primeira modalidade pode ser avaliada neste MVP. Nao houve login automatico, chave API, chamada, compra ou mudanca de configuracao financeira. O responsavel precisa confirmar plano e extras antes do preflight funcional.

O FAC-011 torna visiveis no painel as travas `apiEnabled=false`, `extraUsageEnabled=false`, `paidCreditsEnabled=false`, `autoRechargeEnabled=false` e `paidFallbackEnabled=false`. Elas sao literais validados e nao editaveis, portanto um payload administrativo nao pode liga-las. Isso protege a fabrica, mas nao altera a conta externa do fornecedor: onboarding/preflight ainda deve confirmar que as mesmas opcoes estao desligadas no site e no cliente oficial.

FAC-010B usa exclusivamente autenticacao ChatGPT do cliente Codex por device code e remove chaves de API herdadas do ambiente do processo. O fluxo nao executa prompt nem consome modelo, mas tampouco comprova configuracoes financeiras externas: extra usage, creditos e recarga automatica continuam exigindo confirmacao humana na conta antes da operacao real.

FAC-010C bloqueia no processo Claude chaves/tokens, base URL, AWS, Bedrock, Vertex e Foundry herdados. Somente status first-party com metodo explicitamente de assinatura pode ficar `AVAILABLE`; Console/API e auth ambigua viram `ERROR`. O handoff respeita limite de trocas e retorna `WAITING_PROVIDER` em auth/cota, sem ativar fallback pago. Nenhuma chamada real foi feita, portanto custo/uso continuam desconhecidos.
