# FAC-035 — Nome e cargo dos agentes

2026-10-08 (Europe/Berlin). READY pelo pedido explícito do responsável nesta sessão.

Objetivo: permitir apelidar agentes em Configurações → Equipes, preservando a identificação anterior como cargo. Abrange pré-configurados e personalizados.

Base: 97869109e9d8c74bed5635b0cb3a0c6eaaca6fe9; branch feat/fac-035-agent-names, worktree /home/vinicius/le-fabrique-fac-035, criada limpa e exclusiva desta sessão. Outras sessões no workspace original têm quiescência desconhecida; nenhuma escrita de código ou integração será feita ali.

Escopo: contratos compartilhados, painel de configurações/equipes, testes pertinentes e documentação afetada. Persistir apelido opcional no JSON versionado existente, sem migration. Cargo predefinido mantém role; cargo personalizado mantém name legado. Não alterar roteamento, permissões, ativação, providers ou despesas.

Critérios: nome editável com limite de 100 caracteres; listagem e modal apresentam nome e cargo separadamente; registros antigos continuam válidos; limpar nome retorna à identificação pelo cargo; salvar/cancelar mantêm comportamento versionado e rascunho; checks pertinentes passam. Duas correções no máximo. Sem subagentes, commit, merge, push ou deploy autorizados; revisão independente e aceite exato pendentes.

Fechamento em 2026-10-08: IMPLEMENTADO / AWAITING_HUMAN; código bc38715faffa5c1f3cdad82dca43822508fd0b7b. Pedido explícito posterior autoriza commit e integração local à developer. Revisão independente/browser/aceite exato e deploy continuam pendentes.

Adendo 2026-10-09: usuário relatou falha ao salvar pelo frontend. Reproduzida rejeição de nickname em dist antigo; hook predev prepara contratos/runtime antes dos serviços. [Correção e limites](../../../09-entregas/2026/2026-10-09-FAC-035-correcao-salvamento-nomes.md). Sessão real/browser e aceite pendentes.
