# Checklist de risco Backend

Selecione os eixos afetados pelo diff e pelas regras de risco do projeto. Cada item indica o que investigar. Um achado precisa de impacto demonstrável e uma localização na revisão analisada.

| Superfície | O que verificar |
|---|---|
| Autorização e escopo de dados | Permissão na operação que lê ou escreve, ownership do objeto, escopo de tenant e perfis permitidos. Teste acesso a recurso de terceiro, mudança de papel e rotas alternativas relevantes. |
| Entradas, contratos e erros | Validação no ponto de entrada, campos graváveis, compatibilidade de payload, códigos de erro e consumidores existentes. Verifique se entradas externas alcançam consultas, comandos, URLs ou caminhos sensíveis sem controle adequado. |
| Integridade e concorrência | Fronteira da transação, efeitos antes e depois de falha parcial, locks ou controle de versão, chaves de unicidade e reprocessamento. Repita a mesma operação ou simule concorrência e observe o efeito persistido quando aplicável. |
| Migração e dados existentes | Leitura e escrita durante rollout misto, valores legados, ordem de migração e aplicação, backfill, locks e recuperação. Use dados de teste representativos e o banco autorizado. |
| Integrações e efeitos externos | Autorização, timeout, retry, deduplicação, reconciliação de resultado incerto e propagação de falhas. Use sandbox, fixture ou substituto aprovado para efeitos externos e indique qual fronteira o teste cobriu. |
| Exposição e desempenho | Dados sensíveis em resposta, log ou artefato, limites de paginação e volume, consultas repetidas e operações sem limite. Só afirme impacto de desempenho compatível com o caminho demonstrado. |
| Testes e checks de segurança | Testes pela interface pública, cenários negativos pertinentes e resultados rastreáveis à revisão candidata. Confira se o teste falha quando a proteção ou regra em análise é removida, quando houver verificação segura e viável. |

Compare base e candidata no mesmo cenário quando precisar classificar origem. Use procedimento, dados de teste e ambiente comparáveis. Sem essa evidência, informe origem indeterminada. Quando a regra do projeto exigir check de segurança ou isolamento específico, confirme o pré-requisito antes de executar.
