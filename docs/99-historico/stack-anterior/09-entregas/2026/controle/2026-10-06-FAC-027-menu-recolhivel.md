# FAC-027 — Menu lateral recolhível
2026-10-06. IMPLEMENTADO / AWAITING_HUMAN. Baseline bb0ad15; revisão de código `b1d794035342128b9ed832d5fe9b74c887eb7075`. [Ticket READY](../../../08-desenvolvimento/tickets/controle/tickets/FAC-027.md).
Pedido recebido durante FAC-026, preservado como incremento separado na mesma branch/worktree exclusiva. Mesmo writer/Codex da sessão, sem subagentes/handoff/provider, modelo/cota/custo não comprovados. Merge/limpeza autorizados anteriormente; sem push/deploy.

## Funcionamento
Botão sob a marca recolhe/expande sidebar: nomes/área workspace ficam ocultos, ícones permanecem. aria-label/title identifica botão e entradas; aria-expanded representa estado. Width/margem desktop usam variável comum: expandido 234px, 210px abaixo de 1450px; recolhido 66px. Conteúdo central usa espaço liberado, template/topbar persistem.

DashboardLayout mantém estado ao trocar home/login/admin. Preferência booleana salva no localStorage em la-fabrique.sidebar-collapsed; reload desktop restaura escolha. Storage bloqueado é tratado sem erro: escolha segue em memória do template. Mobile inicia recolhido e recolhe ao cruzar breakpoint 760px; expansão manual abre sidebar 234px sobre conteúdo, sem diminuir a coluna central, que mantém margem 66px. Essa expansão permite ler e navegar; botão recolhe novamente. Sem API, credenciais, contratos/backend/migration ou dependência.

## Checks e evidências
46 testes web/18 arquivos, typecheck/build web e lint raiz PASS na árvore combinada; aviso preexistente da API preservado. Browser: toggle por teclado, width/margem/texto/ícones/title/aria, persistência navegação/reload, expansão mobile, storage deliberadamente bloqueado, nenhum pageerror/overflow em 1536/1024/768/390/320px. [Resultados](../evidencias/controle/evidencias/FAC-027/browser-results.txt), [script](../evidencias/controle/evidencias/FAC-027/browser-check.mjs), [recolhido](../evidencias/controle/evidencias/FAC-027/collapsed.png), [mobile](../evidencias/controle/evidencias/FAC-027/mobile.png). Capturas de menu e agentes inspecionadas. [Checks compartilhados FAC-026](../evidencias/configuracao/evidencias/FAC-026/build.txt), [patch próprio](../evidencias/controle/evidencias/FAC-027/implementation.patch).

Primeiro ensaio apontou width literal 210px de breakpoint anterior sobrescrevendo tamanho recolhido. Substituído por variável comum e override mobile 234px expandido; nova execução confirma os tamanhos e o fluxo dos agentes. Sem alegar funcionamento só por build.

## Limites, rollback, docs e aceite
Preferência é local ao navegador; mobile adota recolhimento inicial mesmo se desktop estava expandido. Não sincroniza conta/dispositivos. Em mobile, menu expandido cobre parte do conteúdo até recolher. Reverter `b1d794035342128b9ed832d5fe9b74c887eb7075` restaura comportamento anterior do menu; chave localStorage remanescente é inerte, sem dados do servidor. Sem rollback executado.
README raiz/controle/configuração, INDEX, CHANGELOG, BACKLOG/ticket e lesson de estado do template atualizados. AWAITING_HUMAN após checks; DONE só após aceite exato. Integração/limpeza serão registradas após realização.

Validação final na revisão d3cf9990128fc4811cf7ed54806a0efd3ddc0949: teste estático antigo exigia atributos HTML adjacentes; inclusão de title expôs essa fragilidade. Regex ajustada para verificar menu ativo independentemente de atributos intermediários; 46 testes passaram novamente, sem alterar comportamento. Logs finais substituem o ensaio anterior.

## Integração e limpeza realizadas
Fast-forward local em developer até f9d20ae, sem conflitos. Build web no destino PASS com mesmos bundles; ambos os fluxos browser reproduzidos no painel habitual/5173 PASS. [Resultado developer](../evidencias/controle/evidencias/FAC-027/browser-results-developer.txt). Ensaios de settings usam cinco PUTs em memória interceptados; nenhuma alteração real de ativação/conta/job.
Worktree limpa e feature ancestral comprovadas. Preview próprio PID 2575191 parado após conferir cwd; git worktree remove /home/vinicius/le-fabrique-fac-026 realizado sem force. Diretório ausente, dependências principais preservadas e somente developer em git worktree list; branch feature preservada. Sem push/deploy; aceite visual/funcional pendente.
