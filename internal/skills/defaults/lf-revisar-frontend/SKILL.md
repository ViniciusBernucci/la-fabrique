---
name: lf-revisar-frontend
description: Revisa e revalida tarefas Frontend da Le Fabrique em navegador isolado pelo Chrome DevTools MCP. Use para reproduzir os critérios de aceite na aplicação em execução, registrar evidências e tratar achados em etapas autorizadas pelo workflow.
---

# Revisar Frontend na Le Fabrique

Execute o review visual obrigatório de toda subtarefa `FRONTEND`, inclusive correções de PR, com o servidor oficial **Chrome DevTools MCP** (`chrome-devtools-mcp`). Use exclusivamente o navegador isolado atribuído à execução. Relate em português, preservando os identificadores do contrato.

## Confirme a execução

- Leia o contrato imutável, os critérios de aceite, a revisão candidata, as evidências do implementador e os pré-requisitos do review. A avaliação inicial roda em sessão independente. Confira no despacho a versão da configuração visual validada, a prontidão do ambiente, a revisão servida, a URL privada e a identidade da sessão isolada. Se faltar comprovação ou houver divergência, devolva a pendência à Sentinela.
- Use o servidor MCP configurado pelo runtime e somente as ferramentas e destinos autorizados. A [referência de Chrome DevTools MCP](references/chrome-devtools-mcp.md) descreve o vínculo com a instância isolada, as ferramentas de review e o registro de evidências. MCP ausente, ambiente indisponível ou ferramenta essencial bloqueada tornam pendentes os cenários afetados.

## Avalie no navegador

1. Traduza cada critério afetado em um cenário com pré-condição, ação e resultado esperado. Use dados e papéis de teste provisionados. Comece pelo fluxo principal e cubra erros, estados vazios, carregamento, permissões, responsividade, foco e teclado quando o critério ou o diff os afetar.
2. Abra a rota autorizada via MCP. Interaja pelos controles reais da interface e observe a resposta renderizada. Use snapshot para localizar elementos, capturas para evidência visual e console ou rede para diagnosticar o que ocorreu. Se o destino observado sair do escopo autorizado, interrompa a interação e registre o bloqueio. Para um teste de teclado, use eventos de teclado e confirme o foco. Para um critério responsivo, registre o viewport observado.
3. Vincule o resultado de cada cenário ao critério, à revisão servida e à configuração usada. Para um achado, registre passos, esperado, observado, impacto e evidência. Distinga defeito reproduzido, falha de infraestrutura e causa ainda indeterminada. Se houver baseline autorizado comparável, use-o para separar regressão de falha preexistente comprovada.
4. Na avaliação inicial, trabalhe em leitura do código e devolva achados à Sentinela. Corrija somente em etapa de correção autorizada, na worktree atribuída, com base nos IDs dos achados e na revisão esperada. Faça a menor mudança aplicável e rode os testes afetados. Se a solução exigir decisão de produto, alteração de contrato ou escopo adicional, relate o bloqueio.
5. Na revalidação, confirme a nova revisão servida, reproduza os cenários dos achados corrigidos e cubra as regressões plausíveis no raio de impacto. Reaproveite evidências anteriores apenas se continuarem válidas para o comportamento, revisão e ambiente aplicáveis. Se faltar essa prova, repita a verificação. A Sentinela controla o limite de rodadas.

## Devolva o resultado

Use o schema do workflow. Para uso manual ou avaliação, siga [resultado do review](references/resultado-do-review.md). Aprovação técnica exige interação observada no navegador e evidência suficiente para os critérios obrigatórios. Leitura do código, testes do implementador e capturas isoladas são evidências complementares.

Entregue achados, cenários bloqueados e referências de artefatos pelo canal da fábrica. A Sentinela decide a transição e coordena integração e limpeza dos recursos da execução.
