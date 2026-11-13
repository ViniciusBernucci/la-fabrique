> Leitura: [← Anterior](01-visao-integrada.md) · [Índice didático](../../02-INDEX.md) · [Próximo →](03-indexacao-e-grafo.md)

# Memória persistente: o que a fábrica precisa lembrar

## Problema e vocabulário

Uma sessão de IA pode acabar, mas uma regra do software precisa continuar disponível. **Memória persistente** é conhecimento conservado fora da sessão, com fontes identificáveis. **Contexto** é a seleção desse conhecimento entregue a um agente para uma tarefa. Ter memória não significa enviar tudo a cada execução.

**Git** guarda versões de arquivos. Uma **branch** aponta para uma linha de trabalho; um **commit** registra uma versão, identificada por SHA. Uma **worktree** é uma pasta de trabalho associada ao Git. Um **hash** identifica o conteúdo de bytes; ele não comprova que uma afirmação é correta. PostgreSQL guarda registros e permite pesquisá-los. Um índice é uma estrutura de consulta derivada das fontes, que deve poder ser reconstruída.

## As seis categorias confirmadas

| Categoria | Conteúdo | Exemplo didático |
|---|---|---|
| Institucional | Políticas e restrições | Um writer por workspace |
| Negócio | Comportamento esperado | Alteração posterior invalida aceite |
| Relações | Ligações entre componentes e regras | Checkpoint relacionado à recuperação |
| Decisões | Escolhas e justificativas | Stack TypeScript aceita |
| Operacional | O que ocorreu nas execuções | Um check falhou em certa revisão |
| Trabalho | Seleção e progresso de uma tarefa | Pacote de contexto daquela tentativa |

Uma categoria descreve a finalidade; não exige seis bancos ou serviços. Uma anotação operacional não vira regra de negócio automaticamente. As conversas ficam fora da memória nesta etapa, por decisão do responsável; a transcrição foi usada apenas para planejar.

## Funcionamento passo a passo proposto

1. O sistema identifica uma revisão e as fontes autorizadas.
2. O worker lê os arquivos sem incluir segredos, registra hashes e prepara índices.
3. Uma informação extraída recebe origem e natureza: observação do código ou proposta de regra.
4. Uma IA pode explicar a proposta e suas evidências. O responsável decide se a aceita.
5. O aceite registra exatamente a revisão e o conteúdo aprovado.
6. Consultas futuras mostram se a regra aceita está incorporada ao código usado pela tarefa.

Exemplo: você aceita uma regra nova em uma branch, antes do merge. A memória registra “aceita; ainda não incorporada à revisão de main”. Uma tarefa na main não pode receber a descrição como se aquele código já existisse. Se isso mudar materialmente a tarefa, há uma divergência a resolver.

## Estados que precisam permanecer separados

**Aceita** indica autoridade humana sobre uma regra. **Incorporada** indica presença numa revisão de código. **Observada** indica leitura do código, que pode conter um bug. **Conflitante** indica afirmações incompatíveis ainda sem resolução. Separar esses estados permite entender por que uma regra aprovada pode existir antes do código correspondente.

Uma branch consulta a base mais sua própria **sobreposição**: alterações locais identificadas por revisão e hash do patch. Patch é o conjunto de alterações ainda separado da base. Isso evita misturar trabalho inacabado de outra branch.

## Falhas e decisões

Quando duas fontes discordam, a fábrica apresenta ambas; não escolhe a verdade por votação de modelos. Bloqueia tarefas afetadas. Se o impacto é desconhecido, investiga e pausa a execução caso a dúvida persista. Não bloqueia automaticamente todo o sistema.

Guardar tudo apenas em conversas seria simples no início, mas prejudicaria proveniência e portabilidade. Guardar conteúdo oficial apenas em banco facilitaria edição, mas criaria outra fonte de verdade. A escolha confirmada é Git para fontes versionadas e PostgreSQL para índices e registros.

Hoje o Context Builder já registra arquivos e hashes, mas esse ciclo de memória estruturada ainda é proposto. Leia o [plano correspondente](../../08-desenvolvimento/planos-memoria-e-decisoes/02-memoria-persistente.md) e os [contratos comuns](../../05-contratos/memoria-e-decisoes/00-contratos-propostos.md).
