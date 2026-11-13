# ADR-0046 — Orientação de testes e E2E por impacto, com evidência presa à revisão

- **Estado:** ACEITA · **Ticket:** [T14-04](../NEW%20MVP/03-BACKLOG.md#t14-04) · **Épico:** E14 · **Relacionadas:** [ADR-0040](0040-checks-baseline-e-classificacao-de-falhas.md), [ADR-0043](0043-classificacao-de-risco-por-pisos-deterministicos.md), [ADR-0008](0008-sandbox-por-feature-em-containers.md)

## Decisão
1. **Orientação de testes:** para cada arquivo de código alterado (Go, TS/TSX/JS, Python), o diff precisa trazer um teste criado ou atualizado — Go: um `*_test.go` no mesmo diretório; TS/JS: o `.test`/`.spec` do arquivo; Python: `test_<nome>.py`. Documentação, migrations, scripts, `cmd/` e os próprios testes não pedem teste. A falta é listada (arquivo e teste esperado); em risco **alto** bloqueia, em médio e baixo é só orientação.
2. **E2E por impacto** (`Decide`): risco baixo não exige; médio exige quando a mudança toca a UI (caminhos configuráveis, padrão `apps/web/**`); alto exige sempre; nível desconhecido é tratado como alto.
3. **Jornada em ambiente isolado:** `Store.Run` recusa um ambiente que não se declare **isolado** (`ErrNotIsolated`; o sandbox é isolado quando o perfil não tem rede) antes de executar qualquer coisa. A jornada roda pelo mesmo executor e registro dos checks (T14-01), com o nome `e2e:<jornada>`, e a evidência guarda jornada, **fixtures** (nomes, sem segredo), navegador, e as **capturas** (arquivos relativos ao workspace, até 20 de 2 MiB) com SHA-256, reconferido na leitura.
4. **Evidência presa à revisão:** o `Gate` confere, por jornada configurada, a evidência da **mesma revisão**: aprovada, falha, ausente ou justificada. Outra revisão não herda nada. Falha e ausência bloqueiam.
5. **Caso inaplicável com justificativa e evidência alternativa** (`Waive`): assunto `tests` ou `e2e:<jornada>` (inclui `e2e:sem-jornada` para projeto sem jornada de usuário testável); justificativa de 20+ caracteres; e uma evidência alternativa que **precisa existir para a mesma revisão** — um check **aprovado**, uma captura, ou uma nota de revisão (15+ caracteres). Inexistente, de outra revisão ou com falha: `ErrNoAlt`. A justificativa vale só para a revisão em que foi dada.

## Limites
- **Navegador real NOT_RUN:** nenhuma imagem de sandbox traz navegador (Playwright/Chromium); a execução real no Docker isolado foi provada com uma jornada simulada (comando, arquivo capturado, hash). A jornada de usuário real do painel precisa de uma imagem com navegador e do app subindo dentro do sandbox, e fica para quem configurar o projeto.
- A heurística de testes é por nome e diretório (um teste no diretório Go pode não cobrir o arquivo alterado); a revisão humana/independente (T14-03) cobre o que ela não vê.
- Não há comandos E2E vindos do perfil do projeto, rotas nem tela; as jornadas são informadas a `Gate`.
