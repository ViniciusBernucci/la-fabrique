# FAC-035 — Nome e cargo dos agentes de IA

2026-10-08, Europe/Berlin. Domínios: configuração, painel, contratos. IMPLEMENTADO / AWAITING_HUMAN; código commitado em bc38715faffa5c1f3cdad82dca43822508fd0b7b. Integração local à developer autorizada pelo responsável em seguida; sem push ou deploy. Base 97869109e9d8c74bed5635b0cb3a0c6eaaca6fe9; branch feat/fac-035-agent-names, worktree /home/vinicius/le-fabrique-fac-035. [Ticket READY autorizado](../../08-desenvolvimento/tickets/configuracao/FAC-035-nomes-agentes.md).

## Objetivo, problema e critérios

Permitir apelidar os agentes em Configurações → Equipes e apresentar sua identificação anterior como cargo. ATENDIDO por código e testes: nome separado do cargo, compatibilidade de leitura legada, limite/trim/remoção, gravação versionada e resumo acessível. NÃO VALIDADO em browser: edição/salvar/cancelar e falha de rede do modal. Revisão independente e aceite exato pendentes.

## Solução e comportamento

Campo Nome do agente nos modais de pré-configurados e personalizados. Exemplo didático: Alex / Cargo: Developer. O cargo predefinido é somente leitura; o personalizado continua editável pelo campo Cargo na empresa. Listas, título das informações e ações acessíveis usam o apelido com fallback para o cargo. Nome opcional de até 100 caracteres após trim; limpar retorna ao cargo. Salvar usa rascunho e PUT versionado existente; cancelar descarta o rascunho. Falha mantém modal/erro existente.

## Arquivos e decisões

[Contratos Zod](../../../packages/contracts/src/index.ts): nickname opcional em agentAssignmentSchema/digitalAgentSchema. [SettingsPanel](../../../apps/web/src/SettingsPanel.tsx), [TeamsPanel](../../../apps/web/src/TeamsPanel.tsx) e [PresetAgents](../../../apps/web/src/PresetAgents.tsx): edição e apresentação. Testes de contratos, serviço settings e resumo React cobrem as regras. ADR novo não necessário: mantém stack/fluxo/persistência.

## Dados, API, eventos e configuração

FactorySettings.configuration JSON versionado mantém nickname. Nenhuma migration, rota ou evento novo. Role/IDs/contas/modelos/permissões/skills/ativação continuam sendo os campos operacionais. Registros anteriores são válidos. Clientes com schema anterior strict não aceitam nickname; atualização coordenada de API/worker/painel é necessária. Nenhum banco ativo foi alterado. [Contrato e compatibilidade](../../05-contratos/schemas/00-entidades.md#nomes-de-agentes-fac-035).

## Diff e artefatos

[Código e testes em patch](evidencias/FAC-035/codigo.patch) e [hashes](evidencias/FAC-035/artefatos.sha256). Docs e ticket permanecem no diff/untracked da worktree, recuperáveis sem tocar o workspace original. [Fontes lidas e hashes](evidencias/FAC-035/fontes.sha256): políticas abertas integralmente; também foram lidos Manual/README/piloto/prompt inicial/guia/ADR-003, perfil/skill local, ticket FAC-026, módulo/feature/contrato/matriz/checklist/template. Não se carregou todo o histórico. Autoload/hierarquia automática não verificados; leitura explícita foi usada. Codex instalado: 0.161.0, --help/--version consultados, sem nova sessão cliente.

## Testes e evidências

Revisão: diff sobre 9786910 desta worktree. [Registro dos checks](evidencias/FAC-035/checks.txt).

| Critério | Comando/procedimento | Resultado |
|---|---|---|
| Compatibilidade/validação | npm run test -w @le-fabrique/contracts | PASS, 53 testes |
| Resumo/acessibilidade/regressões web | npm run test -w @le-fabrique/web | PASS, 64 testes |
| Persistência versionada | npm run test -w @le-fabrique/api -- src/settings/settings.service.spec.ts | PASS, 8 testes com Prisma simulado |
| Roteamento existente | npm run test -w @le-fabrique/worker -- src/configured-agent-router.spec.ts | PASS, 15 testes |
| Tipos | npm run typecheck | PASS após correção |
| Build | npm run build | PASS, cinco workspaces |
| Documentação | python3 scripts/validar-documentacao.py | PASS, zero erros |
| Browser/HTTP/DB real/revisão independente | Não executados nesta sessão | NOT_RUN |

O primeiro typecheck detectou acesso indevido a nickname numa skill por substituição textual ampla; corrigido removendo apenas essa alteração. Não houve mudança de contrato de skills. Lint retorna zero com aviso pré-existente em run-delivery.service.ts fora do escopo. npm ci --ignore-scripts instalou dependências do lock; prisma generate gerou apenas cliente local, sem migration/conexão ao DB.

## Segurança, riscos e limitações

Nome é metadado, renderizado como texto React; não concede autoridade. Nenhum segredo/provider/auth/config global acessado ou alterado. Outras sessões Codex têm cwd no repositório original e quiescência desconhecida: implementação somente em worktree nova exclusiva, sem segundo writer lá. Não há comprovação de experiência visual ou escrita PostgreSQL real. Sem expansão para tarefas/onboarding.

## Rollback

Antes de voltar ao schema anterior, remover nickname de assignments/digitalAgents do JSON pela versão ainda compatível, usando a operação administrativa autorizada e preservando os demais dados. Depois reverter somente o patch deste ticket. Não ensaiado em banco/produção; nenhuma migration para reverter. Worktree isolada pode ser descartada após preservar patch/docs, sem afetar developer.

## Documentação atualizada

[Módulo](../../03-modulos/configuracao/00-README.md#nome-e-cargo-dos-agentes--fac-035), [feature](../../04-features/01-control-settings.md#nome-dos-agentes--fac-035), contrato de entidades, [índice](../../02-INDEX.md), [changelog](../../04-CHANGELOG.md), [backlog](../../08-desenvolvimento/09-backlog.md) e [matriz](../../08-desenvolvimento/05-matriz-cobertura.md). Nenhum capítulo corrente novo: mapa de ordem não muda.

## Lessons e uso de IA

Sem lesson nova: aplicação de rascunho e compatibilidade já documentados, sem conceito novo. Provider da sessão Codex; versão CLI 0.161.0 observada. Modelo efetivo/autenticação/uso/cota/custos fixos não comprovados; desconhecidos. Nenhum cliente IA novo, handoff, subagente ou gasto extra habilitado. Uma rodada de correção do typecheck.

## Pendências e aceite

Revisão independente, validação browser e aceite humano do código exato permanecem pendentes. Sem DONE, push ou deploy.

## Adendo — fechamento e integração autorizados

Em 2026-10-08 o responsável pediu explicitamente juntar a worktree na developer, autorizando commit e integração locais. A developer estava limpa e na base 9786910, permitindo fast-forward sem conflitos. Código registrado em bc38715faffa5c1f3cdad82dca43822508fd0b7b; documentação/evidências em commit posterior. O diff funcional não mudou desde os checks registrados. Fontes e artefatos conferidos por sha256sum; skill local fechar-entrega e referência registro-local lidas. Integração não equivale a revisão independente ou implantação. A autorização atual permite esta integração no workspace original, preservando outras sessões e sem lançar outro executor.
