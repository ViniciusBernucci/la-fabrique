> Leitura: [← Anterior](00-README.md) · [Índice didático](../../02-INDEX.md) · [Próximo assunto →](../runtime-guard/00-README.md)

> Direção posterior do pacote, PLANEJADO. Código atual no [README do módulo](00-README.md); nenhum gate operacional liberado por este texto.

# Sandboxes efêmeros e isolamento do host

Data: 2026-10-06 (America/Sao_Paulo). Status da arquitetura e controles: **PLANEJADO**. Implementação e eficácia: **NÃO VERIFICADAS**. Este documento especifica trabalho futuro; não comprova instalação, configuração ou execução.

## Definição e unidade

Sandbox é ambiente restrito para código, comandos, builds, testes e navegador potencialmente hostis. A unidade é attempt/etapa de execução; várias tentativas de um run geram sandboxes distintos. Developer e reviewer têm sessões/sandboxes diferentes; reviewer sem escrita no código analisado. Código pode arruinar seu workspace autorizado, mas não tem capacidade prevista de atravessar a fronteira.

Executor confiável materializa snapshot da revisão concedida, sem `.env` real, credenciais Git, home, links para host ou repositórios vizinhos. Worktree com `.git` apontando para diretório comum do host NÃO é montada diretamente. Usar clone/snapshot independente sem alternates, hooks, helpers ou object store compartilhado; Git remoto e publicação são tarefas de integração confiável, sem token no sandbox.

## Perfil mínimo obrigatório

| Superfície | Política planejada |
|---|---|
| Identidade | não root; UID remapeado/user namespace quando compatível; sem sudo/setuid efetivo |
| Namespaces | mount/PID/IPC/UTS/network próprios; sem host PID/network/IPC |
| Privilégios | privileged=false; drop ALL Linux capabilities; no-new-privileges |
| Kernel | seccomp e AppArmor/SELinux verificados; não unconfined; bloquear ptrace/mount/bpf conforme perfil |
| Filesystem | rootfs somente leitura; /workspace RW autorizado; /context e /instructions RO; /output RW limitado; /tmp tmpfs limitado |
| Mounts proibidos | /, /home host, /proc host, /sys host, /etc host, auth, SSH, volumes do controle, Docker/containerd sockets |
| Recursos | CPU/RAM/PIDs/I/O/disco/tempo agregados por job, incluindo browser e serviços sintéticos |
| Rede | sem saída/entrada por padrão; perfil explícito, nunca rede do banco/controle |
| Toolchain | imagem aprovada fixada por digest; não escolher imagem/entrypoint livre da IA |

Pseudoarquivos do próprio sandbox podem existir; `/proc` não deve expor processos/ambiente do host. Não proibir literalmente toda leitura fora de /workspace: bibliotecas/toolchain aprovadas são necessárias, mas diretórios privados do host não existem nesse namespace. Garantir que core dumps/debug não revelem auth; desabilitar dumps nos runtimes autenticados.

## Lifecycle

`PREPARING → READY → EXECUTING → STOPPING → QUIESCENT → COLLECTING → DESTROYED`. Falha de configuração impede READY. Parar novas ferramentas e revogar capabilities antes de encerrar; matar árvore/cgroup inteiro, verificar processos e serviços auxiliares parados. Coletar após quiescência, validar e persistir hashes antes de apagar. Cleanup idempotente só de recursos pertencentes à tentativa. Recursos órfãos após reboot ficam em quarentena até reconciliar ownership; nenhum novo writer enquanto morte anterior não estiver comprovada.

Serviços sintéticos de integração usam namespace/rede privados por attempt e dados sintéticos, sem senha de banco da fábrica. Não entregar socket Docker para subir serviços; helper provisiona perfis cadastrados. Browser sem sessão pessoal, extensões ou acesso à rede administrativa; política de saída idêntica à da etapa.

## Paths e coleta

Validação pelo broker por descritor relativo à raiz autorizada e operações resistentes a symlink/TOCTOU, ou equivalente de segurança comprovada. Normalização textual não basta. Rejeitar `..`, paths absolutos, NUL, links para fora, hardlinks externos, devices/FIFO/sockets e arquivos especiais. Check/export também precisam impedir ZIP/TAR slip, links maliciosos, bomba de descompressão e arquivo muito grande. Limitar número, tamanho e profundidade; buscar apenas output registrado, sem path host escolhido pela IA.

Cache privado por tenant/projeto, RO quando reutilizado; não compartilhar node_modules writable nem dependência adulterável por A com B. Cache público confiável só conteúdo verificado/imutável, sem artefatos privados ou segredo. Restore de checkpoint usa nova sandbox e repete validação de paths/hashes.

## Limites de garantia

Container compartilha kernel; [documentação Docker](https://docs.docker.com/engine/security/) descreve superfície do daemon e risco de privilégios/mounts. Perfil container só para piloto próprio/sintético sob risco aceito. Avaliar [gVisor](https://gvisor.dev/docs/architecture_guide/intro/) ou microVM para tenants hostis; validar compatibilidade, virtualização disponível e overhead na VPS contratada. Não alegar que gVisor elimina todos os riscos. Sem perfil adequado, não admitir execução arbitrária de terceiros.


## Origem desta edição

[Versão original preservada](../../99-historico/originais/output/le-fabrique-multitenant/documentacoes/sandbox/README.md). Migração editorial de paths em 2026-10-06; conteúdo de engenharia continua proposto.

## Localização, relações e validação

Na origem do pacote o código estava ausente; AS-IS atual no README; nenhuma classe/entrypoint ou teste foi comprovado. Consulte [ADRs](../../06-decisoes/00-README.md), [features](../../04-features/00-README.md) e [SEC-01–14](../../08-desenvolvimento/12-testes-seguranca.md). Estados/contratos citados são planejados; registrar código/revisão/checks por controle quando implementado.
