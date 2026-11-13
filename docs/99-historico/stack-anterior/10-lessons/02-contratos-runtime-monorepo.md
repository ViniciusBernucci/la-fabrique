> Leitura: [← Anterior](01-definicao-projeto-configuravel.md) · [Índice didático](../02-INDEX.md) · [Próximo →](03-outbox-idempotencia.md)

# Contratos de runtime no monorepo TypeScript

FAC-012P mostra que schema/tipo não prova bytes: `verifyExecutionArtifact` recebe decoder/hash neutros e cruza manifesto/patch/untracked/contagem/tamanho, após DTO estrito/base64 canônico/limite 64 KiB. Hashes são integridade, não prova de ausência de segredos; scanner bloqueia padrões conhecidos, sem alterar patch silenciosamente. Worker verifica arquivo real e API verifica transporte antes de persistir. Painel só escapa texto/download, sem servidor no bundle.

No FAC-012M, o DTO público não reutiliza o resultado privado inteiro: remove workspace/paths, reduz untracked a contagem, limita JSON a 64 KiB e usa nested strict. Runtime observações null não viram estimativas de modelo/uso. React escapa texto de IA e polling sequencial aborta respostas antigas. Redaction de padrões conhecidos complementa minimização, sem prometer reconhecer todo segredo arbitrário.

## Conceito aplicado

Tipos TypeScript desaparecem na execução. Por isso, compartilhar apenas interfaces entre painel, API e worker não valida dados recebidos por HTTP, fila ou variáveis de ambiente. O bootstrap centraliza schemas Zod em `packages/contracts`; os tipos são inferidos desses schemas, mantendo a validação e a tipagem na mesma fonte.

## Exemplo do repositório

O schema de readiness define a resposta esperada da API e é consumido pelo painel. O schema de worker probe valida o payload antes de o processador BullMQ usá-lo. Assim, uma alteração incompatível falha no limite entre processos em vez de circular como um objeto TypeScript presumidamente seguro.

O pacote disponibiliza fonte ESM ao Vite para permitir tree-shaking e compila CommonJS para API e worker. Ele exporta somente contratos neutros: código Prisma, configuração de servidor, segredos e adapters permanecem fora do pacote compartilhado.

No FAC-011, objetos `.strict()` fazem parte da fronteira de seguranca: um campo `token` enviado junto a uma instalacao e rejeitado, em vez de ser silenciosamente persistido. A validacao cruzada tambem comprova que o modelo pertence ao catalogo da conta habilitada, modelos/funcoes nao se repetem e todas as funcoes aparecem. O servico aplica ainda uma regra contextual que o schema isolado nao conhece: `AVAILABLE` e `CONNECTED` nao podem ser promovidos pelo update administrativo, pois exigem evidencia do worker.

No FAC-012E, o worker recebe um DTO dedicado (`workerConfigurationSnapshotSchema`) em vez do registro Prisma. O endpoint interno retorna `{version, observedAt, configuration}`; versão `0` deixa explícito que defaults inativos ainda não foram persistidos. O GET usa `findUnique`, sem criar estado, e `WorkerAuthGuard`; segredos adicionais falham no parse estrito.

No FAC-012F, a decisão usa uma nova leitura para cada rota e devolve junto a versão/observação de settings; assim não há cache de modelo nem troca silenciosa quando a configuração muda. O router confere também que o adapter injetado tem o mesmo `name` do provider roteado. `REVIEWER` com `WORKSPACE_WRITE` é inválido em vez de ser corrigido silenciosamente.

No FAC-012G, não basta validar a rota em uma camada e depois deixar o workflow usar um adapter/modelo capturados antes: `DeveloperWorkflow` resolve novamente por função antes de cada invocação, inclusive correções, e contabiliza o provider retornado pelo router no `RuntimeGuard`. A rota Reviewer continua read-only; a falha não inicia o adapter. O enum de resultado recebe `RUNTIME_ROUTE_UNAVAILABLE` para representar esse bloqueio sem sobrecarregar erro de execução ou veredito inválido.

## Quando repetir

FAC-012W distingue orçamento raw, JSON/base64 e envelope HTTP: constants compartilhadas alimentam DTO, reader e parser. Ampliar só o schema deixaria o default HTTP 100 KiB bloquear artefatos; testes Nest reais aceitaram perto do teto e rejeitaram excesso com 413. Reader soma tamanhos antes de ler e limita buffers por tamanho esperado, pois stat seguido de readFile não impõe limite durante crescimento. Base64 usa roundtrip canônico sem regex recursiva em strings multi-megabyte.

Todo novo endpoint, evento de outbox, payload de fila ou checkpoint precisa de schema versionado na fronteira. Validação de runtime deve acontecer antes de persistir ou executar o dado. Interfaces internas sem entrada externa podem permanecer apenas em `packages/runtime`.

## Evidência

Os testes de `packages/contracts`, da API e do worker exercitam payloads válidos e rejeitam entradas inválidas. Ver também `documentacoes/operacao/2026-10-02-FAC-012E-configuracao-runtime-worker.md` para o snapshot interno do worker; a fundação original está em `documentacoes/infraestrutura/2026-09-30-FAC-000-bootstrap-typescript.md`.

Na integração entre worktrees, regenere artefatos derivados na ordem de dependência antes de tratar falhas de resolução como regressão. Aqui, `npm test` inicialmente carregou `dist` e Prisma Client defasados compartilhados entre worktrees; `npm run db:generate` e builds de contracts/runtime corrigiram o estado local sem tocar o banco. Em seguida, valide a árvore combinada e só remova worktrees limpas cujos SHAs sejam ancestrais do destino. Esse procedimento foi aplicado aos merges FAC-012D e E–J em `developer` pelo OPS-004.


FAC-023: coleções opcionais em FactoryConfiguration preservam configurações antigas sem introduzir defaults artificiais. superRefine valida referências únicas e skill do mesmo projeto; SettingsService consulta Project na transação porque existência no banco não é propriedade do payload isolado. Update versionado preserva concorrência. Rollback de schema estrito precisa tratar campos persistidos antes de retirar o contrato. [Evidências](../09-entregas/2026/configuracao/2026-10-05-FAC-023-agentes-skills-projeto.md).
