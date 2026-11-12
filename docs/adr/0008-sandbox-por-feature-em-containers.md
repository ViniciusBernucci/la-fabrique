# ADR-0008 — Sandbox efêmero por feature em containers

- **Estado:** ACEITA · **Ticket:** [T11-01](../NEW%20MVP/03-BACKLOG.md#t11-01) · **Épico:** E11 · **Decisões:** D02, D30–31, D35, D54, D70, D77–78, D84

## Decisão
1. **Um container + um volume (+ uma rede, se pedida) por projeto/feature**, criados e destruídos pelo supervisor (`internal/execution`, sobre o CLI do Docker). IDs vêm de aleatoriedade; projeto e feature entram só como *labels* (validados por regex), nunca em nomes ou comandos. Falha no meio desfaz o que foi criado; `Destroy` é idempotente.
2. **Endurecimento fixo:** usuário `10001` sem privilégios, rootfs somente leitura, `--cap-drop ALL`, `no-new-privileges`, `--init`, teto de processos, memória (sem swap) e CPU, `/tmp` em tmpfs, **nenhuma montagem do host** e nenhum socket do runtime. O código do projeto não recebe segredos (por variável de ambiente ou argumento).
3. **Disco efêmero com teto:** o workspace é um volume `tmpfs` com tamanho máximo e dono sem privilégios. Consequência: **fica em memória** e some com o sandbox; durabilidade e checkpoints são do E12.
4. **Rede:** padrão `none` (só loopback). `isolated` cria uma rede interna própria do sandbox (sem internet, sem outros sandboxes, sem a rede do stack), útil a serviços auxiliares do mesmo sandbox. **Saída controlada para registros/Git ainda não existe** (exige um intermediário com allowlist; E15/T11-02): hoje o código chega ao workspace por extração de um commit exportado pelo supervisor (`Seed`) e as dependências precisam estar na imagem do perfil.
5. **Perfis iniciais** `base`, `go`, `node`, com as imagens já fixadas e quotas **provisórias** (sem medida da VPS, T02-01).
6. Só o supervisor fala com o runtime. O container do worker **não** recebe o socket do Docker nesta entrega; decidir como o worker alcança o runtime (socket proxy restrito ou execução no host) é pendência explícita antes de ligar o scheduler.

## Consequências e limites
- Isolamento do host: validado com Docker Desktop (VM), onde o host não está na rede do container. Em Linux, serviços do host ligados a `0.0.0.0` podem ser alcançáveis pelo gateway da bridge; a mitigação (regras na cadeia `DOCKER-USER`/firewall) **não foi provada** e consta como pendência de homologação.
- O estouro do tempo em `Exec` não encerra os processos dentro do container; a supervisão da árvore é o T11-03.
