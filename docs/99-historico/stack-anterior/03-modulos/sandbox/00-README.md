> Leitura: [Índice didático](../../02-INDEX.md) · [Próximo →](01-direcao-proposta.md)

# Sandbox, worktrees e snapshots

IMPLEMENTADO no código base `a2cc5e0`; verificação operacional NÃO VERIFICADA nesta migração. Responsabilidade lógica, não nova classe/microserviço.

## Finalidade e limites

Confinar checks e preservar trabalho recuperável, distinguindo bytes de autoridade.

## Fluxo e comportamento

Worktree detached → systemd usuário/cgroups/namespaces → rootfs mínimo/pivot → check permitido → confirmar stop → snapshot Git patch/untracked → restore base limpa exata.

Exemplo didático: ticket com critério “validar um campo” só avança com a base/definição/checks aprovados; request inválido não vira comando autorizado. O exemplo não é execução desta migração.

## Estados, dados e regras

SandboxRunner checks diferente de perfis nativos CLI. /usr RO, workspace /mnt, sem host/home/controle/Docker socket/rede de checks; isolamento de kernel/identidade exige ensaio. Veja [estados por entidade](../../05-contratos/schemas/01-workflow.md) e [contratos canônicos](../../05-contratos/00-README.md).

## Falhas, recuperação e segurança

Payload/versão/base inválidos são recusados antes de efeito pertinente; auth/cota não autorizam fallback pago. Writer unknown permanece bloqueado; não repetir execução ou liberar lease por idade. Credenciais não entram em DTO/contexto/log/artefato; API não executa código do projeto. Limites de cada contrato e gates de provider continuam necessários.

## Código e evidência

[packages/runtime/src/sandbox-runner.ts](../../../packages/runtime/src/sandbox-runner.ts) e [packages/runtime/src/snapshot-manager.ts](../../../packages/runtime/src/snapshot-manager.ts). Testes versionados ao lado dessas implementações e evidências por FAC-007/012C/H/AA no [histórico](../../09-entregas/00-README.md). Não reexecutados aqui. [ADR-003 aceito](../../06-decisoes/ADR-003-stack-typescript.md) e [feature relacionada](../../04-features/04-recovery-operation.md).

## Limitações e direção posterior

[AS-IS versus alvo](../../02-arquitetura/01-as-is.md) separa controles atuais de tenancy/RLS/broker propostos. Presença de implementação/teste versionado não comprova instalação ou eficácia sob UID de serviço.

## Direção posterior completa

[Engenharia do perfil futuro](01-direcao-proposta.md) preserva detalhes de auth/broker/ciclo/negações/compatibilidade do pacote. Proposta não prova eficácia atual.

## Sequência de leitura deste capítulo

1. [direcao-proposta](01-direcao-proposta.md).
