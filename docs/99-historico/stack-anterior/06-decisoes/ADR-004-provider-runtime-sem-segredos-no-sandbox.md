# ADR-004 — provider runtime sem segredos no sandbox

Data: 2026-10-06 (America/Sao_Paulo). Status da arquitetura e controles: **PLANEJADO**. Implementação e eficácia: **NÃO VERIFICADAS**. Este documento especifica trabalho futuro; não comprova instalação, configuração ou execução.

## Estado da decisão

PROPOSTA DOCUMENTADA para implementação incremental. Direção solicitada pelo usuário; eficácia NÃO VERIFICADA. ADR-002 original preservado como histórico v2.1 no PDF, com topologia mantida. IDs 003–006 são propostos: conferir colisões no repositório real antes de integrar.

## Contexto

Cliente oficial autenticado pode executar ferramentas locais; HOME separado e JSONL não provam separação.

## Decisão

Runtime isolado por tenant/provider com sessão por attempt; auth oficial apenas nesse runtime, ferramentas do modelo exclusivamente no broker/sandbox. Cliente incompatível permanece DISABLED/UNSUPPORTED_ISOLATION.

## Alternativas e consequências

Compatibilidade é gate técnico com evidência por versão. Sem workaround por token OAuth/API ou auth montada. Alternativa de CLI dentro do sandbox com auth foi rejeitada. Prompts como barreira única, home global, credencial por máquina, shell host e fallback entre tenants são rejeitados. APIs pagas e segunda VPS não são alternativas habilitadas automaticamente.

## Verificação e reversão

Aplicar testes de [aceite](../08-desenvolvimento/12-testes-seguranca.md) e registrar código/config/revisão real. Até esse gate a implementação continua PLANEJADA. Se falhar, desabilitar recurso/provider e preservar checkpoint; rollback não remove fronteiras nem retorna a auth compartilhada. Reavaliar decisão em ADR posterior se implementação provar incompatibilidade.


## Origem desta edição

[Versão original preservada](../99-historico/originais/output/le-fabrique-multitenant/documentacoes/arquitetura/ADR-004-provider-runtime-sem-segredos-no-sandbox.md). Migração editorial de paths em 2026-10-06; conteúdo de engenharia continua proposto.
