> Leitura: [← Anterior](00-README.md) · [Índice didático](../02-INDEX.md) · [Próximo →](02-execucao-e-recuperacao.md)

> Integração DOC-MV-001: texto de direção/planejamento do pacote. O [AS-IS atual](../02-arquitetura/01-as-is.md) prevalece para implementação. Não confundir proposta com controle instalado.

# Runbook de deploy

Revisão documental: 2026-10-06 (America/Sao_Paulo). Arquitetura **PLANEJADA**; implementação e eficácia **NÃO VERIFICADAS** na origem do pacote.

## Pré-condições e responsável

Responsável administrativo e janela autorizada; VPS/OS/kernel/versões e backup testado; revisão de código/config/imagem/política identificada; gates de segurança e compatibilidade por provider. Sem esses dados este procedimento permanece PLANEJADO.

## Sequência planejada

1. Bloquear novos claims e confirmar fim dos writers; preservar checkpoints/artifacts recuperáveis.
2. Registrar baseline, revisão e config segura anterior; separar volumes temporários/persistentes.
3. Aplicar proxy HTTPS, serviços internos privados, identidades e limites agregados conforme perfil validado.
4. Migrar dados somente com plano/backfill/ownership/restore aprovado; não atribuir órfãos arbitrariamente.
5. Aplicar versão de controle/worker/perfis e executar checks autorizados, incluindo negativos afetados.
6. Validar heartbeat, ownership, auth segregada, admission, observabilidade e backup após reboot de ensaio.
7. Reabrir admission apenas após gate e aceite aplicáveis. Provider incompatível permanece off.

## Rollback e evidência

Suspender jobs, preservar estado e restaurar versão segura de código/config; migração de dados exige estratégia própria testada. Nunca remover RLS/egress/segregação para reverter. Registrar resultado observado e versões/hashes reais. Compose e templates systemd existem no repo: compose.yaml e infrastructure/systemd. Não foram executados/instalados por esta migração; não afirmar estado do serviço a partir do template. Consulte a referência local de infraestrutura e ambiente-local para comandos exatos.

## Base local

[Infraestrutura integral](infraestrutura/01-referencia-v2.3.md), [Compose controle](../../compose.yaml), [setup](../08-desenvolvimento/01-ambiente-local.md).

## Rollout da camada de conta — FAC-038

[Migration](../../apps/api/prisma/migrations/20261009120000_user_accounts/migration.sql) adiciona somente tabelas de conta. Antes de implantar, conferir migrations pendentes/estado real do banco e backup; não aplicar o histórico inteiro por inferência. Este trabalho não executou db:deploy/db:migrate no banco ativo. Prisma generate e a migration foram testados em banco temporário separado.

API e painel devem usar a mesma revisão dos contratos; frontend recomendado no mesmo origin da API /api, com HTTPS, NODE_ENV=production e WEB_ORIGIN correto. Cookie usa Path=/api/account; proxy deve preservar este prefixo/Host. Rate limit usa IP do socket sem confiar em X-Forwarded-For; detrás de proxy pode compartilhar limite entre usuários. Redis indisponível bloqueia login/cadastro; avaliar proxy/IP em janela operacional. Cadastro público de identidade não abre recursos legados, mas implantação pública depende de política explícita de cadastro, verificação/recuperação/MFA e operação.

Reversão: retirar entrada do painel/AccountModule sem converter identidades em ADMIN_API_TOKEN; preservar tabelas/dados pessoais até procedimento de retenção aprovado. Não executar DROP automático.

## Configuração de email, segurança e planos — FAC-039–041

Migration complementar exige backup e janela operativa: acrescenta ownership nullable/grants/tokens/configuração privada/notifications/planos/reservas. Não atribui Project legado a conta. Nenhum comando de deploy/migration foi executado no DB ativo; ensaio usa account_test efêmero. Reverter código sem remover dados/constraints; retirar admissão nova e confirmar término de writers antes de intervenção.

Usar variáveis exemplificadas em .env.example/.env.production.example, guardadas fora do Git: ACCOUNT_DATA_ENCRYPTION_KEY (32 bytes hex/AES-GCM) para segredos/MFA/outbox; ACCOUNT_PUBLIC_URL fixo HTTPS no prefixo real da web; ACCOUNT_SMTP_ENABLED/host/port/secure/user/password/from. Produção SMTP exige TLS; filas falhas têm tentativas limitadas e não expõem corpos/erros privados. Preservar chave dedicada em backup privado; sua perda torna ciphertext irrecuperável. Rotação precisa recriptografar registros em manutenção própria, não apenas trocar a env.

ACCOUNT_PLAN_CATALOG é JSON validado de planos versionados (id/name/version/priceMinor/currency/interval/stripePriceId/members/projects/storageBytes). Sem valor, catálogo vazio e limites operacionais, nunca plano fictício. EUR/BRL/USD/GBP usam centavos. Definir planos/preços/cotas com o responsável antes de habilitar comercial. Limites não aumentam concorrência/writer. Stripe precisa ACCOUNT_STRIPE_ENABLED/SECRET_KEY/WEBHOOK_SECRET, preços idênticos ao catálogo e portal com trocas/cancelamento configurados. Cadastrar webhook de snapshot checkout.session.completed/customer.subscription.*/invoice.* na URL HTTPS /api/account/billing/webhook; body bruto preservado pelo Nest. Não registrar assinatura/header/corpo/chaves no proxy.

O painel usa resultado da Stripe, não query string de retorno. REQUESTED sem resposta conhecida mantém UUID/idempotency key; repetir/reconciliar a mesma intenção dentro de 24h. Após esse prazo, conferir no operador Stripe antes de criar/substituir intenção; não apagar pendência às cegas. Checkout aberto pode ser retomado pelo botão de atualização; expirado é marcado EXPIRED. Assinatura cancelada/inadimplente pausa admissões sem remover histórico. Provedor externo/SMTP/Stripe não foi conectado nesta entrega; testes usam fake SMTP/HTTP e segredos rotulados fixture.

Instalações novas `ac-UUID` usam private provider root existente; não copiar auth entre instalações. Projetos de conta exigem uma integração GitHub própria autorizada. Canal interno de credencial somente em rede privada/HTTPS/loopback com WorkerAuthGuard e lease/fence; token nunca é persistido em checkout/job/artifact ou exposto ao modelo. Novo controle/worker devem ser implantados juntos: worker agora solicita contexto de projeto e credencial do supervisor. Hosts github.com HTTPS sem credenciais embutidas; API de IA continua off.

Fontes primárias dos adaptadores: [SMTP Nodemailer](https://nodemailer.com/smtp), [mensagens sem acesso a arquivos/URLs](https://nodemailer.com/message), [Stripe webhooks/raw body](https://docs.stripe.com/webhooks), [Checkout](https://docs.stripe.com/api/checkout/sessions/create), [consulta de assinatura](https://docs.stripe.com/api/subscriptions/retrieve), [GitHub usuário autenticado](https://docs.github.com/en/rest/users/users#get-the-authenticated-user).
