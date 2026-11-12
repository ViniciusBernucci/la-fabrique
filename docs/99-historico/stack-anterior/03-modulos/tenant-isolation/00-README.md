> Leitura: [Índice didático](../../02-INDEX.md) · [Próximo assunto →](../tool-broker/00-README.md)

# Identidade, autorização e segregação de dados

Data: 2026-10-06 (America/Sao_Paulo). Status da arquitetura e controles: **PLANEJADO**. Implementação e eficácia: **NÃO VERIFICADAS**. Este documento especifica trabalho futuro; não comprova instalação, configuração ou execução.

## Resolução de autoridade

```text
principal autenticado → membership válida no tenant
tenant → project autorizado → run → step/attempt
→ provider installation elegível do MESMO tenant → credential_ref do MESMO tenant/provider
```

Tenant selecionado na interface é solicitação, não autoridade. NestJS autentica principal e verifica membership/role/project grant a cada operação. Tenant efetivo nunca é aceito de prompt, headers não autenticados, eventos de IA ou somente de UUID. A cadeia é reconsultada pelo servidor e novamente pelo Executor no dispatch/claim. Mudança de membership, instalação revogada ou política alterada invalida operação futura e revoga attempts afetadas.

## Modelo planejado

| Entidade | Campos adicionais e integridade |
|---|---|
| Tenant | id opaco, status, policy_version |
| Membership / ProjectGrant | tenant_id, principal_id, papel, project_id quando limitado |
| Project | tenant_id, id, repo/ref cadastrado; UNIQUE(tenant_id,id) |
| Ticket / Run / Step / Attempt | tenant_id e cadeia de IDs; FKs compostas; estado/version/fencing |
| ProviderInstallation | tenant_id, id, provider, worker/runtime_identity, credential_ref opaco, preflight evidence/version/status |
| CredentialMetadata | tenant_id, provider_installation_id, ref opaca, generation/status; nenhum token no banco |
| Artifact / Checkpoint / ContextManifest / Approval | tenant_id/project_id/run_id/attempt_id, hashes, revisão e policy_version |
| UsageObservation / Ledger / Audit | tenant_id e instalação/attempt quando aplicável; uso unknown permitido |

FK (tenant_id,project_id) referencia Project(tenant_id,id); o mesmo padrão liga run, attempt e instalação. Grant de instalação por projeto limita escolha dentro do tenant. Sharing de assinatura apenas entre projetos autorizados do mesmo tenant; nunca fallback entre clientes. Campos comuns de infraestrutura não são listáveis pelo tenant sem projeção mínima autorizada.

## PostgreSQL e RLS

Habilitar RLS e FORCE ROW LEVEL SECURITY em tabelas tenant-scoped, com USING e WITH CHECK para leitura/escrita. Sem contexto válido, negar tudo. Aplicação/worker usam papéis sem superuser, BYPASSRLS, propriedade das tabelas ou permissão para alterar políticas. Papel de migration separado e indisponível aos jobs. Policies precisam incluir autorização de projeto ou a camada de aplicação deve aplicá-la sempre; RLS só de tenant não prova isolamento entre A1 e A2.

Contexto de tenant é definido pelo serviço confiável dentro da transação, com escopo LOCAL; nunca persistir em sessão pooled. Rollback/finalização encerra contexto. Exercitar conexões reutilizadas, transações abortadas e jobs alternados A/B. Uma GUC controlável por qualquer SQL não é proteção contra SQL injection: consultas parametrizadas, sem SQL da IA e sem conexão de banco na zona de execução continuam obrigatórias. Revisar SECURITY DEFINER, views, funções, search_path e acessos de manutenção. [PostgreSQL oficial](https://www.postgresql.org/docs/17/ddl-rowsecurity.html) explica exceções de owner/superuser/BYPASSRLS; adaptar à versão real.

Migração gradual: criar tenant interno somente para registros cuja origem foi comprovada; não atribuir dados ambíguos ao primeiro cliente. Inventariar órfãos, quarentena e validação antes de NOT NULL/FKs/RLS; testar restore e rollback, sem remover RLS como atalho. Índices únicos compostos e respostas genéricas evitam vazamento por erros de integridade; RLS não elimina todos os canais de inferência.

## Além do banco

Redis não oferece RLS: apenas controle/worker confiáveis acessam ACLs por serviço. Namespace de chave por tenant/project/run é organização, não autorização. Payload de fila é envelope versionado sem segredos, autenticado ou revalidado contra fonte de autoridade; consumo verifica cadeia e fencing. Cache inclui tenant, projeto, revisão e política; sem cache privado global por hash. Pub/sub e websocket verificam membership e run em cada subscrição/reconexão. Download/upload e URLs temporárias autorizam objeto exato; não aceitar prefixo livre ou listing. Logs, busca, métricas privadas, exports, backups e artefatos seguem ownership. Redação de segredo não substitui autorização de logs.

## Resultado esperado

A não consegue descobrir IDs, contagens, logs, uso, instalações ou arquivos privados de B. Acesso a A2 sem grant é negado inclusive quando A1 é permitido. Operação administrativa fica em identidade separada com auditoria; IA não recebe essa identidade. Testar detalhe, listagem, escrita, filtros, paginação, export, streaming e background jobs em ambas as direções.


## Origem desta edição

[Versão original preservada](../../99-historico/originais/output/le-fabrique-multitenant/documentacoes/tenant-isolation/README.md). Migração editorial de paths em 2026-10-06; conteúdo de engenharia continua proposto.

## Localização, relações e validação

Código da fábrica existe; não foi encontrada implementação de tenant-isolation em a2cc5e0. Nenhum teste operacional desse controle foi executado nesta migração. Consulte [ADRs](../../06-decisoes/00-README.md), [features](../../04-features/00-README.md) e [SEC-01–14](../../08-desenvolvimento/12-testes-seguranca.md). Estados/contratos citados são planejados; registrar código/revisão/checks por controle quando implementado.

## Conciliação no repo real

[AS-IS conferido](../../02-arquitetura/01-as-is.md). Este capítulo preserva integralmente a direção proposta posterior; não redefine o software entregue nem promove LF-MT a DONE.
