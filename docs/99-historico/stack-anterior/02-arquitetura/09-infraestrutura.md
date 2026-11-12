> Leitura: [← Anterior](08-seguranca.md) · [Índice didático](../02-INDEX.md) · [Próximo →](10-contexto-alvo.md)

> Integração DOC-MV-001: texto de direção/planejamento do pacote. O [AS-IS atual](01-as-is.md) prevalece para implementação. Não confundir proposta com controle instalado.

# Infraestrutura e implantação

Revisão documental: 2026-10-06 (America/Sao_Paulo). Arquitetura **PLANEJADA**; implementação e eficácia **NÃO VERIFICADAS** na origem do pacote.

Uma VPS Linux comporta controle, banco/fila, worker, runtimes e sandboxes separados logicamente. HTTPS público, SSH administrativo restrito, serviços internos privados. Inferência externa; GPU não é requisito. Um executor inicial; ampliar somente após medição.

O [dimensionamento canônico](../07-operacao/infraestrutura/00-dimensionamento-vps.md) preserva perfis e envelopes propostos; os valores não são benchmark da stack atual. [Deploy](../07-operacao/01-deploy.md) e [backup](../07-operacao/05-backup.md) descrevem operação futura. VPS contratada, OS/kernel e acesso permanecem pendentes. [ADR-002](../06-decisoes/ADR-002-vps-unica.md) registra a decisão de topologia.
