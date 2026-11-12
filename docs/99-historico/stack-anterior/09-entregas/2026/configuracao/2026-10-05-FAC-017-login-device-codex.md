# FAC-017 — Autorização Codex em nova aba

Estado AWAITING_HUMAN. Baseline c2e3095, commit funcional b4411fe em developer conforme autorização do responsável. Nenhum provider utilizado para inferência; somente diagnóstico de login/status. Não habilita API keys, extras, writer ou migrations.

## Diagnóstico e correção

O login real iniciado pelo usuário ficou RUNNING e o endpoint de challenge retornou 404. Conta codex-default permanecia AUTH_REQUIRED. Um diagnóstico separado em identidade temporária privada observou saída ANSI do Codex 0.159.2; a URL continha controle de cor e o extractor não identificava o código. Não foram publicados códigos, tokens nem saída bruta nos logs/evidências.

O extractor usa stripVTControlCharacters antes de reconhecer URL/código, preserva o código exato e aguarda linha completa no formato de terminal para não publicar fragmento de stdout. Compatibilidade com fixtures inline preservada. O contrato de desafio Codex passa a aceitar letras maiúsculas/minúsculas mantendo os mesmos limites de tamanho, caracteres alfanuméricos e hífens. O formato real foi diagnosticado como saída colorida; a ampliação de caso é coberta por fixtures, sem presumir que a conta real forneceu letras minúsculas. Contrato GitHub não foi alterado. A validação de host oficial HTTPS continua obrigatória; desafio permanece privado e efêmero no Redis.

O clique em Conectar assinatura Codex reserva uma aba about:blank durante o gesto do usuário, remove opener e mostra texto de espera. Ao obter o desafio, navega uma única vez para o site oficial. A aba nunca recebe o código, token ou parâmetros de autenticação da fábrica. Se foi bloqueada/fechada, o link Abrir login oficial do Codex em nova aba fica disponível no modal junto ao código. Se o usuário já navegou a aba para outro lugar, ela não é sobrescrita/fechada. Erro ao iniciar fecha somente a aba de espera. A tela distingue iniciar/aguardar desafio, aguardar usuário, falhar/expirar e confirmação pelo cliente oficial.

Abrir a aba não confirma autenticação. Sucesso exige sessão COMPLETED e conta AVAILABLE por status do cliente oficial na identidade dessa instalação; o catálogo de modelos continua sendo verificação posterior separada (automática para login iniciado na tela). Não houve inferência para confirmar assinatura.

## Teste humano

1. Recarregue a página e abra a conta Codex já salva/habilitada.
2. Clique Conectar assinatura Codex. A sessão anterior não foi concluída; inicie novo pedido.
3. Aguarde a URL oficial na aba e use o código apresentado no modal. Caso a aba seja bloqueada, use o link do modal.
4. Autorize sua conta ChatGPT no site oficial. Device code login precisa estar permitido nas configurações de segurança/workspace.
5. Volte ao modal e aguarde Assinatura conectada, AVAILABLE e os modelos importados. Se a página foi recarregada durante autorização, use Verificar conta e modelos para atualizar catálogo.
6. Se falhar/expirar, a tela deve informar isso e permitir nova tentativa. Não interpretar uma sessão RUNNING como login bem-sucedido.

[Documentação oficial do fluxo device code](https://learn.chatgpt.com/docs/auth). O cliente executa na VPS; a autorização do usuário ocorre no seu navegador.

## Evidências e limites

- Histórico real: sessão 8546306c-52da-4d1c-9ac2-8eed52e84349 RUNNING sem desafio; posteriormente FAILED/AUTH_REQUIRED, não autenticada.
- Cliente real em identidade temporária isolada: após correção, runCodexDeviceLogin publicou desafio válido, host auth.openai.com, caminho /codex/device limpo, código presente. Diagnóstico terminou sem autorização e removeu somente seu diretório temporário. Nenhum código real foi gravado neste relatório ou patch.
- Casos automatizados: ANSI/caso preservado/fragmento incompleto, contrato do código, tab/opener/link oficial, popup bloqueado/fechado, navegação externa preservada e fechamento somente da aba de espera.
- A primeira rodada detectou alteração no contrato GitHub em vez do Codex e regressão de fixture inline; corrigidas antes da revisão funcional final. Um refinamento restringiu whitespace inline para não aceitar código incompleto na linha seguinte.
- Checks: npm test passou com 547 testes (7 scripts, 46 contratos, 65 runtime, 179 API, 211 worker, 39 web); typecheck monorepo e build monorepo passaram. Build worker e lint repetidos após refinamento final também passaram; warning anterior em run-delivery.service.ts permanece.
- Não existe navegador automatizado no ambiente: popup/interação visual dependem do teste humano. Autorização da conta, catálogo autenticado e inferência permanecem NÃO VERIFICADOS enquanto não houver evidência correspondente.

## Rollback e aceite

Reverter b4411fe restaura parser/UI/contrato anteriores, sem apagar sessões privadas, catálogo ou dados. Nenhuma migration. [Diff sanitizado](../evidencias/configuracao/evidencias/FAC-017-diff.patch), sem contexto, aplicável sobre baseline apropriado com git apply --unidiff-zero. Aceite humano da revisão exata e login oficial pendentes.

Leitura final do controle: versão 19, codex-default habilitado e AUTH_REQUIRED, catálogo vazio. Portanto login da conta do usuário permanece não confirmado nesta revisão.
