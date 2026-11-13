# FAC-013 — Lista de contas IA e configuração em modal

Data: 2026-10-05. IMPLEMENTADO / AWAITING_HUMAN. Baseline `5f642d2`; revisão de código `54475baa1bb1e496f9a7f43f6f391818a8de059e`. Developer diretamente por autorização do responsável. Provider elegível: nenhum; sem IA/login/migration/deploy/push. Escopo somente UI.

## Objetivo e funcionamento

Seção IA e contas mostra lista compacta com nome da conta, provider, habilitação, estado e indicação Configurar. Linha inteira é botão acessível por teclado. Adicionar conta abre modal com defaults existentes sem inserir linha antes de salvar. Nenhum formulário de conta fica expandido na listagem.

Dialog nativo recebe título via aria-labelledby e é aberto por showModal, com foco modal e fundo inerte pelo browser. Campos e ações anteriores foram movidos para o modal. Edição usa clone local; Cancelar, fechar e Escape descartam mudanças de campos. Salvar conta usa a API versionada existente para persistir configuração corrente (incluindo demais alterações pendentes no painel), preserva limpeza de atribuições/modelos/alternativas e fecha somente após sucesso. Erro mantém editor aberto e dados digitados. Durante salvamento controles e cancelamento ficam bloqueados para evitar submissão concorrente.

Verificação e conexão de assinatura permanecem explícitas e operam sobre conta persistida; adicionar conta não inicia essas ações. Conta nova não verifica/conecta/remove antes de estar salva. Remover conta existente conserva o fluxo anterior de rascunho: sai do modal, tira linha/referências e informa que Salvar alterações confirma a remoção. Limite de 20 contas preservado. CSS acomoda celular, scroll do modal e estados hover/focus da lista.

## Checks e evidências

`npm run typecheck -w @le-fabrique/web`, `npm run test -w @le-fabrique/web` (33 testes existentes passaram), `npm run build -w @le-fabrique/web`, `npm run lint` e `git diff --check` passaram. Lint tem só o warning anterior optional chaining em run-delivery.service.ts. Primeira checagem de estilos identificou warning novo de especificidade descendente; corrigido com classe específica na descrição da conta e revalidado. Nenhuma correção funcional exigida. Logs efêmeros `/tmp/fac-013-{typecheck,tests,build,lint}.log`.

Diff sanitizado: `git show 54475baa1bb1e496f9a7f43f6f391818a8de059e -- apps/web/src/SettingsPanel.tsx apps/web/src/styles.css`; apenas UI, sem dados de contas/tokens. Testes existentes cobrem view models/contratos relacionados; não foi executado teste de interação do novo modal em browser real, screenshot ou teste de persistência com conta real. Não se afirma que testes estáticos comprovam foco/Escape, que dependem do browser nativo.

## Teste manual e limitações

Com npm run dev ativo, recarregar Configurações. Conferir lista compacta; abrir uma conta com clique e Enter; editar/Cancelar e reabrir para confirmar descarte; adicionar/Cancelar para confirmar ausência de nova linha; adicionar/Salvar e recarregar para conferir persistência. Testar Escape, Tab/foco, nome longo e viewport estreito. API indisponível deve manter modal aberto e apresentar mensagem de falha no painel. Outras pendências de schema/writer relatadas em OPS-008/009 permanecem separadas; este incremento não atualiza banco nem libera execução.

Rollback em ticket autorizado: reverter commit de UI mantendo configuração persistida. Não há migration ou alteração de contrato. Docs configuração/planejamento, README raiz, índice, backlog/changelog e lesson de modais atualizados. Aceite humano da revisão exata e validação manual do visual pendentes; não DONE.
