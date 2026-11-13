# FAC-038 — Painel da conta do usuário

Data: 2026-10-09 (Europe/Berlin). Branch feat/fac-038-painel-conta; base `ee4485bffb517247cd7b584653ad9ec1423f9202`. Implementação IMPLEMENTADO no diff local, AWAITING_HUMAN. Sem commit/merge/push/deploy. Proposta PLAN-ADM-001 prévia preservada junto ao diff. Responsável autorizou executar; administração global excluída explicitamente.

## Objetivo, funcionamento e critérios

Criar a primeira versão funcional de conta individual/equipe sem distribuir o token administrativo compartilhado. [Ticket](../../08-desenvolvimento/tickets/FAC-038.md), [decisão](../../06-decisoes/ADR-007-conta-usuario-e-sessoes.md), [feature](../../04-features/01-control-settings.md#painel-da-conta-do-usuário--fac-038).

Avatar Minha conta e menu Usuários abrem AccountPanel. Cadastro cria usuário e organização pessoal com proprietário; login usa email/senha e sessão dedicada. Abas: Meu perfil, Minha conta, Pessoas e acesso, Segurança, Plano e uso, Atividade. Perfil/preferências e nome da conta são versionados; cancelar descarta rascunho. Convites têm link privado de uso único/72h/email correspondente; não há envio automático. Gestores administram papéis/status dentro da própria organização, com owner protegido. Suspensão/removal remove sessões da organização. Senha exige a atual e encerra outras sessões; logout/sessões próprias permitem revogação. Histórico mostra ator/ação/data das últimas 50 ações para gestores.

Plano comercial ausente é informado. Membros/convites são contagens reais; métricas de projetos/execuções/storage/IA são indisponíveis. Preferências de notificação ficam salvas, sem entrega automática prometida. Email não verificado, alteração de email/MFA/recuperação/avatares/upload/transferência de ownership/dados fiscais/pagamento/exportação não implementados. Papéis de projeto ainda não concedem acesso aos projetos existentes.

Critérios 1–8 do ticket foram cobertos por contratos, testes de segurança, HTTP/PostgreSQL e navegação/fetch. Fluxo interativo visual em browser NÃO VALIDADO; implantação e revisão independente pendentes. Não declarar aceito/DONE.

## Código, dados, API e configuração

AccountModule/Service/Guard/controllers/security na API; contratos account.ts exportados no pacote compartilhado; AccountPanel/account-api/CSS/navegação/App na web. Carregamento lazy da área mantém o bundle inicial separado. Nenhuma dependência de biblioteca, provider ou serviço novo.

Migration aditiva `20261009120000_user_accounts` cria seis tabelas com FKs/checks/índices. Aplicada somente ao PostgreSQL temporário `fac038_test`, em container novo baseado na imagem já disponível postgres:18.1-alpine; não houve consulta/migration do banco ativo. Generated Prisma Client atualizado localmente. [Dados](../../05-contratos/schemas/00-entidades.md#conta-do-usuário--fac-038) e [API](../../05-contratos/api/01-controle.md#api-da-conta-do-usuário--fac-038) descrevem os contratos físicos.

Cookie HttpOnly/SameSite Strict/Secure em produção, secret hash SHA-256, CSRF e contexto de conta; filtro de exceções da conta impede resposta/log automático com argumentos privados de falhas Prisma; senha scrypt-v1 com salt/parâmetros explícitos e capacidade de hashing bounded. Redis por email/IP do socket limita login/cadastro e falha fechado. Testes PostgreSQL usam fake do Redis para cenários de disponibilidade/limite, não são integração Redis real. ADMIN_API_TOKEN/WORKER_API_TOKEN e cookie humano não se substituem. Config de origem/HTTPS/proxy e política de cadastro público exigem rollout próprio.

## Diff, fontes e evidências

Código recuperável via `git diff`, arquivos novos via `git status --short`; sem commit ou patch publicado. [Estado e hashes](evidencias/FAC-038/estado.json). Leitura explícita de AGENTS, políticas/perfil/skills locais, Manual Vivo, README, piloto, proposta, capítulos e código. Skills utilizadas: tech-lead, implementador-de-ticket e orquestrador local; papéis não criaram subagentes. Autoload não presumido; fontes/hashes do planejamento constam em [PLAN-ADM-001](2026-10-09-PLAN-ADM-001-painel-administrativo-conta.md#diff-e-evidências).

## Checks e resultados

- Prisma generate: PASS; validate em URL sintética do DB efêmero: PASS no workspace API.
- Contratos: 58 testes PASS; API padrão antes do cenário de sanitização adicional: 197 PASS e 9 integração opt-in ignorados quando sem ACCOUNT_TEST_DATABASE_URL.
- Integração focal Account HTTP/PostgreSQL final: 10 cenários PASS; segurança focal: 5 PASS, incluindo hashing/origem/CSRF/roles.
- Web: 70 testes PASS, incluindo navegação/cookie/contexto e fragmento sob StrictMode.
- `npm run typecheck`: PASS para todos os workspaces/scripts/integração.
- `npm run build`: PASS. Primeiro build web acusou chunk >500KB; área account foi separada por lazy import, build final PASS com lazy import: inicial 488.37KB e account 18.53KB, sem aviso >500KB.
- `npm run lint`: PASS, com uma advertência preexistente em run-delivery.service.ts; nenhum fix alheio aplicado. [Log](evidencias/FAC-038/lint.txt).
- `npm test` com ACCOUNT_TEST_DATABASE_URL apontando apenas ao DB efêmero: PASS, 623 testes (7 scripts, 58 contratos, 65 runtime, 207 API incluindo 10 PostgreSQL, 216 worker, 70 web). [Log](evidencias/FAC-038/checks.txt).
- `python3 scripts/validar-documentacao.py`: FAIL pelos mesmos sete Markdown handoff-FAC-035 fora do índice, sem novos erros; 401 Markdown/2488 links/601 snapshots conferidos. `git diff --check`: PASS.
- Browser: NOT_RUN; Chromium instalado não inicia por falta de libatk-1.0.so.0. Não foram instaladas dependências do host para contornar.

Falha inicial de import no teste foi diagnosticada como decorators na própria fixture spec; aplicados programaticamente, como nas fixtures existentes. Resultado esperado da senha atual incorreta foi alinhado ao 403 escolhido para manter a sessão no frontend; suite final focal PASS. Não foram alterados testes de software legado para esconder regressões.

## Segurança, riscos e limites

Não há ownership de projetos/configurações/runtime anteriores, RLS ou mediação broker nova. Sem métricas comerciais/excedentes inventados. Não houve instalação/login de provider, acesso a credenciais ou inferência API. Links de convite só ficam no estado da tela e hash na persistência; fragmento é removido após captura, compatível com StrictMode. Email/login não prova identidade comercial. Cadastro público não abre operação global, mas publicação comercial exige verificação/recuperação/MFA e os gates LF-MT aplicáveis.

Sessões possuem expiração fixa de sete dias e máximo 20 por usuário; limites 200 de vínculos são capacidade operacional, não direito de plano. Históricos não são removidos com suspensão/remoção de vínculo. Rotina de retenção/limpeza deve ser planejada. Rate limit por IP do socket pode agrupar usuários atrás de proxy; não confiar em forwarded headers arbitrários.

## Rollback e documentação

Reverter somente alterações FAC-038, preservando PLAN-ADM-001 e trabalhos posteriores. Remover entrada AccountPanel/AccountModule; preservar tabelas com dados até procedimento autorizado. Nenhum DROP, fallback de identidade humana para token global ou rollback de produção foi executado. [Rollout](../../07-operacao/01-deploy.md#rollout-da-camada-de-conta--fac-038).

Atualizados README/Manual, AS-IS/componentes/dados/segurança, configuração/feature, contratos de entidade/API e catálogo JSON, ADR-007/catálogo, runbook, matriz, handoff/backlog/changelog/índice/catálogo de entregas. Sem novo capítulo didático; ADR/ticket/entrega conservam IDs, sem renumerar mapa. Nenhuma lesson nova: aplicação focal documentada no ADR/contratos, sem duplicação do manual.

## Custos, pendências e aceite

Modelo/versão/custos efetivos da sessão Codex não verificados; uso desconhecido. Banco temporário local usa imagem já disponível; sem contratação/API/provider/gasto habilitado. Container PostgreSQL efêmero criado nesta sessão foi parado e removido automaticamente ao final; serviços existentes preservados.

Migration/deploy no ambiente ativo, browser, revisão independente e aceite humano exato pendentes. Sem subagentes/revisão simulada; autorrevisão técnica não equivale a aprovação independente. FAC-039 ownership/grants projetos/jobs, FAC-040 integrações/uso e FAC-041 verificação/MFA/recovery/oferta comercial permanecem PLANEJADOS e não READY. Administração global fica para trabalho posterior.

## Continuação posterior na mesma branch

Este relato conserva a evidência do primeiro incremento. O pedido seguinte determinou completar as demais partes; FAC-039–041 acrescentam os recursos antes ausentes. Estado corrente, checks e limites constam na [entrega complementar](2026-10-09-FAC-041-complemento-painel-conta.md). As alegações de ausência acima são históricas do FAC-038, não o estado atual do painel.
