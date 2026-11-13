> Leitura: [← Anterior](07-fronteiras-e-invariantes.md) · [Índice didático](../02-INDEX.md) · [Próximo →](09-infraestrutura.md)

> Integração DOC-MV-001: texto de direção/planejamento do pacote. O [AS-IS atual](01-as-is.md) prevalece para implementação. Não confundir proposta com controle instalado.

# Segurança e fronteiras

Revisão documental: 2026-10-06 (America/Sao_Paulo). Arquitetura **PLANEJADA**; implementação e eficácia **NÃO VERIFICADAS** na origem do pacote.

A política é imposta por autorização, RLS, broker, capabilities, filesystem, rede e lifecycle fora do modelo. Prompt é orientação, não barreira. Assumir prompt injection bem-sucedida e código potencialmente hostil.

- [Threat model](seguranca/00-threat-model.md): ativos, adversários, riscos residuais e incidente.
- [Rede e egress](seguranca/01-rede-e-egress.md): destinos efetivos, SSRF, dependências e saída de inferência.
- [Tenant isolation](../03-modulos/tenant-isolation/00-README.md): ownership e dados.
- [Runtime](../03-modulos/runtime/00-README.md): auth oficial fora da sandbox.
- [Sandbox](../03-modulos/sandbox/00-README.md): namespaces, mounts, recursos e lifecycle.
- [Tool broker](../03-modulos/tool-broker/00-README.md): ferramentas e capabilities temporárias.
- [SEC-01–14](../08-desenvolvimento/12-testes-seguranca.md): testes negativos planejados, não executados.

Piloto próprio/sintético e admissão de código hostil são gates distintos. Falha de política bloqueia execução. Nenhum documento confirma eficácia de barreira sem teste na versão/configuração real.

## Controles focais de conta — FAC-038

A arquitetura de isolamento amplo descrita acima continua proposta. A camada /account agora tem identidade individual e membership de conta implementadas no diff FAC-038, com cookie próprio/CSRF, senhas scrypt-v1, rate limit Redis fail closed e acesso por papel/organização. [ADR-007](../06-decisoes/ADR-007-conta-usuario-e-sessoes.md). Revogação remove sessões da organização; cookie de conta não abre APIs globais legadas. Convites exigem email verificado; tokens de email e códigos de recuperação têm uso único. MFA tem anti-replay; credenciais privadas/seed TOTP/SMTP usam AES-GCM. Projetos, grants, integrações e jobs têm fronteira lógica de conta revalidada; storage é reservado antes da execução e acesso de streams é rechecado. RLS/broker/isolamento hostil continuam propostos. Fluxos HTTP/DB temporários testados não comprovam release comercial.

AccountExceptionFilter sanitiza exceções desconhecidas em 503 sem expor/logar argumentos privados de queries Prisma; responses de erro também são no-store. Cenário HTTP específico cobre a fronteira.
