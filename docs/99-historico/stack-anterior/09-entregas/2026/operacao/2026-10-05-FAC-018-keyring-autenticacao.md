# FAC-018 — Testar autenticação

Código 0182a4fa02dfaeb008cad2079084b251a317b12b; AWAITING_HUMAN. Setup dos binários concluído no .env ignorado; WORKER_ANTIGRAVITY_BINARY aponta para agy instalado. Keyring ainda ausente. Nenhuma conta pessoal foi copiada ou autorizada pelo agente.

1. Na VPS, instalar dependência: `sudo apt-get install -y gnome-keyring`. Sudo exige sua senha no terminal; não compartilhar senha no painel/chat. dbus-run-session e dbus-send já existem.
2. Na raiz, `npm run build -w @le-fabrique/worker` prepara launcher compilado. O build final já foi executado nesta revisão; repetir se alterar/atualizar código.
3. Parar o npm run dev anterior e iniciar `npm run dev` novamente para carregar o ambiente novo. Manter túnel SSH que já funcionou e abrir http://localhost:5173.
4. Configurações → Contas → Claude/Antigravity. Habilitar e salvar; verificar conta e modelos para obter estado observado. Em AUTH_REQUIRED, clicar Conectar assinatura Claude ou Conectar conta Google. Aguardar URL oficial; se popup bloqueado, usar link no modal.
5. Autorizar no navegador. Quando o provedor apresentar código de retorno, colar no campo do modal e enviar uma vez. Código é temporário e não deve ir a logs/chat. Aguardar confirmação do cliente; depois conferir modelos e atribuir em Equipes.

Se sessão expirar, iniciar nova conexão. Login não ativa execução da fábrica; Antigravity permanece sem adapter e catálogo Claude manual. Keyring ausente aparece como erro explicado. Prova atual: desafio Claude/Google e fixtures de fluxo, sem consentimento real nem teste do keyring ausente. Não executar migrations nem ativar extra usage/credits/API para testar login. Rollback no [relatório](../configuracao/2026-10-05-FAC-018-login-claude-antigravity.md).
