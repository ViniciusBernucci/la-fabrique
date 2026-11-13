> Leitura: [← Anterior](05-fontes-carregamento.md) · [Índice didático](../02-INDEX.md) · [Próximo →](07-PENDENCIAS.md)

# Fontes e evidências
Consulta inicial: 29/09/2026; autenticacao e CLI Claude Code reconferidas em 01/10/2026. Arquitetura, limites, preferência de papéis e metas são propostas de engenharia. Nenhum prompt de segundo provider foi executado no FAC-010.
## OpenAI
- https://learn.chatgpt.com/docs/auth — login Codex com ChatGPT, `codex login status`, device auth para headless e protecao do cache de credenciais; consultado novamente em 01/10/2026 para FAC-010A.
- https://developers.openai.com/codex/noninteractive — exec, JSONL, sandbox e reutilização de autenticação CLI; endereço redireciona para https://learn.chatgpt.com/docs/non-interactive-mode.
- https://github.com/openai/codex — cliente oficial local e login ChatGPT. Validar plano/modelos/limites reais no onboarding; não derivar capacidade pelo número de tickets.
## Anthropic
- https://code.claude.com/docs/en/headless — -p não interativo, saída estruturada e descoberta de regras; --bare ignora CLAUDE.md.
- https://code.claude.com/docs/en/legal-and-compliance — autenticação oficial, credenciais e distinção entre uso próprio do binário e serviços para usuários. Não transformar OAuth de assinatura em chave de SDK nem intermediar credenciais.
- https://code.claude.com/docs/en/costs — acompanhamento de uso e distinção entre estimativa de tokens e faturamento de assinantes. Condições podem mudar; verificar cobrança do modo automatizado contratado.
- https://docs.anthropic.com/en/docs/claude-code/getting-started — Claude App Pro/Max inclui Claude Code; Anthropic Console exige billing proprio. O MVP aceita somente a modalidade de assinatura confirmada no preflight.
- https://docs.anthropic.com/en/docs/claude-code/cli-usage — `-p`, JSON/stream-json e controles de permissao para automacao.
## Google
- https://codelabs.developers.google.com/antigravity-cli-hands-on — instalação/login, agy -p, modelos e controles de permissões/créditos.
- https://codelabs.developers.google.com/agentic-ui-automation-with-antigravity — CLI e teste de UI; presença de capacidade não prova cobertura de todos os planos/modelos.
## Orca
- https://www.onorca.dev/docs/settings — referencia publica das secoes de configuracao de agentes, contas e integracoes usada como inspiracao funcional no FAC-011.
- https://www.onorca.dev/docs/agents/supported — catalogo/deteccao de clientes e customizacao por agente. A La fabrique nao adotou flags de bypass amplo descritas nessa pagina.
- https://github.com/stablyai/orca — implementacao publica MIT consultada para confirmar o escopo do produto; nenhum codigo ou asset foi copiado.
## Limites da evidência
Não usamos preços fixos ou percentuais de quota. O tutorial técnico não comprova elegibilidade ilimitada, SLA 24/7, quota consultável programaticamente ou reset exato. Essas capacidades precisam de preflight com evidência e versão.
Um resultado de busca sugeriu mudanças de crédito headless do Claude; a página aberta não confirmou essa afirmação. Ela não foi adotada como fato. Validar modo de cobrança real antes de habilitar cada instalação, especialmente SDK/headless.
APIs, caching e Batch do kit anterior continuam extensões possíveis, mas fora da operação MVP. Não somar descontos nem transferir condições de API para assinatura.
