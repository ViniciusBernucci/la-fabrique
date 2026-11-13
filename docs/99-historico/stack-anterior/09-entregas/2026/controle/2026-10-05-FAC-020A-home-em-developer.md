# FAC-020A — Home no aplicativo habitual
Data: 2026-10-05. Status: IMPLEMENTADO / AWAITING_HUMAN.
Domínio: controle. Baseline developer: `8285173`; integração inicial: `e87206f`; revisão atual do código: `cde0cc6` (FAC-021).
## Objetivo e autorização
O responsável informou que a tela criada é o dashboard principal/home e pediu explicitamente merge em developer e remoção da worktree após integração. Ticket [READY FAC-020A](../../../08-desenvolvimento/tickets/controle/tickets/FAC-020A.md). A entrega anterior estava apenas na worktree e o Vite habitual ainda servia a entrada antiga.
## O que mudou e funcionamento
Fast-forward local de `feat/fac-020-home-dashboard` para `developer`, primeiro até e87206f, depois até cde0cc6 com nome/tamanho solicitados posteriormente. Vite existente, PID 2353760, cwd `/home/vinicius/le-fabrique/apps/web`, porta 5173, passou a servir HomeDashboard. Nenhuma reinicialização do aplicativo ou alteração de banco. Autenticação administrativa continua protegida. Sem push/deploy.
## Checks e evidências
Root limpo antes do merge; ancestry permitiu fast-forward sem conflitos. Fonte servida por `/src/App.tsx` contém HomeDashboard/showHome. 42 testes web passaram na developer após integração inicial; browser Chromium 153.0.8010.12 passou na porta 5173: home sem API, busca, diálogos/foco, seis larguras e login 401 sintético. [Resultado](../evidencias/controle/evidencias/FAC-020A/browser-results.json). Verificação posterior com novo nome/escala: [FAC-021](2026-10-05-FAC-021-nome-escala-home.md).
## Diff e revisão
[Patch da home](../evidencias/controle/evidencias/FAC-020/implementation.patch), código 904d0b5, e [patch de nome/escala](../evidencias/controle/evidencias/FAC-021/implementation.patch), código cde0cc6. Integração Git sem conflitos e sem diferenças extras de implementação. Logs são sintéticos e sanitizados.
## Limites e rollback
Operação local de desenvolvimento, não publicação em produção. Dados da home demonstrativos; aceite visual não presumido. Reverter 904d0b5 para retirar a home e cde0cc6 para desfazer nome/escala, sem apagar histórico nem dados. Remoção da worktree condicionada à preservação de todos os commits em developer; registro de conclusão será acrescentado após o comando real.
## Documentação e lessons
README raiz/controle/operação, INDEX, CHANGELOG, BACKLOG, ticket e [lesson de baseline](../../../10-lessons/11-baseline-regressao-review.md). Código pronto em branch não comprova que o serviço habitual serve essa revisão: verificar cwd/porta e navegador no endpoint real.
## IA e custos
Mesmo Codex da sessão, um writer; nenhum provider adicional, acesso a credenciais ou alteração de gastos. Modelo efetivo, cota e uso total não comprovados.
## Aceite
Integração/limpeza autorizadas. AWAITING_HUMAN para aceite visual da revisão; merge não equivale a DONE.

Atualização posterior: `4e042a1` integrado por fast-forward, com template/fontes FAC-022. Browser na porta 5173 confirmou a home e menus persistentes. Build completo em cde0cc6 e web em 4e042a1 PASS.

## Conclusão da integração e limpeza
Em 2026-10-05, merge final fast-forward até 776a742, ancestry da feature comprovada, worktree limpa. Preview próprio PID 2345979 encerrado após conferir cwd; `git worktree remove /home/vinicius/le-fabrique-fac-020` executado sem force. `git worktree list` passou a conter somente `/home/vinicius/le-fabrique [developer]`; diretório antigo ausente e node_modules principal preservado. Branch feature preservada, sem push/deploy. Registro administrativo de encerramento feito em developer após remover a worktree, sem alterar código/evidências.
