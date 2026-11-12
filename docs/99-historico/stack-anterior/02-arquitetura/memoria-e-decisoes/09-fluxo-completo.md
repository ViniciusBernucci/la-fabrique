> Leitura: [← Anterior](08-aprendizado.md) · [Índice didático](../../02-INDEX.md)

# Fluxo completo: como os componentes trabalham juntos

## Tarefa, execução e tentativa

**Tarefa** descreve o trabalho. **Run** acompanha uma execução. **Attempt** identifica uma tentativa concreta, que pode parar e ser retomada. Terminar uma tentativa não significa aceitar a tarefa. O fluxo lógico também pode ter etapas de revisão e documentação que não correspondem a estados persistidos iguais em todas as entidades.

## Um exemplo do começo ao aceite

1. Você descreve uma melhoria. O Tech Lead esclarece requisitos e propõe tarefa verificável.
2. Memória e índice mostram regras e áreas possivelmente afetadas.
3. O Tech Lead define agentes/modelos; você pode ajustar no controle de tarefas.
4. Gates validam escopo, revisão, configuração e disponibilidade antes de READY.
5. Worker prepara ambiente, baseline e pacote de contexto.
6. Developer implementa; ferramentas executam checks e registram resultados reais.
7. Reviewer independente analisa a revisão exata; QA verifica comportamento com evidências.
8. Correções respeitam limites; documentação e propostas de memória são preparadas.
9. Você recebe resultado, riscos e fontes para aceite.
10. Seu aceite publica conhecimento vinculado à revisão. Merge e deploy continuam ações distintas.

Baseline é o resultado antes da mudança. Ele ajuda a distinguir falha antiga de regressão introduzida. QA é verificação funcional; code review analisa código. Um teste passar não prova sozinho que todo requisito foi atendido.

## Quando a execução para

Se o provider esgota cota, verificar parada, preservar snapshot/checkpoint e selecionar continuidade elegível conforme configuração. Sem prova de parada, novo writer fica bloqueado. Um lock da fila não comprova que o processo anterior terminou.

Se o contexto tem conflito, investigar e bloquear somente tarefas afetadas; impacto ainda incerto impede execução dependente. Se JEV falha, usar LLM elegível e depois você. Essas respostas são caminhos de exceção, não sucesso fingido.

## Como ler a matriz

A transcrição traz 145 operações, 56 exceções e 20 decisões candidatas ao JEV. A [matriz](../../08-desenvolvimento/planos-memoria-e-decisoes/10-matriz-fluxo.md) preserva cada item, identifica seu plano e separa próximo escopo de evolução futura. “Mapeado” significa rastreável; não significa implementado ou testado.

IDs FAC ilustrativos da transcrição não serão reutilizados: vários já têm outro significado no backlog real. O Tech Lead criará os tickets depois da avaliação destes planos. O próximo passo é avaliar contratos e propostas técnicas, não executar 201 itens por autorização implícita.

Leia o [plano](../../08-desenvolvimento/planos-memoria-e-decisoes/09-fluxo-completo.md).
