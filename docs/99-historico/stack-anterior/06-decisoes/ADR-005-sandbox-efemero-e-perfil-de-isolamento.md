# ADR-005 — sandbox efemero e perfil de isolamento

Data: 2026-10-06 (America/Sao_Paulo). Status da arquitetura e controles: **PLANEJADO**. Implementação e eficácia: **NÃO VERIFICADAS**. Este documento especifica trabalho futuro; não comprova instalação, configuração ou execução.

## Estado da decisão

PROPOSTA DOCUMENTADA para implementação incremental. Direção solicitada pelo usuário; eficácia NÃO VERIFICADA. ADR-002 original preservado como histórico v2.1 no PDF, com topologia mantida. IDs 003–006 são propostos: conferir colisões no repositório real antes de integrar.

## Contexto

Código cliente e IA podem ser hostis; container compartilha kernel da VPS com o controle.

## Decisão

Attempt efêmero endurecido, deny-by-default, helper lifecycle mínimo; piloto próprio/sintético. Código arbitrário de terceiros exige avaliar perfil forte na VPS e revisão adversarial antes da admissão.

## Alternativas e consequências

Não prometer isolamento absoluto contra root/kernel. MicroVM/gVisor têm overhead/compatibilidade a testar. VPS dedicada futura depende de decisão, não é requisito imediato. Prompts como barreira única, home global, credencial por máquina, shell host e fallback entre tenants são rejeitados. APIs pagas e segunda VPS não são alternativas habilitadas automaticamente.

## Verificação e reversão

Aplicar testes de [aceite](../08-desenvolvimento/12-testes-seguranca.md) e registrar código/config/revisão real. Até esse gate a implementação continua PLANEJADA. Se falhar, desabilitar recurso/provider e preservar checkpoint; rollback não remove fronteiras nem retorna a auth compartilhada. Reavaliar decisão em ADR posterior se implementação provar incompatibilidade.


## Origem desta edição

[Versão original preservada](../99-historico/originais/output/le-fabrique-multitenant/documentacoes/arquitetura/ADR-005-sandbox-efemero-e-perfil-de-isolamento.md). Migração editorial de paths em 2026-10-06; conteúdo de engenharia continua proposto.
