> Leitura: [← Anterior](07-perfis-permissao-cliente-cli.md) · [Índice didático](../02-INDEX.md) · [Próximo →](09-leases-fencing-checkpoints.md)

# Lifecycle seguro de um cliente CLI

FAC-012Y aplica `DeveloperWorkflow.tryHandoff`: quiescência → snapshot posterior → nova worktree/restore → contexto/rota atual. Falha de restore preserva origem e não chama destino. `ConfiguredAgentRouter.resolve(role, excluded)` só usa alternativas explícitas; ausência espera, erro de segurança não autoriza fallback. Mesma tentativa conserva lease/fence e budgets, novo job exige fence novo. Teste de orçamento total prova que restore não reinicia chamadas disponíveis; histórico de handoff não equivale a prova de chamada do destino.

FAC-012X distingue falha de código e indisponibilidade de provider. `DeveloperWorkflow.waitForProvider` exige quiescência, captura snapshot e retorna WAITING_PROVIDER; AUTH_REQUIRED/RATE_LIMITED/PROVIDER_BUSY não consomem loops de correção. Rejeição com stop unknown continua propagando/bloqueando, mesmo se o texto mencionar auth. Typed ProviderUnavailableError evita classificar configuração/segurança inválidas como espera. Processor/journal/API compartilham outcome/checkpoint para não perder o motivo na finalização.

FAC-012R demonstra que Set vazio não prova quiescência: rejeição de execução/check desconhecido deixa flag terminationUnknown sticky. Só AbortSignal após término conhecido permite capturar snapshot e publicar CANCELLED com observações parciais. `execution.processor` lança InterruptionEvidenceError se workflow iniciado não retornar evidência, mesmo que cancelActive seja true; não completa com snapshot inventado.

## Conceito aplicado

Executar um cliente oficial exige controlar o processo, nao apenas interpretar sua resposta. O gateway precisa distinguir pedido de cancelamento, timeout, excesso de logs, falha do provider e resultado incompleto, encerrando a arvore antes de liberar outro writer.

No FAC-005, cada `executionId` aponta para um unico grupo de processos ativo. Cancelamento e timeout enviam `SIGTERM`, aguardam o fechamento e possuem fallback `SIGKILL`. O resultado final deriva do motivo registrado pelo supervisor e do evento `turn.completed`, em vez de confiar somente no exit code.

## Eventos como entrada nao confiavel

O JSONL do cliente e validado e reduzido a eventos internos. Mensagens de raciocinio sao ignoradas; output bruto de comandos e stderr servem apenas para classificacao limitada e nao sao devolvidos. Uso e modelo so aparecem quando o evento realmente os informa.

## Exemplo do repositorio

`packages/runtime/src/codex-adapter.ts` mantem o mapa de processos ativos, limita bytes e produz contratos de `packages/contracts`. `fake-codex.cjs` simula sucesso, auth, cota, contexto, permissao, timeout, cancelamento, JSON invalido e excesso de output sem consumir assinatura.

FAC-012L conecta lifecycle a lease/fencing/checkpoint no consumer desabilitado por padrão. AbortSignal aciona cancelamento do runtime e espera a execução terminar; perder autoridade nunca aceita o resultado como sucesso. O sinal também alcança o SandboxRunner, que encerra a unidade/cgroup e distingue parada observada de consulta indisponível. A prova da identidade de serviço e a recuperação de tentativas interrompidas continuam pendentes.

No FAC-010A, ate um probe de autenticacao recebe lifecycle: comando/argv sao fixos por provider, shell fica desligado, ambiente perde chaves de API herdadas, processo tem 15 s e 64 KiB e somente classificacao sanitizada e persistida. Exit code zero nao basta: `agy models` pode responder "sign in" sem modelo, portanto a classificacao procura evidencia de autenticacao antes de declarar `AVAILABLE`.

No FAC-010B, o lifecycle inclui uma informacao sensivel de curta duracao. O processo `codex login --device-auth` pode viver dez minutos, mas URL/codigo sao extraidos em memoria e publicados somente no Redis com TTL menor ou igual ao da sessao. PostgreSQL recebe estado, nao desafio. Falha ao publicar mata o processo; SIGINT/SIGTERM mata todos os logins ativos; e exit code zero ainda exige `codex login status` antes de atualizar a instalacao para `AVAILABLE`. Assim, vida do processo, vida do desafio e evidencia persistente sao limites distintos.

No FAC-011A, uma CLI de integracao traz outro risco: variaveis de ambiente podem substituir a credencial armazenada oficialmente. O worker remove `GH_TOKEN`, `GITHUB_TOKEN`, `GH_ENTERPRISE_TOKEN` e `GITHUB_ENTERPRISE_TOKEN`, fixa `gh auth status --hostname <host> --json hosts` e nunca usa `--show-token`. O JSON pode conter login, scopes e origem da credencial; ele e classificado em memoria e descartado. Se o host administrativo mudar durante o processo, a conclusao antiga falha sem atualizar a configuracao nova. Assim, argv seguro, ambiente seguro e validade do snapshot de configuracao sao verificacoes independentes.

No FAC-011B, sucesso do device flow e armazenamento seguro tambem sao evidencias separadas. `gh auth login --web` pode terminar com zero e ainda usar fallback `hosts.yml`; por isso o worker exige uma segunda leitura com conta ativa `success` e `tokenSource=keyring`. O desafio tem vida menor que a sessao, Redis falha fechado e shutdown mata a arvore. O sistema sinaliza fallback em texto simples, mas nao apaga/desloga automaticamente porque essa recuperacao poderia atingir outra conta; remediacao de credencial continua um gate humano.

No FAC-011C, uma consulta rotulada como leitura ainda precisa limitar metodo, alvo e evidencia. O worker fixa `gh api --method GET`, deriva endpoints apenas do snapshot validado e codifica owner, repositorio e branch como segmentos. Metadados do repositorio so sao persistidos se a segunda consulta confirmar a branch base e `permissions.pull` for verdadeiro; qualquer falha descarta tambem os dados parciais da primeira resposta. A API reconfere o snapshot no complete e normaliza callbacks antigos de forma idempotente. Assim, read-only envolve tanto ausencia de verbo mutavel quanto consistencia temporal e minimizacao do resultado.

FAC-017: clientes oficiais podem enviar ANSI mesmo com stdout em pipe. provider-onboarding.processor usa stripVTControlCharacters e só publica código de linha completa, preservando caso, para não persistir URL com escapes ou código cortado entre chunks. Fixture de texto simples não comprova formato de CLI real; diagnóstico isolado confirma publicação do desafio sem expor o código ou autorizar conta. A UI reserva aba no clique, mas só status oficial confirma assinatura.

FAC-018: um cliente interativo pode exigir PTY, respostas fixas de terminal e retorno OAuth via stdin. provider-authorization.processor separa desafios de códigos retornados; Redis NX/GETDEL com owner/TTL evita sobrescrita/reuso pelo canal. Antigravity usa D-Bus novo e keyring por conta, sem fallback global. Prova de URL, fixture e login real são evidências diferentes; keyring ausente impede a última.
