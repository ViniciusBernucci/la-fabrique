# FAC-034 — Centro de Comando em todo o painel

Status inicial READY por especificação e autorização explícitas do usuário em 2026-10-07 (Europe/Berlin). Responsável: implementação nesta sessão, aceite pelo proprietário. Risco R1, apresentação sem mudança de API ou workflow.

## Objetivo e escopo

Aplicar o [handoff visual](../../../09-entregas/2026/evidencias/design_handoff_centro_comando/README.md) a apps/web: tokens, frame/home, login, Projetos/Operação, Tarefas, Configurações e modais. Criar núcleo SVG determinístico com animação CSS e modo de movimento reduzido. Preservar contratos de navegação, textos, aria, estado e chamadas de API. Não instalar dependências, alterar providers/credenciais, fazer push ou merge. Trabalho na developer expressamente solicitado; único writer nesta sessão.

## Critérios e baseline

Base developer e origin/main idênticas: de5dbfac0b00f3e1699b999f987856f592f65899, comparadas por git log/diff antes da edição; somente handoff não rastreado. Baseline web: 20 arquivos/53 testes PASS. Gates: npm run lint; npm run typecheck -w @le-fabrique/web; npm run test -w @le-fabrique/web; npm run build -w @le-fabrique/web; python3 scripts/validar-documentacao.py. Aceite visual e revisão independente registrados separadamente dos checks automatizados.

## Entregáveis e limites

Etapas separadas: tokens → frame/home → login/primitivos → Projetos/Operação → Tarefas → Configurações → documentação. Git index somente leitura no ambiente: tentativa de git add falhou; preparar patches por etapa e comandos de commit comentados. Sem contorno de permissões. Não executar IA pelo produto, bancos, produção ou providers. Uso/custo desta sessão não informado pelo ambiente; sem contratação ou gasto adicional autorizado.

[Entrega e resultados](../../../09-entregas/2026/2026-10-07-FAC-034-centro-comando.md). AWAITING_HUMAN após implementação/checks; DONE depende de aceite da revisão exata.
