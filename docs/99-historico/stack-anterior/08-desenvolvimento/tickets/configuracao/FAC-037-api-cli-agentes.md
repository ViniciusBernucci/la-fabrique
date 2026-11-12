# FAC-037 — Cadastro por API ou assinatura CLI e seleção de IA/modelo

Data: 2026-10-09, Europe/Berlin. READY pelo pedido explícito do responsável nesta sessão. Base: 36e48f91a29e808ebd625bf11a4e24264c78a867; branch feat/fac-037-api-cli-agents, worktree isolada. O pedido amplia a configuração antes exclusiva de assinaturas para aceitar também chaves de API. Não autoriza chamadas pagas durante os checks.

## Objetivo e critérios

1. Adicionar conta e cadastrar integração abre primeiro a escolha Cadastrar chave de API / Cadastrar plano de assinatura (CLI); cancelar não cria conta.
2. API permite cadastro OpenAI/Claude com chave privada, preservando login CLI existente. Chaves não aparecem nas respostas, configuração, snapshot do worker ou logs.
3. Pré-configurados e personalizados escolhem uma IA cadastrada e depois um modelo daquela IA; apenas integrações habilitadas são atribuíveis. Trocar IA redefine modelo.
4. Persistência versionada salva configuração e chave criptografada atomicamente. Remover integração remove seu segredo. Configurações CLI antigas permanecem válidas.
5. Nenhuma conta API pode ser enviada ao login/verificação CLI nem executada pelo adapter CLI.

## Escopo e limites

apps/web, packages/contracts, apps/api/settings e schema/migration de segredo, gate de roteamento em apps/worker; capítulos e evidências afetados. Sem mudanças de stack, credenciais reais, sources, produção, gastos, push/merge/deploy. API de inferência/execução agentiva permanece fora deste incremento de cadastro: o runtime disponível é CLI, sem adapter API. A UI informa esse limite.

Risco: médio (persistência de segredo e compatibilidade). Chave AES de implantação separada via PROVIDER_API_KEY_ENCRYPTION_KEY; não há chave padrão. Migration preparada, sem aplicação automática. Checks: contracts/API/worker/web, typecheck, lint, build e gate documental. Autorrevisão não substitui revisão independente; aceite humano exato pendente.
