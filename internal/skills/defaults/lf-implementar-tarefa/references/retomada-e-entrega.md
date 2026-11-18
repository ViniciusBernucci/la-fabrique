# Retomada e entrega técnica

## Retomar pelo checkpoint

1. Use o contrato da tentativa atual, a referência do checkpoint validado e a reconciliação fornecida pelo runtime. Confira a liberação do escritor anterior, o repositório, a worktree, a branch, a base e a cabeça esperadas. Uma worktree restaurada pode ter alterações pendentes legítimas.
2. Compare o checkpoint com o estado observado. Confira commits já registrados, diff, arquivos não rastreados e checks, preservando alterações autorizadas. Divergência não explicada ou operação de resultado incerto volta à Sentinela antes de qualquer repetição. Preserve também trabalho de autoria incerta e evite operações que o descartem ou alterem.
3. Retome do primeiro passo incompleto comprovado. Reutilize um commit confirmado se corresponder ao trabalho, sem criar outro para a mesma alteração. Reutilize somente evidência que ainda vale para o conteúdo testado e o ambiente requerido. Se não houver prova suficiente, repita o check aplicável quando autorizado. Informe o que recuperou, revalidou ou deixou pendente.

## Fechar a entrega

Antes de um commit atribuído ao agente, confira branch e índice, selecione caminhos explícitos e revise o diff preparado. Índice com mudanças alheias ou de origem incerta requer reconciliação. Preserve o estado e reporte a pendência. Depois do commit, confira a revisão e o conteúdo efetivamente registrado. Se um hook ou outra etapa mudar conteúdo relevante após os testes, reexecute os checks afetados ou marque-os pendentes.

Quando precisar preservar uma entrega parcial, informe arquivos alterados, pendências, estado do índice e referências ao checkpoint pelo canal previsto. A criação do snapshot, a liberação da vaga e a recuperação do processo pertencem ao runtime.

## Saída

O schema configurado para a etapa tem precedência. A tabela abaixo é um formato inicial para uso manual ou avaliação da skill. Os códigos descrevem o resultado técnico do agente. Em automação, se o formato de saída não estiver disponível ou for incompatível, relate a pendência no canal configurado.

| Campo | Conteúdo |
|---|---|
| `origem` | Identificadores recebidos de contrato, subtarefa, execução e tentativa. Inclua base, branch temporária e referência de checkpoint quando aplicável. Em correção, inclua PR original, rodada e comentários tratados. |
| `resultado` | `IMPLEMENTADA` quando os critérios atribuídos ao agente estão comprovados, os checks sob sua responsabilidade passaram ou foram tratados conforme a política da fábrica e a entrega local exigida está disponível. `PENDENTE_VALIDACAO` quando o código está disponível, mas falta uma verificação obrigatória do agente. `BLOQUEADA` quando o trabalho ou o handoff não pode avançar, incluindo falha obrigatória conhecida sem resolução ou impedimento de entregar o commit atribuído. `SEM_ALTERACAO` quando a revisão observada e a verificação aplicável comprovam que o comportamento pedido já existe. A revisão independente e a validação integrada permanecem etapas próprias. |
| `mudancas` | Resumo, caminhos relativos, documentação afetada, contratos, migrações e decisões técnicas aplicáveis. |
| `criterios` | Referência do critério e resultado `ATENDIDO`, `NAO_ATENDIDO` ou `NAO_VALIDADO`, com evidência ou motivo. Um critério que depende de verificação ainda não feita fica `NAO_VALIDADO`. |
| `checks` | Comando ou procedimento observado, revisão ou artefato exato testado, ambiente, resultado e referência de evidência sanitizada. Use `PASSOU`, `FALHOU` ou `NAO_EXECUTADO`. Quando houver falha, registre a classificação e a prova de baseline, se disponível. Identifique checks delegados ao workflow. |
| `entrega` | SHA dos commits locais confirmados e/ou referência ao diff autorizado. Se o commit ainda cabe ao runtime, indique essa etapa. Atribua à tarefa apenas o que foi comprovado. |
| `pendencias` | Causa, impacto, evidência, responsável pelo próximo passo e instrução de reprodução quando útil. Inclua os dados necessários para o revisor testar a mudança. |

Evidências devem excluir segredos, dados reais e caminhos privados do host. Preserve as referências já sanitizadas fornecidas pelo runtime.
