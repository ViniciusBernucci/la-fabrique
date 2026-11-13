> Leitura: [← Anterior](06-usage-dashboard.md) · [Índice didático](../02-INDEX.md) · [Próximo assunto →](../05-contratos/00-README.md)

> Integração DOC-MV-001: texto de direção/planejamento do pacote. O [AS-IS atual](../02-arquitetura/01-as-is.md) prevalece para implementação. Não confundir proposta com controle instalado.

# Isolar tenants e projetos

Revisão documental: 2026-10-06 (America/Sao_Paulo). Arquitetura **PLANEJADA**; implementação e eficácia **NÃO VERIFICADAS** na origem do pacote.

## Comportamento e usuário

Principal/job acessa somente recursos autorizados e tools não alcançam controle/credenciais. Usuário: responsável autorizado do projeto; administração tem identidade separada.

## Regras e critérios de aceite

SEC-01–14 aplicáveis; casos A/B e A1/A2; perfil hostil tem gate próprio. Todos são critérios planejados; não há evidência de execução no espelho.

## Módulos e contratos

[Tenant isolation](../03-modulos/tenant-isolation/00-README.md); [Provider Runtime](../03-modulos/runtime/00-README.md); [Sandbox](../03-modulos/sandbox/00-README.md); [Tool Broker](../03-modulos/tool-broker/00-README.md). Contratos detalhados estão no [catálogo](../05-contratos/00-README.md); fluxos no [capítulo de fluxos](../02-arquitetura/05-fluxos.md).

## Rastreabilidade e limites

Backlog relacionado: LF-MT-01–08/10/11. IDs originais preservados; estes arquivos não criam novos FAC nem afirmam entrega. Release/código/revisão/testes: não disponíveis. Implementação deverá ligar registro de entrega e evidências reais.
