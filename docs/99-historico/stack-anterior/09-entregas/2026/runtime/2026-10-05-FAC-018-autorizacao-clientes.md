# FAC-018 — Autorização dos clientes oficiais

Revisão 0182a4fa02dfaeb008cad2079084b251a317b12b; IMPLEMENTADO/AWAITING_HUMAN. Worker separado executa comandos fixos; API nunca executa cliente. node-pty 1.1.0 é necessário porque agy exige TTY e consultas de terminal; respostas de cor/cursor são fixas, somente Google OAuth é escolhido, nenhum prompt de inferência é enviado. Claude usa pipes e `--claudeai` para assinatura, sem Console/API.

ProviderIdentityManager mantém HOME/store/cache por conta e ambiente allowlisted sem chaves API/tokens do controle. Antigravity recebe D-Bus novo e keyring nativo privado pelo launcher compilado. Login tem limite de dez minutos/256 KiB, mata grupo de processos no timeout/shutdown, publica URL com host vinculado ao provider e encaminha uma resposta ao stdin somente quando solicitado. Inspeção posterior separada confirma AVAILABLE; catálogo vazio do Antigravity falha fechado. Não declara autenticação apenas pelo popup ou exit code de login.

557 testes/checagens e limites estão no [relatório](../configuracao/2026-10-05-FAC-018-login-claude-antigravity.md). Desafio Claude comprovado com runner real e expiração; Antigravity CLI real comprovou URL Google, transporte PTY automatizado usa fixture. Keyring nativo real e consentimento de ambas as contas pendentes. Rollback: reverter revisão, recompilar/reiniciar, preservar stores privados.
