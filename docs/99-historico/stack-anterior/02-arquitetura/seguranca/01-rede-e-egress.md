> Leitura: [← Anterior](00-threat-model.md) · [Índice didático](../../02-INDEX.md) · [Próximo assunto →](../../03-modulos/00-README.md)

# Rede deny-by-default e saída controlada

Data: 2026-10-06 (America/Sao_Paulo). Status da arquitetura e controles: **PLANEJADO**. Implementação e eficácia: **NÃO VERIFICADAS**. Este documento especifica trabalho futuro; não comprova instalação, configuração ou execução.

## Matriz de fluxos

| Origem → destino | Regra |
|---|---|
| Usuário → proxy HTTPS | permitido com auth/limites; SSH administrativo restrito à origem autorizada |
| NestJS/serviços confiáveis → Postgres/Redis | identidades específicas, privado; não público |
| Worker → protocolo interno do controle | canal autenticado mínimo; não visível a modelo/sandbox |
| Executor → helper lifecycle | IPC restrito, perfis fixos, sem parâmetros administrativos livres |
| Provider Runtime → fornecedor da instalação | somente endpoints oficiais necessários, por perfil e tenant |
| Provider Runtime → broker da tentativa | IPC/protocolo dedicado com capability curta; sem API geral |
| Broker → sandbox | canal da tentativa apenas; nenhuma autoridade de controle |
| Sandbox → serviços sintéticos da mesma tentativa | só portas cadastradas, sem ponte para outras redes |
| Sandbox → internet | negado; exceção explicitamente concedida passa por fetch/egress controlado |
| Sandbox/runtime do modelo → controle/DB/Redis/auth/outros tenants/daemon | negado |

Interações administrativas de Executor não são exportadas ao modelo. IPC para ferramenta é exceção estreita de comunicação, não endpoint capaz de consultar a fábrica. Rede host, default bridge compartilhada, descoberta lateral e portas publicadas por job são proibidas. Loopback do namespace só alcança processos da tentativa; negar proxy para loopback do host. Bloquear também URL pública da própria fábrica a partir da zona de execução, mesmo se resolver para IP público.

## Enforcement

Namespaces e firewall de forwarding reais, regras para IPv4 e IPv6 e verificação após criação/reboot. Docker pode alterar rotas/regras; não presumir que regra de firewall de entrada do host bloqueia tráfego de containers. Sem DNS externo direto, UDP/QUIC livre, raw sockets, CONNECT genérico ou proxy configurável pelo repo. Se proxy/firewall/resolvedor falhar ou política faltar, negar acesso e suspender jobs; nunca bypass direto.

Bloquear loopback externo ao namespace, RFC1918, link-local/metadata, faixas reservadas/multicast, IPv6 ULA/link-local/loopback e IPv4 mapeado em IPv6; inventariar IPs/interfaces do host, gateways, serviços públicos da fábrica e endpoints metadata do fornecedor da VPS. Bloquear por destino efetivo, não só hostname. Revalidar resolução e conexão para impedir rebinding; redirecionamento exige nova checagem completa a cada hop. Parser canônico rejeita URLs ambíguas, userinfo, formatos numéricos alternativos e protocolos não aprovados. [OWASP SSRF](https://cheatsheetseries.owasp.org/cheatsheets/Server_Side_Request_Forgery_Prevention_Cheat_Sheet.html) fundamenta as checagens de destinos e DNS.

## Dependências e exfiltração

Preferir preparação determinística de dependências por serviço confiável que obtém coordenadas/version/lockfile autorizados, valida integridade e entrega blobs; scripts de instalação rodam apenas no sandbox. Não permitir scripts no fetcher confiável. Acesso aberto a github.com/npmjs.org não comprova prevenção de exfiltração: atacante pode controlar repositório/pacote/URL no domínio permitido. Allowlist deve limitar operações, caminhos, métodos e tamanho; upload/publicação/search autenticado não fazem parte de fetch de dependências.

Egress de inferência é separado do egress de build. O fornecedor recebe somente contexto autorizado do projeto; dados desse projeto podem sair para ele por design. Não prometer confidencialidade contra o próprio fornecedor nem proteção de conteúdo já autorizado ao modelo. Para browser que precisa sites arbitrários, usar etapa explicitamente aprovada com dados sintéticos, sem segredos e proxy validado; registrar risco de exfiltração dos dados da etapa. Nenhuma exceção autoriza outro tenant ou o controle.

Logs de proxy contêm tenant/run/policy e destino sanitizado, sem headers auth, corpo completo ou query sensível. Testes precisam provar pacotes negados e ausência de conexão no destino canário, não só erro de ferramenta da IA.


## Origem desta edição

[Versão original preservada](../../99-historico/originais/output/le-fabrique-multitenant/documentacoes/seguranca/REDE-E-EGRESS.md). Migração editorial de paths em 2026-10-06; conteúdo de engenharia continua proposto.
