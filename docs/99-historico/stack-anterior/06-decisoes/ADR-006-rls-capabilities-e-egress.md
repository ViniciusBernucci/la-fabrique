# ADR-006 — rls capabilities e egress

Data: 2026-10-06 (America/Sao_Paulo). Status da arquitetura e controles: **PLANEJADO**. Implementação e eficácia: **NÃO VERIFICADAS**. Este documento especifica trabalho futuro; não comprova instalação, configuração ou execução.

## Estado da decisão

PROPOSTA DOCUMENTADA para implementação incremental. Direção solicitada pelo usuário; eficácia NÃO VERIFICADA. ADR-002 original preservado como histórico v2.1 no PDF, com topologia mantida. IDs 003–006 são propostos: conferir colisões no repositório real antes de integrar.

## Contexto

Autorização no prompt ou filtro tenant de aplicação isolado falha por bugs, replay e confused deputy.

## Decisão

Aplicação + FKs + RLS/USING/WITH CHECK, capabilities breves vinculadas ao canal/attempt, broker sem controle e egress por operação/destino efetivo.

## Alternativas e consequências

Pools transacionais e roles sem bypass; revogação/fencing; allowlist de domínio sozinha não impede exfiltração. Namespace de cache/Redis não é barreira suficiente. Prompts como barreira única, home global, credencial por máquina, shell host e fallback entre tenants são rejeitados. APIs pagas e segunda VPS não são alternativas habilitadas automaticamente.

## Verificação e reversão

Aplicar testes de [aceite](../08-desenvolvimento/12-testes-seguranca.md) e registrar código/config/revisão real. Até esse gate a implementação continua PLANEJADA. Se falhar, desabilitar recurso/provider e preservar checkpoint; rollback não remove fronteiras nem retorna a auth compartilhada. Reavaliar decisão em ADR posterior se implementação provar incompatibilidade.


## Origem desta edição

[Versão original preservada](../99-historico/originais/output/le-fabrique-multitenant/documentacoes/arquitetura/ADR-006-rls-capabilities-e-egress.md). Migração editorial de paths em 2026-10-06; conteúdo de engenharia continua proposto.
