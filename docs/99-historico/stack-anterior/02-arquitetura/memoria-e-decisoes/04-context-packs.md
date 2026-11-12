> Leitura: [← Anterior](03-indexacao-e-grafo.md) · [Índice didático](../../02-INDEX.md) · [Próximo →](05-decision-engine.md)

# Context Packs: como entregar conhecimento a uma tarefa

## Pacote e manifesto

Um **Context Pack** é um pacote selecionado para uma tarefa, papel e revisão. O **manifesto** é seu recibo: quais fontes entraram, hashes, motivos, omissões e versões. Não é suficiente ter uma lista de nomes; o conteúdo pode mudar mantendo o mesmo nome.

O Context Builder atual ordena e lê arquivos indicados e calcula hashes. A evolução proposta adiciona descoberta, obrigatório versus opcional, expansão e vínculo com memória. O manifesto atual marca qualquer omissão com `truncated`; a evolução precisa distinguir omissão de opcional e ausência de requisito obrigatório.

## Exemplo passo a passo

Uma tarefa altera validação de um campo. O pacote inclui políticas obrigatórias, objetivo, critérios, contrato do campo, implementação e testes associados. Documentos de outra área não entram apenas porque existem. Se a validação produz um evento, o grafo acrescenta o contrato e consumidor relacionados.

Antes de executar, o validador confere todas as fontes obrigatórias declaradas. “Todas” significa as conhecidas e explicitamente catalogadas para aquele escopo; nenhum algoritmo prova conhecer toda regra ainda não documentada.

Durante a execução, a IA pode pedir mais informação. **Expansão progressiva** é adicionar fontes sob demanda, mantendo a mesma revisão e registrando o novo recibo. A fábrica não presume controlar toda a janela interna do cliente oficial. Só afirma o que entrega e observa.

## Contexto por papel e continuidade

Developer recebe fontes para mudar código. Reviewer recebe critérios, revisão exata, diff e evidências em uma sessão independente. Papéis podem usar o mesmo fornecedor, mas têm responsabilidades diferentes.

No handoff, o pacote acompanha um **checkpoint**, registro de progresso, e um **snapshot**, cópia verificável das alterações. Isso permite continuar sem transferir conversa privada ou credenciais. Antes do novo writer, é obrigatória a confirmação externa de parada do anterior.

## Cache e falhas

**Cache** reutiliza trabalho já calculado. Sua chave precisa incluir revisão, fontes, tarefa, papel, política e sobreposição. Se algum desses dados mudar, uma seleção antiga pode estar errada. Cache interno não é memória oficial nem cache do fornecedor.

Se fontes obrigatórias não couberem, não são cortadas em silêncio: dividir tarefa, expandir de maneira controlada ou pausar. Se uma fonte contém segredo, excluir o envio e explicar a lacuna. Reduzir texto só é melhoria quando não piora cobertura e retrabalho.

Leia o [plano](../../08-desenvolvimento/planos-memoria-e-decisoes/04-context-packs.md).
