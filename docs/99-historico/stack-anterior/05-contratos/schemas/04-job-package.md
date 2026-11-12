> Leitura: [← Anterior](03-checkpoint.md) · [Índice didático](../../02-INDEX.md) · [Próximo assunto →](../api/00-catalogo-rotas.md)

# Contratos implementáveis v3 propostos

Data: 2026-10-06 (America/Sao_Paulo). Status da arquitetura e controles: **PLANEJADO**. Implementação e eficácia: **NÃO VERIFICADAS**. Este documento especifica trabalho futuro; não comprova instalação, configuração ou execução.

Esta revisão complementa as entidades e estados v2.1 com tenant/project scope obrigatório. [Identidade e RLS](../../03-modulos/tenant-isolation/00-README.md), [runtime](../../03-modulos/runtime/00-README.md), [tools](../../03-modulos/tool-broker/00-README.md) e [operação](../../07-operacao/02-execucao-e-recuperacao.md) definem a semântica normativa. Contratos são internos da fábrica; nenhum JSON abaixo é configuração nativa de fornecedor.

## JobPackage ilustrativo sem segredos

```json
{
  "schema_version": "3-proposta",
  "tenant_id": "fixture-A",
  "project_id": "fixture-A1",
  "run_id": "fixture-run",
  "attempt_id": "fixture-attempt",
  "provider_installation_id": "fixture-install-A",
  "revision": "REVISÃO_REAL_OBRIGATÓRIA_NA_EXECUÇÃO",
  "policy_version": "fixture-policy",
  "context_manifest_id": "fixture-manifest",
  "tool_profile": "project-developer",
  "network_profile": "deny-all",
  "deadline": "DEFINIDO_PELO_SERVIDOR",
  "fencing_token": 1
}
```

IDs são fixtures ilustrativas, não registros reais. Envelope autenticado inclui hash do manifesto e expiração concretos no código futuro. `credential_ref` fica só na resolução interna e não integra pacote da IA. Paths de host, secrets, DSN, tokens de controle e endpoints administrativos são proibidos. Hashes do contexto/base e versão da política garantem associação; contexto vindo de projeto não redefine esse envelope.

## APIs relacionadas

As rotas, autorização, idempotência e erros têm fonte canônica em [API de controle](../api/01-controle.md) e [protocolo worker](../api/02-worker.md). A origem v3 abreviava `POST /runs`; a base v2 especificava `/tickets/{id}/runs`. A conciliação provisória e a pendência de controller/OpenAPI estão no contrato de controle, sem impor alias por este documento.

## Persistência e validade relacionadas

Ownership/integridade de entidades é definido em [entidades](00-entidades.md) e [tenant isolation](../../03-modulos/tenant-isolation/00-README.md); dispatch/dedupe em [filas](../filas/00-dispatch.md); revisão e aceite em [workflow](01-workflow.md); imutabilidade/path/hash/retention em [artifact/context](02-artifact-context.md). JobPackage referencia essas autoridades, não redefine seus contratos.

## Implementado versus planejado

Nenhuma interface, migration, controller, adapter, RLS ou infra foi implementada neste pacote. Quando código existir, publicar tabela por contrato com caminho/revisão/teste/evidência; não promover todo documento para IMPLEMENTADO por concluir um módulo.


## Origem desta edição

[Versão original preservada](../../99-historico/originais/output/le-fabrique-multitenant/ESPEC-MVP.md). Reorganização editorial por seção em 2026-10-06; APIs/persistência apontam fontes canônicas em vez de manter contratos duplicados. Conteúdo de engenharia continua proposto.
