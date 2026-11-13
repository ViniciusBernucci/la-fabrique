# FAC-018 — Keyring privado Antigravity

Código 0182a4fa02dfaeb008cad2079084b251a317b12b; IMPLEMENTADO, serviço nativo NÃO VERIFICADO nesta VPS por falta de gnome-keyring-daemon. Requer Linux, dbus-run-session/dbus-send/gnome-keyring-daemon e launcher worker compilado. worker.env.example inclui binário agy; setup resolve binário local.

ProviderIdentityManager cria diretórios 0700 fora do repositório, arquivo keyring-unlock aleatório 0600, valida dono/arquivo regular/sem symlink. Não lê credenciais pessoais existentes. dbus-run-session fornece barramento novo por processo; daemon recebe senha de desbloqueio somente por stdin. HOME/XDG_CONFIG_HOME usam conta privada, control-directory é temporário por processo, pronto significa nome org.freedesktop.secrets disponível. Client herda terminal; launcher encerra daemon e remove somente seu control-directory. Keyring e chave privada permanecem para próximos logins. D-Bus global não é reaproveitado.

Instalação de pacote requer usuário com sudo; pedido já enviado, sem senha compartilhada. Criptografia/persistência real do keyring, login e modelo de conta real dependem dessa instalação e consentimento humano. Não afirmar disponíveis antes da inspeção. Testes de transporte e contratos passaram; detalhes no [relatório](../configuracao/2026-10-05-FAC-018-login-claude-antigravity.md). Rollback preserva dados privados.
