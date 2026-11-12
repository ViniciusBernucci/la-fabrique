> Leitura: [← Anterior](04-contexto-deterministico-limites.md) · [Índice didático](../02-INDEX.md) · [Próximo →](06-sandbox-worktree-snapshot.md)

# Checkout confiável mantém credenciais fora do projeto

Um checkout de repositório externo precisa acontecer no supervisor confiável, antes de entregar a árvore ao sandbox do agente. O destino deriva de UUIDs validados sob um root privado, e a referência é um SHA exato — nunca um caminho ou branch fornecido pelo prompt. Hosts permitidos vêm de configuração operacional exata; URL não pode carregar credenciais.

No FAC-012J, o worker comprova `tokenSource=keyring` pelo `gh auth status` antes de obter um token efêmero do cliente oficial. A autenticação Git fica num cabeçalho host-scoped somente no ambiente do processo Git; não aparece em argv, remote URL, stdout/stderr, job nem API. O processo Git ignora configurações globais/sistêmicas, hooks e protocolos locais. Clone sem checkout seguido de fetch e checkout detached reduz efeitos do repositório antes da revisão. Falha/timeout remove apenas o diretório temporário criado e nunca declara checkout concluído por exit code isolado: o SHA resolvido precisa ser igual ao snapshot.

FAC-012L conecta o preparador ao consumer gated e reconfere projeto/URL/SHA retornados antes de compilar o workflow. Checkout e execução usam raízes separadas, e o ID do checkout é o da tentativa com fencing token. Ainda não houve Git/GH/keyring reais ou identidade systemd; testes sintéticos não autorizam ativação operacional.

Referências: [relatório FAC-012J](../09-entregas/2026/infraestrutura/2026-10-02-FAC-012J-checkout-confiavel-worker.md), `apps/worker/src/repository-checkout.ts`.
