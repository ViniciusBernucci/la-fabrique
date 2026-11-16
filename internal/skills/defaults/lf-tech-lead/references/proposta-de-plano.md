# Entrada e saída do planejamento

Use esta referência para conferir os insumos e estruturar a proposta. As chaves e os códigos abaixo formam um formato inicial para uso manual ou em testes da skill. Se o workflow fornecer um schema, use-o e mapeie os conceitos desta referência para os campos aceitos. Em execução automatizada, quando o formato de saída estiver ausente ou for incompatível, comunique a pendência pelo canal de resultado configurado antes de tratar a proposta como consumível.

## Entradas

| Comum aos dois modos | Feature | Correção de PR |
|---|---|---|
| Contrato imutável, identidade da Issue, revisão autorizada do repositório, objetivo, critérios e regras aplicáveis. Use as referências do contrato, inclusive configuração e política de risco. | Revisão do PRD validado, evidência da liberação para planejamento e escopo aprovado. | Liberação humana registrada, rodada encerrada e comentários acionáveis com IDs, PR e branch originais, cabeça reconciliada e referências ao plano anterior quando disponíveis. |

As evidências registram o que foi observado. Requisitos, comentários e arquivos sugeridos são insumos. Ausência de acesso ao repositório, liberação não comprovada, revisão incompatível ou falta de uma regra essencial para distinguir resultados aceitáveis são pendências explícitas. Se um plano anterior estiver indisponível, só proponha o trecho da correção que puder ser delimitado com critérios, comentários e código verificáveis.

## Campos da proposta

| Campo | Conteúdo |
|---|---|
| `modo` | `FEATURE` ou `CORRECAO_PR`, conforme a origem registrada no contrato. |
| `resultado` | `PROPOSTA` quando há trabalho planejado e as condições essenciais foram verificadas. `BLOQUEADA` quando falta condição indispensável. `SEM_TRABALHO` quando a evidência da revisão mostra que todos os itens de correção já foram atendidos. O resultado é da análise, sem transição de Issue. |
| `origem` | Objeto com `contrato_ref`, `issue_ref`, `revisao_inspecionada` e `prd_ref` quando aplicável. Em correção, incluir `pr_ref`, `rodada_ref`, `plano_anterior_ref` se disponível e `cabeca_reconciliada`. Repetir identificadores conforme recebidos. Omitir referências ausentes e explicar em bloqueios se forem indispensáveis. |
| `objetivo` e `fora_do_escopo` | Resultado esperado e limites do trabalho. Na correção, delimitar os efeitos dos comentários em relação à feature existente. |
| `tarefas` | Subtarefas candidatas para validação. Deixe vazia em `BLOQUEADA` e `SEM_TRABALHO`. Em `BLOQUEADA`, detalhe o trabalho independente em `rascunho`, sem disponibilizá-lo para despacho. |
| `ordem_sugerida` | Grupos ordenados de referências de `tarefas`. Cada grupo indica candidatas a iniciar implementação em paralelo, sujeito às vagas. Uma dependência apenas de review não adia a implementação. Tarefas com conflito de escrita precisam estar em grupos distintos ou ter restrição de execução explícita. |
| `comentarios` | Em correção, um registro por comentário acionável. Indique `ref`, `tratamento` (`GERA_TAREFA`, `JA_ATENDIDO` ou `PENDENTE`), `evidencia` e `tarefas` referenciadas. Use lista vazia para feature. |
| `hipoteses` e `bloqueios` | Hipóteses e sua base. Cada bloqueio indica a informação ou ação necessária, o que foi consultado e o efeito no plano. |
| `rascunho` | Trabalho independente que pode ser especificado mesmo com a proposta bloqueada. Use lista vazia quando inaplicável. |

### Campos de cada subtarefa

- `ref` local, `trilha`, `objetivo`, `escopo` e `fora_do_escopo`. A referência local só identifica a tarefa dentro da proposta.
- `criterios`, com referência ao critério ou requisito de origem e resultado observável. Se o critério técnico for derivado, identifique a origem e o vínculo com o requisito aprovado.
- `dependencias`, com `ref`, `tipo` (`TAREFA_INTERNA` ou `MERGE_EXTERNO`), `exigida_em` (`IMPLEMENTACAO` ou `REVIEW`) e `entrega_necessaria`. Dependência interna aponta para subtarefa da proposta. Dependência de merge aponta para vínculo externo já conhecido. Descreva outros pré-requisitos de execução em `pre_requisitos`.
- `contratos_afetados`, `caminhos_previstos` e `recursos_compartilhados`. Identifique caminhos relativos ao repositório com evidência ou como prováveis. Para recursos, informe identidade, acesso de leitura ou escrita e restrição de concorrência quando houver. Os caminhos propostos orientam o despacho e precisam caber no escopo autorizado.
- `risco`, com `nivel` (`BAIXO`, `MEDIO` ou `ALTO`), justificativa e gatilhos relevantes. Avalie autenticação e autorização, dados sensíveis, alterações em dados existentes, migrações, contratos públicos, integrações, concorrência e reversibilidade quando pertinentes.
- `testes`, com critério, cenário, interface, resultado esperado e referência existente quando localizada. Indique quando criar ou ajustar um teste. `revisoes_previstas` registra navegador MCP para Frontend e revisão de IA para Backend de alto risco. Em `documentacao`, indique quais documentos ou contratos vivos são afetados.
- `comentarios`, com referências da rodada que justificam a tarefa. Em feature, use lista vazia.

## Conferência antes do envio

Confirme que as referências internas existem e são únicas, que o grafo é acíclico e que a ordem respeita as dependências por etapa e as reservas de recursos. Confira se cada subtarefa tem pelo menos um critério e um teste aplicável, se toda correção tem comentários de origem e se um comentário tratado como atendido tem evidência na revisão inspecionada. Verifique que o resultado não traz alterações no contrato, propostas de ampliação de permissões, credenciais ou afirmações de execução de testes que ainda são planejados.
