# FAC-012Z — Gate de documentação técnica

Data: 2026-10-03. Status: IMPLEMENTADO / AWAITING_HUMAN. Base `745ce2d`, READY `767e0f7`, código `9c48dd992646db317714c75421a5bf872f3c5f69`. Branch `feat/fac-012z-documentation-gate`, worktree `/home/vinicius/le-fabrique-fac-012z`. Origem limpa sem patch/untracked; nenhum outro writer nesta worktree. Worktree raiz e serviços preservados.

## Objetivo e critérios de aceite

Exigir documentação técnica atualizada no projeto antes de entrega, com configuração pela interface, validação estrutural bounded, revisão semântica independente e evidências da revisão exata. Um writer e até duas correções; nenhum provider real. Preservar stack externa, caminhos e contratos legados.

## Funcionamento

`executionProfile.documentation` guarda `requiredFiles` (1–30 Markdown), `reportPath` pertencente à lista e `requiredSections` (1–20 títulos únicos). Definição valida todos os documentos dentro dos caminhos permitidos, fora dos proibidos e de `.git`/`.codex`. Painel permite configurar os padrões reais de README, índice, changelog, backlog e lessons do projeto, sem impor estrutura da fábrica ao externo. Novos READY exigem política; registros legados permanecem legíveis, mas execução real sem política falha fechado no compilador.

Developer recebe requisitos, SHA base e checks, atualiza documentos junto do código antes de checks/snapshot/revisão. Prompt exige não inventar resultado de supervisor ainda não executado. Depois de stop e checks sem regressão, gate compara arquivos alterados/untracked via Git sem shell/fsmonitor/config global e lê documentos regulares até 256 KiB cada, sem symlinks/traversal ou truncamento. Seções vazias, placeholders conhecidos, ausência do SHA base ou nomes de checks no relatório, documentos ausentes/inalterados bloqueiam. Falta documental consome rodada de correção existente; no limite retorna PAUSED_LIMIT/DOCUMENTATION_INCOMPLETE com snapshot preservado e sem Reviewer.

Gate publica hashes de conteúdo/manifesto e códigos controlados, nunca texto de documento. Reviewer recebe política/evidência e deve ler/conferir documentação contra código, critérios, checks reais, limites e rollback. Após revisão, novo snapshot byte-equivalente e novo gate são obrigatórios; mutação de patch/untracked durante sessão READ_ONLY invalida a revisão, preserva o estado e retorna FAILED. Hash do gate aponta ao snapshot final. Mesmo APPROVE segue AWAITING_HUMAN.

`RunDeliveryService` exige PASS sem findings, conjunto exato de documentos e manifesto final; hashes de documentos novos também correspondem ao bundle. Documento público e RunPanel mostram as evidências. Dado divergente impede elegibilidade/DONE mesmo sob resultado aprovado. O digest de entrega existente inclui esses campos, mantendo aceite vinculado à revisão exata.

## Alterações e diff

Diff sanitizado revisável: `git show 9c48dd992646db317714c75421a5bf872f3c5f69 -- apps packages/contracts/src`. 15 arquivos, 689 inserções/3 remoções; fixtures sintéticas, sem segredo. Não houve migration nem dependência nova.

## Verificação

- `npm ci --ignore-scripts` e `npm run db:generate`: concluídos localmente, sem banco.
- Baseline: primeira `npm test` falhou por ausência de dist runtime em checkout novo; após `npm run build`, 411 testes passaram. Falha de preparação, anterior às mudanças.
- Revisão final: `npm test`: 422 passaram (launcher 2; contracts 34; runtime 46; API 141; worker 181; web 18).
- `npm run typecheck`, `npm run lint` (200 arquivos) e `npm run build`: passaram. Worker typecheck repetido após endurecimento ambiental/UTF-8 do gate.
- `git diff --check`: passou.

Testes cobrem arquivos alterados rastreados/novos, hash/manifesto, ausente/inalterado, seção vazia/placeholder/SHA/checks ausentes, symlinks e limite de tamanho; workflow impede Reviewer quando incompleto e invalida aprovação após mutação; READY/compilador bloqueiam política ausente; contrato rejeita documentação fora do escopo; controle recusa manifesto documental divergente. Sem prova operacional de provider/serviço.

## Riscos e limitações

Verificação estrutural não comprova verdade nem links de todo projeto. Nomes de checks e SHA no texto são referências mínimas; Reviewer/humano verificam afirmações e cobertura. Arquivos históricos contendo placeholders conhecidos precisam ser atualizados ou excluídos da lista obrigatória conforme política do projeto; não há exceção silenciosa. Projetos legados precisam salvar política antes de novos READY; jobs antigos sem ela não executam. Gate depende de quiescência dos processos anteriores, não é proteção contra writer externo não supervisionado. Ainda falta confinamento granular Claude/preflight real e validação operacional/piloto. Consumer permanece desligado.

## Rollback

Com writer parado e ticket autorizado, reverter código preservando configurações/resultados e artefatos. Readers antigos strict podem rejeitar campos novos; manter reader compatível para recuperar evidências e não apagar dados para forçar compatibilidade. Nenhum merge/deploy/migration desta entrega.

## Documentação e lessons

Atualizados README raiz, controle/operação/planejamento, ADR-003, INDEX, CHANGELOG, BACKLOG, CONTROLE-MVP e lesson `baseline-regressao-review` com exemplo concreto do gate estrutural/semântico. Ticket/documento de entrega associados ao código exato.

## Uso de IA e aceite

Nenhuma chamada dos providers da fábrica, API/extras/créditos/recarga/login. Assistente Codex CLI 0.159.2; modelo efetivo/uso/cota/custo não observáveis nesta interface. Fixtures não contam como piloto. IMPLEMENTADO; aceite humano da revisão exata pendente, não DONE. Sem push/merge/deploy ou acesso a produção.
