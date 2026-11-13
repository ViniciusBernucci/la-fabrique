> Leitura: [← Anterior](00-dimensionamento-vps.md) · [Índice didático](../../02-INDEX.md) · [Próximo assunto →](../operacao/00-referencia-v2.3.md)

> Instruções de criação em documentacoes/ ou lessons/ abaixo são histórico da fonte; novas entregas/lessons seguem exclusivamente docs/00-governanca/POLITICA-DOCUMENTACAO.md.

> Nota editorial DOC-MV-001, 2026-10-07: conteúdo integral local preservado como referência da evolução v2.3. Afirmações por incremento têm a revisão/data original; trechos sobre consumer ausente, recuperação pendente e caminhos antigos não descrevem necessariamente HEAD a2cc5e0. O [AS-IS conferido](../../02-arquitetura/01-as-is.md) e a [auditoria](../../00-governanca/08-AUDITORIA.md) delimitam o estado atual; números de testes abaixo são evidência histórica, não reexecução desta migração.

# Infraestrutura atual

FAC-018: Keyring privado; implementação pronta, consentimento real e keyring Antigravity pendentes. [Relatório](../../09-entregas/2026/configuracao/2026-10-05-FAC-018-login-claude-antigravity.md).

OPS-009: migration 20261002010538_reconcile_prisma_schema_drift registrada como aplicada por prova de schema; histórico antigo mantido e zero falhas ativas. Nove migrations restantes dependem de reconciliação das duas attempts sem stop. [Relatório](../../09-entregas/2026/infraestrutura/2026-10-05-OPS-009-reconciliar-migration-historica.md).
OPS-008: `db:status` inspeciona migrations; `db:deploy` aplica somente quando chamado explicitamente, com `.env` raiz carregado pelo launcher. Banco dev antigo tem divergência de histórico e duas tentativas sem stop; nenhuma migration foi aplicada nesta correção. Não executar reset ou marcar stoppedConfirmed por suposição. [Relatório](../../09-entregas/2026/infraestrutura/2026-10-05-OPS-008-bootstrap-migrations-telas.md).
FAC-012AG versiona factory_operations singleton paused=true por default, versão e check constraints; aplicada só em PostgreSQL efêmero. Índice/estado DB são autoridade, queue Redis pausa entrega sem apagar jobs/volumes. [Evidências](../../09-entregas/2026/controle/2026-10-03-FAC-012AG-pausa-global.md).

FAC-012AF prova pg_dump custom → backup criptografado → staging → pg_restore em DB sintética, preservando eventos/cursor/índice. Chave separada, fontes offline/privadas e restore nunca sobrescreve destino. Storage externo/RPO/RTO não comprovados. [Evidências](../../09-entregas/2026/operacao/2026-10-03-FAC-012AF-backup-restauracao.md).

FAC-012AE versiona eventos/cursors por projeto e triggers PostgreSQL, testados apenas em banco efêmero. Nginx streaming route usa buffering/cache off, sintaxe validada sem deploy. Preservar triggers/check constraints não representáveis no schema Prisma em migrations futuras. [Evidências](../../09-entregas/2026/controle/2026-10-03-FAC-012AE-eventos-projeto.md).

FAC-012AC versiona índice único parcial `attempts_single_unconfirmed_writer`; migration recusa conflitos antigos, não fabrica parada. Aplicada só em PostgreSQL efêmero; futuras migrations devem preservar índice não representável no schema Prisma. [Evidências](../../09-entregas/2026/controle/2026-10-03-FAC-012AC-writer-global.md).

FAC-012X versiona enum RunStatus.WAITING_PROVIDER; migration aditiva não aplicada. TicketStatus já possui o valor. Preservar status/resultado/checkpoint/artifact/journal no backup; não remover enum/dados ao reverter cliente. Nenhuma operação real. [Relatório](../../09-entregas/2026/operacao/2026-10-03-FAC-012X-espera-provider.md).

FAC-012W configura teto JSON API 8 MiB+4096 bytes, DTO 8 MiB/6 MiB raw; transporte worker→API loopback, proxy público mantém limites/bloqueio internal. Considerar memória/concurrency operacional; nenhum serviço alterado nem migração. Retenção/backup manuais, sem apagar artefatos. [Relatório](../../09-entregas/2026/controle/2026-10-03-FAC-012W-artefatos-ampliados.md).

FAC-012V adiciona WORKER_PROVIDER_ROOT no env host: raiz canônica existente 0700/owner serviço, separada de checkout/execução e fora do repositório; não provisionada neste ticket. Clientes usam filhos privados por instalação, nenhum cache antigo copiado. Claude escrita bloqueada até confinamento granular. [Relatório](../../09-entregas/2026/runtime/2026-10-03-FAC-012V-identidades-instalacoes.md).

FAC-012S versiona PAUSED e runs.control_action nullable com constraint PAUSE|CANCEL; migration não aplicada. Preservar pedidos/evidências em backup; atualização operacional só em janela autorizada/writer parado. [Relatório](../../09-entregas/2026/controle/2026-10-03-FAC-012S-comandos-run.md).

FAC-012Q versiona runs.approval JSONB/RunStatus.DONE, não aplica migration. Preservar registro de aceite/documento na restauração; enum aditivo não tem rollback destrutivo automático. [Evidências](../../09-entregas/2026/controle/2026-10-03-FAC-012Q-documentacao-aceite-entrega.md).

FAC-012P versiona migration aditiva de artifact JSONB/artifact_digest em attempts; não aplicada. Preservar snapshots/journal fora do namespace e dados persistidos; sem limpeza automática. Transporte limitado 64 KiB para tickets pequenos. [Evidências](../../09-entregas/2026/controle/2026-10-03-FAC-012P-artefatos-painel.md).

FAC-012C endurece o `SandboxRunner`: workspace read-only por padrão, com caminhos graváveis explícitos validados antes da execução. FAC-012H faz pivot para uma raiz mínima antes de iniciar checks: `/usr` fica read-only, workspace é montado em `/mnt`, e raiz antiga, home, `/run`, `/var` e `/sys` do host não ficam acessíveis. O processo roda sem capabilities e com `no-new-privileges`. Isso não confina o Developer CLI (FAC-012D separado) nem autoriza ativar o consumer real.

FAC-012J prepara checkout confiável por SHA/URL HTTPS/host allowlisted, com credencial efêmera do `gh` oficial após confirmar keyring, Git sem configuração/hook/protocolo local e saída bruta descartada. FAC-012L liga o preparador ao consumer gated; raízes checkout/execução são separadas. Git/GH/keyring na identidade efetiva permanecem pendentes de validação operacional.

OPS-005 fornece serviço systemd dedicado no mesmo host para o sandbox com `systemd-run --user`. Compose mantém controle/bancos e API/Redis em loopback. Unit não instalado; execução desabilitada por padrão. OPS-006 alinha o env example e exige XDG_RUNTIME_DIR/DBus com o UID real do usuário dedicado; `%U` num system unit representa o gestor, não `User=`. Ver [documentação oficial systemd](https://github.com/systemd/systemd/blob/main/man/systemd.unit.xml).

Status: bootstrap do FAC-000 e sandbox/snapshots do FAC-007 aceitos. Implantação em VPS NÃO REALIZADA.

## Componentes

- `web`: React/Vite compilado e servido por Nginx, que encaminha `/api` para a API.
- `api`: NestJS com validação de ambiente, Prisma e health checks para PostgreSQL e Redis.
- `worker`: processo Node.js/TypeScript separado, suportado como serviço host dedicado; não é iniciado pelo Compose.
- `postgres`: persistência relacional com migration inicial para projetos, tickets e outbox.
- `redis`: fila interna do worker e verificação de prontidão da API.

Na composição de controle, Nginx é publicado somente em `127.0.0.1:8080`; API e Redis somente em `127.0.0.1` para acesso local do serviço worker. PostgreSQL não tem porta publicada; os serviços usam redes internas e volumes nomeados. Os containers de aplicação usam filesystem somente leitura, `tmpfs` e `no-new-privileges`.

## Variáveis e execução

`.env.example` atende ao desenvolvimento local. `.env.production.example` documenta somente Compose/control plane e contém valores sintéticos. O worker host recebe configuração de `/etc/le-fabrique/worker.env`, nunca versionada; o exemplo está em `infrastructure/systemd/worker.env.example`.

O bootstrap local atual e:

```bash
cp .env.example .env
docker compose -f compose.dev.yaml up -d
npm ci
npm run db:generate
npm run db:deploy
npm run dev
```

O launcher raiz carrega `.env` antes de iniciar os tres workspaces. O painel usa `VITE_API_URL=/api` e o proxy do Vite, evitando que um browser remoto interprete `localhost` como a propria maquina. `db:deploy` aplica migrations existentes; `db:migrate` e reservado a autoria de uma nova migration e nao faz parte de um simples start.

```bash
cp .env.production.example .env
docker compose config
docker compose up -d --build
```

Esse comando não inicia o worker. Para futura ativação manual, após ticket de integração e gates operacionais aceitos, provisionar usuário `le-fabrique-worker`, instalar Node/Git/GH e cliente(s) oficial(is) por fonte/verificação apropriada, configurar keyring sob essa identidade, preparar `/opt/le-fabrique` e `/etc/le-fabrique/worker.env`, habilitar user manager/linger, validar mounts/namespaces e só então instalar/ativar o unit. Este ticket não executa nenhum desses comandos nem autentica clientes. Verifique `GET http://localhost:8080/api/health/live` e `/ready`. Para interromper o controle sem apagar dados, execute `docker compose down`; a remoção de volumes não faz parte do rollback padrão.

Provisionamento, HTTPS, firewall, backup externo, restauração, isolamento de clientes oficiais e limites reais da VPS continuam sujeitos aos tickets de infraestrutura do MVP.

OPS-003 versiona uma migration que remove defaults PostgreSQL de UUID gerados pelo Prisma Client e renomeia o índice de verificações GitHub para refletir o schema. A migration ainda não foi aplicada a banco algum.

## Sandbox atual

FAC-007 implementa worktree detached, unidade systemd de usuario com cgroup e snapshot local com patch binario/untracked verificados por SHA-256. FAC-012H complementa os namespaces com raiz isolada: apenas runtime `/usr` somente leitura, workspace `/mnt`, `/proc` da execução, devices mínimos e temporários isolados são montados; raiz antiga é desanexada antes do processo não confiável iniciar. O supervisor confiavel continua fora do namespace para operar Git e artefatos. Integracao ao worker e persistencia pertencem ao FAC-008.

Checkout é materializado fora do sandbox sob root absoluto privado (`0700`) da identidade do worker. Antes da ativação, provisionar roots/hosts exatos, versões compatíveis de Git/GH/CLIs e keyring. Consumer real FAC-012L fica desligado por padrão, sem fixture em serviço. Dockerfile.worker legado não constitui caminho suportado de execução.
