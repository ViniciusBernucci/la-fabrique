---
name: lf-revisar-backend
description: Revisa tarefas Backend de alto risco despachadas pela Le Fabrique. Use para avaliar o diff e os critérios de aceite em sessão independente, corrigir achados autorizados e revalidar com testes reais na revisão candidata.
---

# Revisar Backend na Le Fabrique

Revise a subtarefa `BACKEND` classificada pela fábrica como alto risco no diff efetivo, inclusive quando ela pertence a uma correção de PR. Use o modo fornecido no despacho. Relate em português e preserve os identificadores do contrato.

## Confira a entrada

- Leia o contrato imutável, a classificação de risco efetiva, os critérios de aceite, os checks exigidos, a base, a revisão candidata e as evidências de implementação. Confirme repositório, worktree e conteúdo exato do diff. Se faltar uma referência essencial ou a revisão divergir, devolva a pendência à Sentinela.
- O review por IA é reservado a alto risco. Encaminhe à Sentinela despachos incompatíveis ou novos gatilhos de risco descobertos no diff. Os testes reais e checks determinísticos aplicáveis continuam obrigatórios em todas as classificações.

## Avaliação independente

1. Examine em sessão independente, com leitura do código e dos artefatos. Compare a base com a candidata fornecidas. Rastreie os caminhos alterados desde a entrada até a autorização, a regra de negócio e o efeito observável. Use o [checklist de risco](references/checklist-risco.md) nas superfícies afetadas.
2. Compare o comportamento com os critérios e contratos aprovados. Confirme as proteções efetivas em middleware, serviço, persistência ou fronteira externa antes de reportar um defeito. Para cada achado, registre condição reproduzível, resultado esperado e observado ou caminho estático que demonstra a falha, impacto e localização. Suspeita sem prova fica como pendência de verificação.
3. Verifique os testes e checks de segurança aplicáveis na revisão candidata. Use resultados rastreáveis do runtime ou execute as verificações atribuídas ao revisor no ambiente de teste autorizado. Associe os resultados aos critérios e à revisão testada. Check obrigatório ausente ou inconclusivo fica pendente. Uma falha preexistente requer evidência de baseline comparável. Registre separadamente a regressão, a falha preexistente comprovada e a causa indeterminada. A política da fábrica determina como cada falha afeta o gate.
4. Classifique a severidade pelo impacto demonstrado e pela política do workflow. Achados críticos e relevantes bloqueiam conforme essa política. Sugestões de baixa severidade ficam no relatório. Devolva os achados à Sentinela sem alterar código nesta etapa.

## Correção autorizada

Use somente os IDs de achados do despacho, na worktree e revisão atribuídas. Reconfirme que o achado permanece aplicável e faça a menor mudança capaz de resolvê-lo. Preserve os contratos aprovados e as alterações de terceiros. Acrescente ou ajuste o teste de regressão pertinente e rode as verificações afetadas. Se a solução exigir nova decisão, alteração de contrato ou ampliação de escopo, devolva a pendência. Quando o despacho atribuir o commit ao agente, confira branch e índice e inclua somente os arquivos da correção.

## Revalidação

Confirme a revisão resultante, vincule o diff de correção ao relatório anterior e preserve os IDs. Verifique se cada achado foi resolvido, persiste ou ficou bloqueado. Reexecute os checks afetados que cabem ao revisor e confira regressões plausíveis na interface e nos efeitos atingidos. Evidência anterior só vale para o conteúdo e ambiente aplicáveis. Encaminhe resultado e regressões à Sentinela, que controla as rodadas e o gate.

## Entrega

Use o schema do workflow. Para uso manual ou avaliação, siga o [resultado do review](references/resultado-do-review.md). Registre as limitações de evidência e os checks realmente executados. A Sentinela controla status, publicação e integração. A skill entrega resultados pelo canal da fábrica.
