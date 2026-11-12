# FAC-012W — Transporte ampliado de artefatos

Data: 2026-10-03. Status: IMPLEMENTADO / AWAITING_HUMAN. Baseline `f996169`, READY `fb5ad0c`; código `2a1ae96e42ffb1c6f0f0348c88e53986e5ec7de9`. Branch `feat/fac-012w-artifact-budget`, worktree `/home/vinicius/le-fabrique-fac-012w`.

## Objetivo e implementação

Entregar alterações acima do teto anterior 64 KiB sem truncar, mantendo integridade, fencing, journal, gate humano e limites explícitos. Contracts exporta ARTIFACT_JSON_MAX_BYTES (8 MiB), ARTIFACT_RAW_MAX_BYTES (6 MiB) e ARTIFACT_HTTP_MAX_BYTES (JSON + 4096 bytes para envelope). Metadados/base64 participam do teto JSON: 6 MiB raw não garante caber, pois base64 e manifesto adicionam bytes. Nenhum payload é truncado/comprimido silenciosamente.

Schema mantém base64 canônico, agora sem regex de grupos repetidos em strings grandes; caracteres/tamanho/alinhamento e roundtrip atob/btoa são verificados. DTO máximo continua obrigatório antes de persistência. Hashes e filtro de segredos conhecidos permanecem; não há garantia universal de detecção de segredo.

ArtifactReader confere totais declarados antes de ler patch/arquivos, exige tamanho físico igual ao manifesto e usa leituras bounded (teto esperado + um byte), não readFile ilimitado após stat. Isso impede acumular arquivos grandes com total declarado falso. Canonicidade/owner/root privado/O_NOFOLLOW continuam. Oversized deixa evidências no disco/journal, não aprova nem repete IA.

API configura parser JSON Nest/Express limitado ao teto de envelope, superando default 100 KiB; nenhum parser ilimitado. Auth/guards/schema por endpoint inalterados. Essa política é global para JSON; rotas comuns continuam limitadas por seus próprios contratos. Proxy público mantém seu limite anterior e bloqueia /api/internal; worker usa API loopback. Downloads GET não precisam ampliar upload público. [NestJS oficial](https://docs.nestjs.com/faq/raw-body), consultado 2026-10-03, confirma useBodyParser com limite e tipo NestExpressApplication. Painel mantém diff integral como texto escapado/download sob demanda e informa teto.

## Diff e checks

Diff sanitizado: `git show 2a1ae96e42ffb1c6f0f0348c88e53986e5ec7de9 -- apps/api/src apps/web/src/ArtifactPanel* apps/worker/src/artifact-reader* packages/contracts/src`; nove arquivos, 197 inserções/45 remoções, sem credenciais reais.

- `npm ci --ignore-scripts`: dependências privadas, audit zero vulnerabilidades.
- `npm run db:generate`: geração local sem banco/migration.
- `npm run typecheck`: todos passaram na revisão final.
- `npm test`: 387 passaram (launcher 2, contracts 34, runtime 46, API 137, worker 152, web 16).
- `npm run build`: cinco workspaces passaram; alterações finais subsequentes foram só fixtures HTTP/UI e foram testadas/typechecked.
- `npm run lint`: 196 arquivos sem fixes/info na revisão final.
- `git diff --check`: passou.

Evidências novas: bundle multi-megabyte com hashes, oversized raw/JSON recusado, Git real com arquivo 256 KiB entregue sem truncamento, totais incoerentes recusados antes de conteúdo, painel estático escapado acima de 64 KiB com final preservado. Servidor HTTP de fixture Nest real em loopback/porta efêmera aceita 256 KiB e perto de 8 MiB+envelope, rejeita excesso com 413; fechado após cada teste. Não foi inicializado AppModule/banco/API ativa, nem essa fixture substitui teste operacional dos endpoints autenticados.

A primeira suíte encontrou incompatibilidade Rolldown com decorator de parâmetro na fixture HTTP; uma repetição já iniciada reproduziu o mesmo erro antes da correção. Diagnóstico: usar as funções de metadata Nest na fixture, sem alterar implementação/parsers. Dois testes HTTP então passaram e a suíte inteira passou. Um aviso informativo useTemplate posterior foi corrigido; lint e testes UI repetidos. Não foi falha de baseline nem ocultado como sucesso.

## Limites, rollback e aceite

Operação/retention/backup seguem manuais; não há limpeza automática. Teto finito explícito é adequado ao transporte de incrementos pequenos do MVP, não repositório inteiro, arquivos ilimitados ou streaming distribuído. Snapshots locais podem exceder entrega; excesso é recusado com trabalho preservado. Parser aumentado exige controle operacional de concorrência/rede/memória; um writer permanece. API/extras/fallback pago desligados. Consumer false, sem migration/provider/piloto/login/serviço real/push/deploy.

Rollback com worker parado: reverter código preservando bundles/journals; clientes antigos de 64 KiB não conseguem carregar bundles maiores já persistidos, portanto manter acesso à revisão compatível para recuperar evidências. Nenhum campo/migration alterado. Aceite humano da revisão exata pendente, não DONE.

## Documentação, lessons e uso de IA

Atualizados READMEs controle/runtime/operação/infraestrutura/planejamento, README raiz, controle MVP, ticket, índice/changelog/backlog e lesson contratos. Sem chamada de IA pelos clientes da fábrica ou gasto habilitado; modelo/tokens/custo do assistente não observáveis. Próximas lacunas: handoff/WAITING_PROVIDER, confinamento Claude e docs técnicas do projeto.
