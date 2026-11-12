# DOC-MV-001 — Migração inicial do Manual Vivo

Data 2026-10-07 Europe/Berlin. IMPLEMENTADO documentalmente / AWAITING_HUMAN. Base de código a2cc5e0db0f30bf227dfc64300f74bde23a6f412; branch docs/manual-vivo-inicial, worktree /home/vinicius/le-fabrique-manual-vivo. Revisão final será vinculada ao commit/patch e manifesto de revisão, sem hash circular no próprio texto.

## Objetivo e critérios

Mesclar /home/vinicius/le-fabrique/reorganizacao-documental com o repo executável: Manual Vivo, política comum, C4, módulos/features/contratos/ADRs/operação/desenvolvimento/entregas/lessons, inventário e mapa por seção, preservação dos originais. Critérios completos no [ticket READY](../../08-desenvolvimento/tickets/DOC-MV-001.md).

## Funcionamento e alterações

docs/README ensina o fluxo; índices navegam. Políticas em docs/00-governanca são fonte única corrente, wrappers preservam especificidades e leitura explícita. Conteúdo local transferido integralmente com links editoriais e nota de revisão onde frases de incremento estão superadas. Relatos por domínio em docs/09-entregas/ano, lessons reais em docs/10-lessons e tickets em desenvolvimento. Backlog/changelog reais preservados, pacote de planejamento não sobrescreve aceites.

Código/schema/controller prevalece para implementação: C4 atual separado do alvo, AS-IS explícito, rota READY/claim real, enums distintos, ADR-003 TypeScript aceito separado da proposta colidente. Tenancy/broker/RLS/SEC/LF-MT permanecem propostas. Só exemplos de paths Markdown no painel mudaram; política configurável/gates/segurança/execução não alteradas.

## Origem, arquivos, diff e revisão

[Manifesto](../../00-governanca/MANIFESTO-FONTES.json), [inventário](../../00-governanca/INVENTARIO-REPO.json), [destinos](../../00-governanca/DESTINOS-REPO.json), [mapa por seção](../../00-governanca/MAPA-SECOES-REPO.json) e [auditoria](../../00-governanca/08-AUDITORIA.md). Snapshots repo-a2cc5e0 e pacote-recebido preservam originais/PDF/ZIP/manifestos integralmente. [Patch editorial sanitizado](evidencias/DOC-MV-001/editorial.patch) e [manifesto da revisão](evidencias/DOC-MV-001/REVISAO.json) identificam o conjunto revisável. Patch exclui snapshots/evidências binárias e JSONs volumosos de proveniência; git diff/commit da branch contém a totalidade, inclusive esses artefatos. Exclusões do manifesto evitam hash circular e são explícitas.

## Checks e resultados

[Validação e resultados reais](../../00-governanca/11-VALIDACAO.md). Web typecheck PASS e 46 testes web PASS para mudança exclusiva de exemplos. Validador final PASS: 367 Markdown correntes, 1.734 links, 601 snapshots/origens e 2.444 seções. Whitespace corrente PASS, histórico íntegro preserva whitespace original; lint PASS com warning prévio; PDF extraído e sete páginas conferidas; Mermaid revisão textual PASS/render NOT_RUN por bibliotecas locais ausentes. Resultados registrados após os checks; ausência de ensaio operacional declarada NOT_RUN, sem transformar catálogo de testes em PASS.

## Dados, API e configuração

Nenhum schema/controller/migration/ambiente real alterado. Contratos documentais corrigidos a partir de código: não cria aliases de rota, enums ou tabela/serviço. ProjectDefinitionPanel só troca placeholders documentacoes por docs; operador ainda configura projeto externo segundo sua stack/paths próprios. Nenhum default runtime ativado.

## Riscos, limitações e rollback

Links externos históricos/preflight e autoload não comprovados; proposta mais nova não recebe aceite por cronologia. Worktree de origem e pacote fornecido preservados, sem env/credenciais/DB lidos. UI exemplo não migra configuração salva. Sem revisão humana independente de todo conteúdo ou eficácia de segurança/produção.

Rollback: revisar dependentes e reverter somente commit/patch desta migração documental; snapshots restauram apenas paths alterados no mapa, nunca reset destrutivo ou overwrite de trabalho prévio. Não reverter DB/serviço/auth: fora do escopo. Entradas legadas essenciais permanecem bridges; individuais redundantes foram removidos após conferência de destinos/hashes, e originais íntegros fornecem base de comparação. Reversão não ensaiada em serviço.

## Docs e lessons

[Manual](../../01-README.md), [políticas](../../00-governanca/00-README.md), [C4/AS-IS](../../02-arquitetura/00-README.md), [módulos](../../03-modulos/00-README.md), [features](../../04-features/00-README.md), [contratos](../../05-contratos/00-README.md), [ADRs](../../06-decisoes/00-README.md), [operação](../../07-operacao/00-README.md), [desenvolvimento](../../08-desenvolvimento/00-README.md). Lessons prévias têm exemplos/revisões preservados; [lesson de documentação/proveniência](../../10-lessons/12-documentacao-proveniencia.md) descreve somente conceito aplicado nesta migração.

## Uso e aceite

Agente Codex nesta sessão; versões somente --version: codex 0.159.2, Claude 2.1.285, agy 1.2.17. Modelo efetivo/tempo/tokens/cota/cobrança desconhecidos; não lidos de auth/log privado. Zero subprocesso de inferência/provider, gasto API/extra/fallback não habilitado. Consultas de ferramentas locais não são preflight operacional. Dependência temporária PyMuPDF 1.26.7 fora do repo para ler/renderizar PDF, sem alteração de manifests/lockfile/providers.

Revisão semântica própria registrada com fontes locais; revisão independente/aceite humano da revisão exata PENDENTE. DOC-MV-001 AWAITING_HUMAN, nenhum software FAC/OPS/LF-MT promovido a DONE. Sem merge/push/deploy/publicação/ativação/migração irreversível.

## Limpeza de redundâncias autorizada

Pedido posterior do usuário autorizou apagar MD desnecessários/redundantes. Removidos 202 bridges individuais em documentacoes/lessons/templates após conferir capítulo canônico e snapshot/hash original. Guias, atalhos essenciais, módulos/contratos/relatos/lessons e originais íntegros permanecem. [Lista verificável](../../00-governanca/MD-REMOVIDOS.json). Rollback desses paths usa exatamente os originais do manifesto ou a revisão anterior; não exige recuperar conteúdo perdido.

## Integração em developer — 2026-10-07

Após a entrega inicial, o usuário autorizou explicitamente os commits, merge em developer e remoção de worktrees órfãs. Integrado o commit documental 607d6ededcb6499cac5df10ee4adb60a4b3156b1 por `git merge --ff-only docs/manual-vivo-inicial`, partindo de a2cc5e0db0f30bf227dfc64300f74bde23a6f412; sem conflitos. O Manual Vivo agora está em `/home/vinicius/le-fabrique/docs`, na branch developer. As menções à worktree e ausência de merge acima registram o estado da entrega original.

A worktree documental foi removida por `git worktree remove` sem force, após confirmar integração e ausência de mudanças pendentes. Seus únicos itens ignorados eram outputs locais de build/test e o symlink de dependências; a origem de node_modules foi preservada. `git worktree prune --verbose` não encontrou registros órfãos adicionais. Resta somente a worktree principal. A branch documental permanece como referência; o pacote recebido reorganizacao-documental/ continua intacto e untracked. Nenhum push/deploy/serviço/migration foi executado.

[Checks da integração](evidencias/DOC-MV-001/INTEGRACAO.json) e [validação pós-integração](evidencias/DOC-MV-001/VALIDACAO-INTEGRACAO.json) registram evidências reais. Sem novo diff de software desde os checks web originais; não repetidos por esta operação Git. O manifesto REVISAO.json e o patch editorial descrevem a revisão inicial 607d6ed, anterior a este registro adicional. Autoload, render Mermaid e revisão semântica independente permanecem com os limites já declarados; autorização de merge não comprova esses testes.

Rollback da integração: `git revert 607d6ededcb6499cac5df10ee4adb60a4b3156b1` em revisão isolada, preservando alterações posteriores; sem reset destrutivo. Para consultar a revisão original, usar Git; recriar worktree não é necessário para acessar docs atuais.

## Remoção do pacote redundante — 2026-10-07

Após pedido do usuário para retirar a pasta se tudo estivesse preservado, os 230 arquivos de reorganizacao-documental/ foram comparados byte a byte e por SHA-256 com docs/99-historico/pacote-recebido/. Zero ausências ou divergências; validador documental PASS. A pasta untracked original foi removida; a cópia histórica íntegra permanece versionada. Este registro sucede a preservação temporária relatada na integração acima. [Evidência e hashes](evidencias/DOC-MV-001/REMOCAO-PACOTE.json). Rollback: copiar pacote-recebido/ de volta para reorganizacao-documental/, sem modificar docs canônicos.
