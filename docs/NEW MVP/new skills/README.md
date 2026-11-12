# Skills para Codex, Claude e Antigravity

O conteúdo original permanece em `skills/`, sem alterações. As pastas abaixo
contêm cópias completas e idênticas, incluindo `_shared`, referências, scripts e
metadados existentes.

| Agente | Pasta de descoberta no projeto |
| --- | --- |
| Codex | `.agents/skills/` |
| Claude Code | `.claude/skills/` |
| Antigravity (App, CLI e IDE) | `.agents/skills/` |
| Antigravity (compatibilidade legada) | `.agent/skills/` |

Abra este repositório como projeto no agente desejado. Para usar em outro projeto,
copie a pasta de descoberta correspondente para a raiz dele, preservando `_shared`
e os demais recursos. Codex e Antigravity compartilham a mesma pasta padrão.

## Sincronização

Mantenha as skills em `skills/`. Depois de uma alteração futura, atualize as
cópias com Node.js 18 ou superior:

```sh
node scripts/sincronizar-skills.mjs --apply
node scripts/sincronizar-skills.mjs --check
```

`--check` compara os bytes de todos os arquivos e retorna código 1 se encontrar
arquivos ausentes, diferentes ou extras. `--apply` copia os arquivos da fonte,
substituindo cópias diferentes; arquivos extras são apontados e preservados.
O script não altera `skills/`, configurações globais ou outros projetos.

A organização prepara a descoberta das skills. Ferramentas, integrações e
permissões exigidas pelos fluxos continuam dependendo do ambiente de cada agente.
Os scripts originais de sincronização Intranet foram preservados; para distribuir
este repositório, use o comando acima.

Padrões consultados: [Codex](https://developers.openai.com/blog/skills-agents-sdk),
[Claude Code](https://code.claude.com/docs/en/skills) e
[Antigravity](https://antigravity.google/docs/skills).

## Modelos mínimos de regras por projeto

Use [modelos/CLAUDE.md](modelos/CLAUDE.md) e [modelos/AGENTS.md](modelos/AGENTS.md)
como blocos para incorporar às regras de outro projeto. Se o arquivo já existir,
acrescente o conteúdo sem substituir as regras atuais. Os modelos contêm o mesmo
contrato essencial e ficam separados para não alterar as regras ativas desta distribuição.

Veja [como usar e o que configurar por projeto](modelos/README.md). A instalação das
skills continua igual; os modelos não instalam o Contexto nem ferramentas externas.
