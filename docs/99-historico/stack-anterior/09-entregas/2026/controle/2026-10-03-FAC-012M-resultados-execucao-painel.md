# FAC-012M — Resultados de execução no painel

Data: 2026-10-03. Status: AWAITING_HUMAN, sem aceite humano.
Baseline: `d24245a83e8686373f237a2a7fcb2db6cba69746`; ticket READY: `1921282`.
Branch/worktree: `feat/fac-012m-execution-results`, `/home/vinicius/le-fabrique-fac-012m`.

## Checkpoint de validação

Implementação local em andamento: contratos de relatório minimizado, persistência por tentativa/fencing, consultas administrativas, metadados observados do runtime e painel de resultados. Migration aditiva apenas versionada, não aplicada. Nenhum serviço/provider/banco/fila/piloto usado. Prisma Client local foi gerado; builds contracts/runtime e typecheck API/worker passaram antes da revisão final.

Duas rodadas de lint da UI foram interrompidas antes dos checks gerais: a primeira identificou atributo ARIA em div genérica, dependência de refresh sem leitura e keys por índice; a segunda recusou role=group em div por existir elemento semântico nativo. Checkpoint/diagnóstico registrado antes de prosseguir: trocar o agrupador por fieldset/legend, preservar seleção por botão e usar identidades estáveis. Não há diagnóstico de regressão do backend nem evidência de aprovação do MVP; testes completos ainda pendentes neste checkpoint.

## Entrega e funcionamento

Código: `14ae5baf9fe7b7ba98934ac581e4788e794da56c`. API recebe relatório por tentativa com WorkerAuthGuard, worker/fencing atuais e transação serializável. Digest SHA-256 torna repetição idêntica idempotente; divergência é recusada. Gravar evidência não altera lease, stoppedConfirmed ou estado do writer. Consumer envia resultado antes de checkpoint/complete; falha de persistência impede conclusão.

GET administrativo `/api/runs?projectId=UUID` retorna até 50 runs; `/api/runs/:runId` retorna até 50 tentativas recentes. DTOs explícitos não expõem registros internos completos. Painel consulta a cada dez segundos, sequencialmente, cancela/desconsidera respostas antigas e mostra checks, revisão, diagnóstico, hashes/contagens dos snapshots e observações por chamada. Instalação configurada não comprova identidade autenticada. Modelos efetivos/uso desconhecidos permanecem null, sem estimativas. Não há aprovação, retomada ou publicação automática.

Contrato estrito limita relatório a 64 KiB e rejeita workspace, paths de artefatos, prompts, stdout/stderr, sessões e campos extras. Textos selecionados recebem minimização e remoção de padrões conhecidos de segredos/caminhos; isso não é garantia de detecção de todo segredo arbitrário. React renderiza texto escapado, não HTML do modelo.

Migration aditiva cria `attempts.result`/`result_digest`; somente versionada, não aplicada. Configuração de contas/modelos permanece no software, sem piloto ou credenciais hardcoded.

## Checks e diff sanitizado

`npm ci` e `npm run db:generate` locais concluídos, sem conexão ao banco. `npm run typecheck`, `npm test`, `npm run build` passaram: 240 testes (launcher 2, contracts 24, runtime 44, API 61, worker 101, web 8). Após esses checks, ajuste textual da UI foi validado com os oito testes web. `npm run lint` passou em 162 arquivos e `git diff --check` passou. Prisma validate passou com URL sintética sem conexão; primeira tentativa de localizar binário na raiz falhou com exit 127, corrigida usando npm exec no workspace API.

Após o checkpoint acima, fieldset/legend resolveu o diagnóstico sem ampliar permissões. Lint final encontrou apenas formatação encadeada não estabilizada no primeiro passe; formatador corrigiu e lint passou. Testes cobrem fence/digest, ausência de mutações de writer, campos indevidos, redaction/unknown, persistência antes de conclusão e falha de persistência, polling sem overlap/resposta tardia e renderização escapada. Não foi executado E2E de navegador, HTTP/banco real ou autenticação de clientes.

Diff sanitizado reproduzível: `git diff --stat 1921282 14ae5baf9fe7b7ba98934ac581e4788e794da56c` (20 arquivos, 1018 inserções, 10 remoções). Mudanças: contratos públicos/testes; dois campos/migration; serviço/controllers/registro API; observações/cliente/processador worker; painel/cliente/polling/CSS/testes web. Nenhum segredo ou fixture de conta real incluído. Documentação/lessons são commit posterior separado.

## Limitações, rollback e aceite

Não entrega diff completo/download de artefatos, recuperação após interrupção/persistência falha, handoff, isolamento real de contas nem comandos de run. Gate real segue false; nenhum provider, fila, banco, serviço, login, piloto, push ou deploy foi iniciado. FULL MVP permanece incompleto; migração e operação são manuais. Revisão humana deve aceitar SHA exato antes de DONE.

Rollback: reverter o commit de código com writer parado; migration ainda não aplicada não requer rollback de banco. Se aplicada futuramente, preservar os relatórios; não remover colunas com dados automaticamente. Integração local na developer autorizada pelo responsável, condicionada à árvore limpa e ancestry; branches são preservadas após remover worktree limpa. Lessons atualizadas sobre evidência imutável sem liberar writer e DTO minimizado/unknown.
