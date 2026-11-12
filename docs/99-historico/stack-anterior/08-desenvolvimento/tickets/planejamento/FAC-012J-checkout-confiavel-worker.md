# FAC-012J — Checkout confiável no worker

Status: AWAITING_HUMAN

## Objetivo

Materializar no worker, fora do sandbox do código, um checkout por execução no root operacional configurado, vinculado ao projeto/URL/SHA imutável do job. Não usar caminhos recebidos do job nem executar hooks/filtros/conteúdo de repositório durante preparação.

## Critérios de aceite

1. Configuração de runtime valida root absoluto e allowlist explícita de hosts Git; ausência/inconsistência falha fechada.
2. URL do job deve ser HTTPS, sem userinfo/query/fragment, e o hostname precisa corresponder exatamente a um host permitido; IDs usados nos caminhos são UUIDs validados.
3. Credencial é obtida apenas sob demanda pelo `gh` oficial autenticado no host do repositório e usada em memória para transferência; token não entra em argv, disco, logs, contrato de controle ou ambiente do workflow/sandbox.
4. Clone/fetch/checkout invoca `git` com argv e cwd fixos (`shell:false`), desabilita configurações globais/sistêmicas, hooks, protocolo local/ext, LFS e submódulos, valida o commit exato e checkout detached.
5. Checkout fica em diretório temporário 0700 sob o root, é publicado por rename atômico para path derivado dos UUIDs e nunca sobrescreve destino existente; falhas limpam apenas o temporário criado pelo manager.
6. Limites de tempo/output encerram a árvore de processos e não retornam conteúdo bruto de stderr/stdout. Testes provam host/URL/sha/path gates, comandos/ambiente, sucesso, colisão, cancelamento/timeout e limpeza sem rede real.
7. Nenhuma integração com consumer/workflow, migration, provider de IA, checkout remoto, credencial ou serviço real durante desenvolvimento; rollout permanece explicitamente desligado.
8. Typecheck, lint, testes, build e `git diff --check` passam.

## Baseline, caminhos e limites

Base `191a5377ddde1d9a15393c8a49cdbe36e98741da`; branch `feat/fac-012j-trusted-checkout`, worktree reutilizado `/home/vinicius/le-fabrique-fac-012f`. Alvos: config/serviço novos em `apps/worker/src`, testes, instruções de configuração/operação e lesson de checkout. Sem dependência nova ou migration. Limite de execução: 5 minutos e 64 KiB de saída por subprocesso.

## Provider elegível e gastos

Nenhum provider de IA será usado. Testes usam runners sintéticos; nenhum token GitHub real, rede, banco/fila, projeto externo ou VPS será acessado. O futuro runtime chama `gh auth token` somente se o host tiver conta oficial elegível, não guarda a saída nem a transfere ao controle.

## Fora de escopo

Ligar o consumer, integração lease/fencing/workflow, sandbox do Developer CLI, teste operacional sob identidade systemd e prova da disponibilidade segura de keyring em VPS. Esses gates continuam impedindo execução real até tickets separados e aceitos.

## Implementação e evidências

Implementado em `apps/worker/src/repository-checkout.ts`, configuração/runtime e testes sintéticos. Hosts entram por `WORKER_REPOSITORY_HOSTS` e o root absoluto por `WORKER_CHECKOUT_ROOT`; sem ambos a preparação falha fechada. `gh auth status` precisa confirmar `tokenSource=keyring` antes da chamada efêmera `gh auth token`. Git recebe credencial em cabeçalho HTTP restrito ao host por variável de ambiente interna, sem argv/remote URL/log; usa `shell:false`, config isolada e comandos fixos. O manager reserva path por lock exclusivo, cria clone 0700, verifica objeto e HEAD exatos, prova estado detached e publica por rename.

Revisão do código: `c587569c484387e9f821a6a4c0f4dcfe81127c36`.

Em 2026-10-02 passaram: worker 80 testes; suíte total 203 (21 contracts, 40 runtime, 55 API, 80 worker, 5 web e 2 launcher); `npm run typecheck`, `npm run lint`, `npm run build`, `git diff --check`. Nenhum comando real `git`/`gh`, host remoto, keyring, banco, fila, provider ou consumer foi usado; todos os processos de teste de checkout usam runner sintético. A revisão exata aguarda aceite humano.
