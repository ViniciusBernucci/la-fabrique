> Nota editorial DOC-MV-001: colisão com ADR-003-stack-typescript ACEITO no repo. Este é o ADR-003 PROPOSTO do pacote; número/título/status original preservados. Não substitui a decisão de stack vigente.

# ADR-003 — multi tenant e stack

Data: 2026-10-06 (America/Sao_Paulo). Status da arquitetura e controles: **PLANEJADO**. Implementação e eficácia: **NÃO VERIFICADAS**. Este documento especifica trabalho futuro; não comprova instalação, configuração ou execução.

## Estado da decisão

PROPOSTA DOCUMENTADA para implementação incremental. Direção solicitada pelo usuário; eficácia NÃO VERIFICADA. ADR-002 original preservado como histórico v2.1 no PDF, com topologia mantida. IDs 003–006 são propostos: conferir colisões no repositório real antes de integrar.

## Contexto

PDF v2.1 descreve uso pessoal e Laravel/Angular; pedido atual define multi-tenant e React/NestJS.

## Decisão

Preservar React + NestJS + worker Node/TypeScript, PostgreSQL/Redis e VPS única; introduzir cadeia tenant/project/run/installation/credential obrigatória. ADR-002 segue quanto à VPS; escopo pessoal exclusivo e stack do PDF são substituídos nesta proposta.

## Alternativas e consequências

Mais controles de autorização, dados e testes; prontidão comercial não é inferida. Não migrar código inexistente nem mudar stack do piloto. Prompts como barreira única, home global, credencial por máquina, shell host e fallback entre tenants são rejeitados. APIs pagas e segunda VPS não são alternativas habilitadas automaticamente.

## Verificação e reversão

Aplicar testes de [aceite](../08-desenvolvimento/12-testes-seguranca.md) e registrar código/config/revisão real. Até esse gate a implementação continua PLANEJADA. Se falhar, desabilitar recurso/provider e preservar checkpoint; rollback não remove fronteiras nem retorna a auth compartilhada. Reavaliar decisão em ADR posterior se implementação provar incompatibilidade.


## Origem desta edição

[Versão original preservada](../99-historico/originais/output/le-fabrique-multitenant/documentacoes/arquitetura/ADR-003-multi-tenant-e-stack.md). Migração editorial de paths em 2026-10-06; conteúdo de engenharia continua proposto.
