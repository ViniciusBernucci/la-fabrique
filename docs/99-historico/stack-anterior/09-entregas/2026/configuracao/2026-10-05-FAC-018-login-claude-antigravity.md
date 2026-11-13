# FAC-018 — Login oficial Claude e Antigravity

IMPLEMENTADO / AWAITING_HUMAN. Developer autorizado pelo usuário; baseline d6802fe; revisão 0182a4fa02dfaeb008cad2079084b251a317b12b. Sem inferência, migrations ou adapter Antigravity. Aceite de conta real e revisão humana pendentes.

## Funcionamento e contratos

SettingsPanel abre uma aba no clique e encaminha somente URLs HTTPS do provedor correspondente. Claude usa `auth login --claudeai`, Google/Antigravity usa o menu Google OAuth do cliente oficial em PTY. O modal aceita o código devolvido pelo navegador quando solicitado, mostra andamento/erros e confirma sucesso somente depois de nova inspeção do cliente. Codex mantém device flow.

Sessão e outbox contêm metadados. URL OAuth e código de resposta existem somente em memória/Redis, por até dez minutos. POST `/api/settings/onboarding/:id/authorization-code` exige autenticação administrativa, sessão ativa e desafio do mesmo provider. POST `/api/internal/provider-onboarding/:id/authorization-code` exige autenticação do worker e proprietário exato; GETDEL consome uma vez. NX impede sobrescrita antes do consumo; código com controle/newline é recusado. Conclusão apaga desafio/resposta. Nenhum token/cache, senha, URL completa, código real ou output bruto é enviado a SQL, outbox ou logs.

## Evidência e limites

Clientes presentes: Claude 2.1.285, agy 1.2.14. Diagnósticos privados sem autorização humana confirmaram Claude `claude.com/cai/oauth/authorize`, PKCE e callback web; Antigravity `accounts.google.com/o/oauth2/auth` com código devolvido pelo navegador em SSH. O runner TypeScript real de Claude publicou desafio e expirou em dez segundos, sem login; evidência sanitizada em `evidencias/FAC-018-checks.txt`. O PTY Node real passou fixture de seleção Google, resposta única e inspeção separada. Isso não prova autenticação de conta real.

`gnome-keyring-daemon` ausente nesta VPS: armazenamento nativo real do Antigravity NÃO VERIFICADO. Integração falha com orientação de instalação em vez de usar sessão global. Assinaturas/planos/modelos dependem da conta; Claude mantém catálogo manual. Não há adapter de execução Antigravity; nenhuma execução de ticket foi habilitada.

## Verificação, revisão e rollback

557 testes passaram (scripts 7, contratos 47, runtime 65, API 183, worker 215, web 40). Typecheck e build completos passaram; lint passou com aviso anterior em run-delivery.service.ts:209. Diff sanitizado: [FAC-018-diff.patch](../evidencias/configuracao/evidencias/FAC-018-diff.patch). Dois ajustes de implementação antes dos checks finais: tipagem das respostas do terminal e separação do erro de polling.

Rollback de código: reverter 0182a4fa02dfaeb008cad2079084b251a317b12b após revisar configurações; recompilar e reiniciar supervisor. Preservar diretórios privados de autenticação/keyrings e arquivo de desbloqueio; não limpar credenciais nem banco. Consulte [operação](../operacao/2026-10-05-FAC-018-keyring-autenticacao.md), [runtime](../runtime/2026-10-05-FAC-018-autorizacao-clientes.md) e lessons/lifecycle-processo-cli.md.

Fontes primárias: [Claude autenticação](https://code.claude.com/docs/en/authentication), [Antigravity CLI](https://www.antigravity.google/docs/cli/install/), [node-pty](https://github.com/microsoft/node-pty), [daemon GNOME](https://wiki.gnome.org/Projects/GnomeKeyring/RunningDaemon).
