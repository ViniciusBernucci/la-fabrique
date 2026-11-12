# FAC-012AA — Perfil granular e preflight Claude

Data: 2026-10-03. Status: IMPLEMENTADO / AWAITING_HUMAN; prova de serviço NÃO VERIFICADA. Base `ca41328`, READY `90b165c`, código `56f2e38d2d204418bb594d3679a753057f8e4589`. Branch `feat/fac-012aa-claude-confinement`, worktree `/home/vinicius/le-fabrique-fac-012aa`; inclui FAC-012Z por ancestry, sem merge. Root `developer` segue `745ce2d`, limpo. Um writer por worktree; serviços/root preservados.

## Objetivo e critérios de aceite

Confinar o Developer Claude aos caminhos permitidos e tornar sua elegibilidade dependente de prova nativa privada da identidade/binário/política. Implementar preflight opt-in capaz de recusar sucesso fictício, mas não usar prova sintética para liberar o CLI oficial. Manter assinatura oficial, sem APIs/extras, shell ou login automático.

## Funcionamento

`claudePermissionSettings` substitui `acceptEdits` por `dontAsk`, allow `Edit(//caminho-absoluto)` e `Edit(//diretório/**)` somente para entradas existentes, canônicas, sem symlinks/traversal/padrões especiais. Read-only não possui allow de edição e expõe somente Read/Glob/Grep; writing acrescenta Edit/Write, sem código executável gerado. Deny explícito protege `.git`, `.codex`, `.claude`, Bash/PowerShell/REPL/Agent/WebFetch/WebSearch/NotebookEdit; safe-mode/restricted/MCP estrito/browser desligado/sessão efêmera/prompts none permanecem. Flags e regras são dados estruturados de argv, não prompt. Caminho incompatível falha antes de spawn.

Regras `Edit` se aplicam às ferramentas de edição/escrita; `dontAsk` nega ações que precisariam de confirmação sem allow. Caminhos absolutos usam duas barras; symlinks exigem correspondência do caminho solicitado e resolvido. Consultado em 03/10/2026: [permissões oficiais](https://code.claude.com/docs/en/permissions), [referência CLI](https://code.claude.com/docs/en/cli-reference). Help local: Claude Code 2.1.285; versão/binário de serviço continuam pendentes. Isso fundamenta o perfil; não substitui ensaio real.

`ProviderIdentityManager.adapter` só fornece Claude writer quando `claude-<installationId>/confinement.json`, fora do store oficial/workspaces, é privado/canônico, do UID atual, até 64 KiB e validado em runtime. Prova exige política `claude-granular-v1`, ID/UID exatos, CLI 2.1.285, SHA-256 do binário atual, duas operações permitidas, os 14 casos negados e data até sete dias (futuro recusado). Troca do binário, outra conta/UID/política, prazo vencido, prova incompleta/pública/symlinked bloqueiam escrita. Router/elegibilidade fiscal da UI permanecem gates independentes, sem conta/modelo fixos ou fallback. Library direta é usada pelo preflight; consumer real usa factory protegida.

`npm run preflight:claude -- ...` é operação opt-in sob UID do serviço, depois de login oficial, verificação humana de assinatura/extras e stop físico do worker. Recusa falta de confirmação ou overrides de API/cloud/OAuth herdados antes de criar identidades; status precisa ser assinatura AVAILABLE e versão suportada. Lock exclusivo no root privado impede duas provas concorrentes; lock abandonado por crash permanece até reconciliação física manual. Flags de confirmação registram declaração humana, não fingem observação de processo/financeiro.

Uma chamada Claude (modelo explícito, máximo 180 s/1 MiB de saída) tenta Write/Edit permitidos e 14 operações proibidas em canários sintéticos: escrever/editar fora da allowlist, fora da worktree, via symlink e metadados protegidos; ler fora/via symlink. Observador privado correlaciona tool_use de assistant com tool_result de user por ID/path/tool, retendo só classificação, nunca conteúdo. Cada negação exige tentativa e resultado nativo de erro/negação; IDs/resultados incoerentes recusam prova. Supervisor verifica bytes de arquivos permitidos/canários e SHA do CLI novamente. Mensagem/exit zero ou arquivo intacto sem tentativa não bastam.

Prova só nasce após todas as verificações. Prova anterior é preservada por rename antes do novo experimento de ferramentas, retirando elegibilidade durante o teste; falha não restaura automaticamente evidência antiga. Novo JSON é exclusivo, privado e sem prompt/auth/trace bruto; stdout publica metadados/uso efetivamente observados e local de fixture sintética. Falhas mostram estágio controlado. Fixtures ficam para diagnóstico; não há limpeza de evidências reais automática. SIGINT/SIGTERM cancelam a execução e impedem publicação, inclusive sinal durante preparação do adapter. Crash duro conserva lock e exige operador.

## Alterações e diff

`git show 56f2e38d2d204418bb594d3679a753057f8e4589 -- apps/worker packages/runtime scripts package.json biome.json tsconfig.scripts.json`: diff sanitizado revisável, 15 arquivos, 967 inserções/11 remoções. Sem dependência/migration/serviço novo. Typecheck/lint agora incluem scripts TypeScript/preflight/fixture; testes usam só diretórios temporários e binários sintéticos. Fingerprint da fixture difere do cliente oficial; nenhum arquivo de prova foi criado para instalação real.

## Verificação

- `npm ci --ignore-scripts`, `npm run db:generate`, build da baseline: passaram, sem banco.
- `npm run typecheck`: passou em todos os workspaces e scripts.
- `npm test`: 433 passaram (launcher/preflight 5; contracts 34; runtime 50; API 141; worker 185; web 18).
- `npm run lint`: passou, 207 arquivos, um aviso herdado de Z (`useOptionalChain` no gate de delivery), sem erro. Primeira rodada teve erro noImplicitAnyLet novo; corrigido com tipo explícito, sem falha de testes.
- `npm run build`: passou nos cinco workspaces.
- Após ajuste final de sinal no script: `npm run typecheck:scripts`, `node --test scripts/claude-preflight.test.mjs` (3) e lint passaram; script não faz parte do build compilado dos apps.
- `git diff --check`: passou.

Testes comprovam configuração literal, read-only sem escrita, paths inválidos/ausentes/symlink/padrão, prova ausente/insegura/stale/CLI diferente, correlação sem conteúdo secreto, sucesso sem trace recusado, prova sintética privada e renovação preservada. A fixture positiva apenas simula transporte/negações e escreve arquivos sintéticos; NÃO testa o mecanismo de permissões do Claude. Preflight real não executado, assinatura/cota/financeiro/isolamento sob serviço NÃO VERIFICADOS.

## Operação pendente

Conferir o checklist em `documentacoes/operacao/README.md`. É necessário informar instalação/modelo da UI, WORKER_PROVIDER_ROOT/WORKER_CLAUDE_BINARY reais e UID do serviço, autenticar pelo cliente oficial e confirmar extras desligados. Executar preflight sob essa identidade com worker comprovadamente parado e guardar saída sanitizada como evidência. Nenhuma credencial deve ser copiada/exportada. Real piloto/serviço/migrations/backup/reboot/recursos seguem validações próprias. Sem resposta sobre piloto/conta até esta entrega; não foram inferidos.

## Riscos e limitações

Restrição é nativa dos file tools, não sandbox de todo processo CLI. Ferramentas que executam código/customizações/MCP não estão expostas; checks mantêm o sandbox OS separado. Managed settings e administração do host são confiáveis e precisam de novo preflight após mudança; prazo de sete dias não comprova revogação instantânea. Origem oficial do binário é responsabilidade do operador; hash detecta mudança, não autenticidade. Prova privada pressupõe exclusão dos writers externos e root fora da escrita do projeto. O teste pode falhar se o modelo não tentar todos os casos ou formato nativo mudar; nesse caso manter bloqueio e diagnosticar, sem atalho de flag/declaração. Expiração/versão desconhecida falham fechado. `tsx`/dependências de desenvolvimento são necessários para o comando operacional.

A implementação do software cobre as duas lacunas finais mapeadas em CONTROLE-MVP, mas o MVP operacional não está concluído: prova real, piloto, instalação/migrations/backup e aceite humano faltam. Consumer continua desligado; nenhum serviço foi alterado.

## Rollback

Com writer parado e ticket autorizado, voltar código compatível preservando settings/resultados/snapshots/provas superseded. Retirar elegibilidade é preferível a restaurar prova vencida; versão anterior da factory volta a bloquear Claude writer. Não voltar ao perfil amplo para contornar gate. Git revert exige revisão humana/integração autorizada; root/deploy não foram modificados.

## Documentação, lessons e IA

README raiz/runtime/operação/planejamento, ADR-002/003, índice/changelog/backlog/CONTROLE-MVP e lesson de perfis atualizados. Código e prova operacional explicitamente separados. Assistente Codex CLI 0.159.2; modelo/cota/tempo/tokens/custo efetivos não disponíveis. Nenhuma chamada de provider real, login, APIs de IA/extras/créditos/recarga/migrations/piloto/push/merge/deploy. Status IMPLEMENTADO/AWAITING_HUMAN, aceite da revisão exata pendente; não DONE.
