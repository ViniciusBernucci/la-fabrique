> Leitura: [Catálogo de decisões](00-README.md) · [Índice didático](../02-INDEX.md)

# ADR-007 — Conta do usuário e sessões individuais

Data: 2026-10-09 (Europe/Berlin). Status: PROPOSTA IMPLEMENTADA em FAC-038–041; aceite da revisão exata e implantação pendentes. Pedido do responsável autoriza execução do painel da camada de usuário; administração global será feita depois. Não substitui ADR-003 TypeScript aceito nem promove controles LF-MT.

## Problema e decisão focal

ADMIN_API_TOKEN identifica acesso administrativo compartilhado, não uma pessoa. Não atribuir esse token a membros nem transformar o cadastro de uma conta em acesso ao controle global. Criar identidade humana e plano de conta independentes das rotas administrativas existentes.

Conta individual recebe AccountUser e AccountOrganization com um proprietário. AccountOrganization representa a organização/tenant lógico da camada de conta; não é uma segunda identidade concorrente com Tenant. Integração LF-MT futura deve reutilizar ou migrar explicitamente esse ownership, nunca criar donos paralelos para os mesmos recursos. Não há vínculo automático com Project/FactorySettings/provider installation anteriores.

Sessões PostgreSQL, cookie HttpOnly/SameSite Strict e Secure em produção, segredo aleatório de 256 bits persistido como SHA-256 e validade fixa de sete dias. CSRF separado da autenticação, cabeçalho de requisição e origem permitida; mutações exigem organização ativa esperada para evitar salvar na conta selecionada por outra aba. Senhas scrypt-v1 (N=32768/r=8/p=3, salt aleatório), trabalho limitado a quatro hashes simultâneos. Redis limita autenticação por email e IP do socket; falha impede login/cadastro.

Papéis fixos OWNER/ADMIN/OPERATOR/APPROVER/READER em AccountMembership; somente OWNER/ADMIN gerenciam equipe. Grants por projeto conferem leitura/operação/aceite; OPERATOR/APPROVER exigem o papel correspondente na organização. Proprietário não pode ser despromovido/suspenso/removido. Administração de ADMIN é restrita ao OWNER.

Convites por link privado com segredo hash, validade de 72 horas e uso único, aceitos por conta autenticada com email correspondente. Email correspondente precisa ser verificado; SMTP configurado entrega links, mantendo compartilhamento privado como opção. Reemissão revoga convite pendente anterior. Lock transacional da organização serializa convite/membership/configuração; auditoria acompanha mutações. Membros removidos permanecem como vínculo histórico e podem ser reativados por administrador autorizado.

## Alternativas e consequências

Token compartilhado para todos foi rejeitado: não revoga acesso individual nem atribui autoria. JWT somente no cliente foi evitado: exige mecanismo adicional para revogação imediata. Provedor externo de identidade fica para requisito específico, sem contratação/dependência nova. Sessões no banco introduzem consultas e rotina futura de retenção, mas já aproveitam persistência disponível.

Cadastro da camada de conta não exige token global e não abre recursos globais. Implantação pública exige configurar SMTP/chave privada, política de MFA/cadastro permitido e prontidão operacional; verificação/recuperação/MFA estão implementadas. Contas novas ficam sem plano comercial e sem métricas de recursos legados; desconhecido não vira zero. Limite operacional de 200 vínculos por conta/usuário e 20 sessões não é plano comercial.

## Verificação e reversão

[Ticket FAC-038](../08-desenvolvimento/tickets/FAC-038.md) e [entrega](../09-entregas/2026/2026-10-09-FAC-038-painel-conta-usuario.md). Testes isolados não comprovam produção nem isolamento de código hostil. Migration FAC-038 cria identidade; FAC-039–041 adicionam colunas/vínculos compatíveis em tabelas existentes, sem backfill arbitrário. Reversão de código remove entrada/módulo account; tabelas com dados ficam preservadas, sem DROP automático nem fallback das contas para token global.

## Complemento autorizado de escopo

Pedido explícito para terminar as partes restantes em 2026-10-09 autoriza implementar ownership de projetos/jobs/integrações, dados/uso, segurança e adaptadores de planos. [Tickets](../08-desenvolvimento/09-backlog.md#fac-039041--complemento-da-camada-de-usuário) e [entrega complementar](../09-entregas/2026/2026-10-09-FAC-041-complemento-painel-conta.md). SMTP via Nodemailer e Stripe REST são adaptadores condicionais por env privado; nenhum provider real ou cobrança foi ativado. Dados comerciais não foram presumidos. Integração de clientes hostis/RLS/broker e administração global continuam fora desse incremento.
