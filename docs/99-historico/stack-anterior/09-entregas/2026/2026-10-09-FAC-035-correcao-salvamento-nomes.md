# FAC-035 — Correção da inicialização para salvar nomes

2026-10-09, Europe/Berlin. Domínios: desenvolvimento e configuração. IMPLEMENTADO / AWAITING_HUMAN, base 38998ebf313625d2202dcb51c574803d78a3a1ef, branch `fix/agent-name-persistence`, código 6184f78. Correção solicitada pelo responsável ao relatar que o nome não salva no frontend. Relacionada ao [ticket de nomes](../../08-desenvolvimento/tickets/configuracao/FAC-035-nomes-agentes.md), distinta do pacote Scrum ainda não aplicado.

## Objetivo, problema e critérios

O formulário envia `nickname` via PUT /api/settings. O código-fonte atual aceita o campo; o build CommonJS local de 2026-10-05 não aceita. Reproduzido erro `unrecognized_keys` no contrato carregado por require. O frontend resolve a exportação import para o fonte e a API CommonJS usa dist, permitindo divergência após atualização do código. Critério desta correção: iniciar desenvolvimento com contratos atuais e preservar nomes nos testes existentes de gravação/leitura.

## Solução e funcionamento

O hook npm `predev` recompila contratos e runtime antes de iniciar os três workspaces. Se a compilação falhar, npm não executa dev. O runtime é dependência compilada do worker e segue a mesma preparação. Reiniciar `npm run dev` na raiz utiliza os artefatos novos; iniciar apenas web não atualiza a API. Nenhum dado, rota, contrato público ou schema de banco mudou.

## Arquivos e diff

`package.json`: preparação de desenvolvimento. Guia de ambiente local e módulo de configuração: explicação do requisito. Evidências e script de browser sintético em [correcao-nomes](evidencias/FAC-035/correcao-nomes/checks.txt). Diff disponível na branch isolada sobre a base declarada. Sem dependências novas; lockfile preservado. Políticas, AGENTS, perfil/skills locais, Manual/README/piloto/ADR-003 e ticket foram lidos explicitamente nesta conversa; autoload não verificado.

## Testes e evidências

[Resultados por comando](evidencias/FAC-035/correcao-nomes/checks.txt): predev, contratos (53), serviço SettingsService (8), launcher (2), builds API/web PASS. O contrato compilado após o hook preserva Alex. Serviço usa Prisma simulado; não prova banco real. Browser NOT_RUN: Chromium falhou antes de abrir a tela por ausência de libatk-1.0.so.0. [Script preparado](evidencias/FAC-035/correcao-nomes/browser-save.mjs) usa SettingsService real e persistência sintética em memória; nenhuma credencial real.

## Ambiente, riscos e limites

O único listener de painel identificado na VPS é 8080, com containers de 2026-09-30, API sem rota settings. Isso não identifica a origem exata da tela usada pelo responsável. Os containers não foram atualizados: substituir toda a instalação antiga exigiria validar diferenças além desta correção. Não foi consultada configuração real ou segredo; não houve escrita no banco, migration, provider ou execução de agentes. O bug de compilação foi reproduzido; a causa da falha na sessão específica do usuário permanece NÃO VALIDADA. Risco R1: preparação local acrescenta tempo de build e pode revelar erro de compilação antes de iniciar os serviços.

## Rollback

Remover somente predev de package.json reverte a preparação automática; não afeta dados. É necessário compilar contratos manualmente antes de iniciar API se o código mudou. Não é necessário rollback de banco.

## Documentação, lessons, uso e aceite

Atualizados [ambiente local](../../08-desenvolvimento/01-ambiente-local.md), [configuração](../../03-modulos/configuracao/00-README.md), backlog/matriz/índices/changelog e adendo do ticket. Sem ADR ou lesson nova: mesma arquitetura e exports existentes. Codex; modelo/uso/custo da sessão não informados, sem gastos novos ou autenticação consultada. Revisão independente, persistência no ambiente do usuário e aceite humano pendentes. FAC-035 não é DONE.
