# ADR-002 - Controle e execução na mesma VPS
Data: 2026-09-29. Decisão de topologia ACEITA pelo usuário; implementação PLANEJADA.
## Contexto
A v2 separava VPS de controle e worker no MacBook. O usuário optou por executar também clientes oficiais, autenticação, builds e testes na VPS.
## Decisão
VPS Linux única com separação lógica de controle, supervisor, runtime de agentes e código do piloto. Assinaturas via clientes oficiais; APIs/extras desligados. Perfil Bom (8 vCPU/16 GB/200 GB) como recomendação de dimensionamento, contratação ainda pendente.
## Consequências
Independência do MacBook e operação centralizada. Maior VPS, sessões pessoais no servidor, disputa de recursos e ponto único de falha. Isolamento precisa impedir acesso do código às credenciais e ao controle. Revisão e merge/deploy mantêm políticas anteriores.
## Verificação antes de ativar
Login oficial headless e em identidade de serviço; termos/cobrança compatíveis; teste de segredo inacessível; build/testes dentro do envelope; cancelamento/lease, reboot e restauração. Não houve implantação nesta revisão.
## Evolução

FAC-012V (`22f6375`) implementa stores oficiais e home/cache privados por instalação UI no supervisor Linux, fora de checkout/execução: CODEX_HOME/file/ChatGPT e CLAUDE_CONFIG_DIR para assinatura. Allowlist ambiental não expõe credenciais do controle; confinamento de ferramentas é independente. Escrita Claude bloqueada até prova granular. Aceite/preflight pendentes; nenhum cache antigo copiado.

Preservar contratos de worker para futura separação se risco, carga ou disponibilidade justificarem. Não exigir segundo servidor no MVP.

## Detalhamento operacional (OPS-005)
O worker continua processo Node/TypeScript na mesma VPS, mas roda como usuário systemd dedicado do host para que `systemd-run --user` e namespaces do sandbox sejam disponíveis sem container privilegiado. Docker Compose mantém web, API, PostgreSQL e Redis. API/Redis expõem portas somente em loopback para o worker host; o unit, ainda não instalado, recebe credenciais por env file externo. O consumidor de execução permanece ausente até integração real do workflow e aceites/gates operacionais próprios.

FAC-012AA implementa perfil Claude por caminho e preflight oficial opt-in sob UID do serviço; factory exige prova privada current/fingerprint, preservada fora de credenciais/workspaces. Não alterou topologia ou ativou provider/serviço. Prova real continua requisito operacional; fixtures não substituem isolamento/financeiro/identidade verificados.
