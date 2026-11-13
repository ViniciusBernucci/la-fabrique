# ADR-0055 — Ensaios de falha e segurança e relatório de aceite

- **Estado:** ACEITA · **Tickets:** [T17-02](../NEW%20MVP/03-BACKLOG.md#t17-02), [T17-03](../NEW%20MVP/03-BACKLOG.md#t17-03), [T17-04](../NEW%20MVP/03-BACKLOG.md#t17-04) · **Épico:** E17 · **Relacionadas:** [ADR-0041](0041-matriz-de-evidencias-derivada.md)

## Decisão
1. **Ensaio de feature ponta a ponta nos dobles (T17-02):** um teste de integração percorre a feature da Draft PR ao pós-merge com PostgreSQL real e GitHub/origem simulados e confere que PR, commits, checks, dependência e memória ficam ligados à mesma revisão integrada, com repetição idempotente. É evidência de software; **não** substitui o ciclo no ambiente e no projeto reais, que segue NOT_RUN até as decisões e acessos F01–F12.
2. **Catálogo de ensaios gerado (T17-03):** `internal/acceptance` mantém o catálogo (ID, área, esperado e testes) e `make ensaios` (`cmd/ensaios`) roda cada cenário e gera `docs/aceite/03-ensaios-de-falha-e-seguranca.md` com esperado, observado, evidência e revisão. Regras: o catálogo é conferido contra o código (teste inexistente, ID repetido ou área obrigatória sem cenário quebram a suíte); sem `TEST_DATABASE_URL` os cenários com banco ficam NOT_RUN, nunca PASS; falha vira FAIL e saída 1, e volta ao ticket de origem para ser retestada. A coluna de revisão é NOT_RUN até a revisão independente.
3. **Relatório de aceite derivado (T17-04):** `make matriz` também gera `docs/aceite/04-relatorio-de-aceite.md` a partir do checklist: situação por épico, todo ticket aberto com os limites que ele mesmo declarou, artefatos do aceite e a decisão do responsável em branco. Há teste de deriva. O relatório só mostra a situação ("NÃO FECHADO" enquanto houver ticket parcial ou pendente) e nunca declara o aceite: mesmo com tudo `[x]`, a decisão e a evidência do projeto real continuam do responsável.

## Limites
- O observado do relatório de ensaios é o da máquina e da revisão em que `make ensaios` rodou (a revisão aparece no relatório, com "+alterações locais" quando houver); é preciso rodar de novo para atualizar.
- Cada cenário agrupa testes existentes; não há ensaios novos de caos além dos já presentes na suíte, e nada roda contra serviços reais.
- O aceite integral do MVP não está fechado: depende do projeto real, dos acessos e da decisão do responsável.
