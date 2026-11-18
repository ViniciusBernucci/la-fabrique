# Perfis de implementação

Use o perfil da trilha atribuída e selecione os cenários que correspondem ao critério ou ao risco real da mudança.

## FRONTEND

- Exercite o comportamento público pela interface do componente ou da rota. Para mudança de integração, confira o contrato aprovado de request, response e erro. Use fixtures que representem o contrato quando o serviço real for pré-requisito apenas do review.
- Cubra estados afetados de carregamento, vazio, erro e sucesso, além de teclado, foco, acessibilidade ou responsividade quando a interação exigir. Preserve padrões do produto e trate feedback de falha de forma verificável.
- No handoff, informe rota, papel de teste, pré-condições e passos para reproduzir o critério. Aponte cenários que usam o serviço real. A revisão independente em Chromium isolado via MCP segue como etapa do workflow.

## BACKEND

- Exercite a interface pública pertinente, como endpoint, comando, evento ou serviço exposto. Cubra autorização, isolamento de dados, entradas inválidas, idempotência, concorrência e falha parcial quando o diff tocar esses comportamentos. Para reprocessamento, verifique o efeito persistido após repetir a mesma operação.
- Em mudanças de dados, teste dados existentes e sequência de migração no ambiente autorizado. Registre compatibilidade, backfill e como recuperar uma falha quando aplicável. Use serviços e dados de teste segregados.
- No handoff, indique contratos alterados, riscos observados e evidências dos caminhos de segurança afetados. Sinalize gatilhos de risco descobertos no diff para a Sentinela avaliar. O workflow exige review de IA de Backend em alto risco.
