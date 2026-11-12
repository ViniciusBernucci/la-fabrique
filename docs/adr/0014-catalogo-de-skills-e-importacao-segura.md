# ADR-0014 — Catálogo de skills e importação segura

- **Estado:** ACEITA · **Ticket:** [T08-02](../NEW%20MVP/03-BACKLOG.md#t08-02) · **Épico:** E08 · **Decisões:** D10–13, D74, D78–80

## Decisão
1. **Quatro modalidades** de skill (`executor`): `ai` (instruções do SKILL.md), `script` (comandos declarados), `deterministic` (passos `set`/`template`/`assert`, sem efeitos) e `hybrid` (IA + script + passos, com ao menos dois tipos). O manifesto vem de `skill.json` ou do cabeçalho do `SKILL.md` (só texto simples); coerência com o executor é validada.
2. **Importação é leitura, nunca execução.** Pacotes (`tar.gz` ou `zip`) e `SKILL.md` avulso são lidos na memória. Recusados: travessia de caminho (`..`, absoluto, barra invertida, unidade, `.git`), links simbólicos/físicos e entradas especiais, duplicados (sem diferenciar maiúsculas), acima de 200 arquivos, 1 MiB por arquivo, 5 MiB descompactado e 8 MiB compactado (o tamanho **real** conta, não o declarado). Recursos citados no manifesto precisam existir no pacote.
3. **Scripts contra a política (ADR-0012):** todo comando declarado (`commands` e passos de script) é avaliado; o que a política **nega** (inclusive por estar fora da lista) recusa a importação, e o **sensível** é aceito e marcado `needs_approval`. Arquivos em `scripts/` só são recusados quando contêm algo **comprovadamente proibido** (regra de proibição explícita), pois o padrão de negar não se aplica a texto livre.
4. **Origem e integridade:** cada skill guarda origem (`manual`/`file`/`package`/`git`), referência, revisão, **hash do conteúdo** (arquivos, manifesto e instruções, independente da ordem) e os arquivos (caminho, SHA-256, tamanho, conteúdo). Mesmo nome e mesmo conteúdo é **idempotente**; conteúdo diferente exige `replace=true` e gera nova versão.
5. **Trilha** (`skill_imports`): toda tentativa, inclusive as recusadas, com origem, resultado e motivo, **sem** o conteúdo do pacote.
6. **Execução separada e sob política** (`skills.Run`): IA via um `AIRunner` injetado; script no sandbox **sempre** pelo `Guard` (cada comando é reavaliado na hora de rodar); determinística pura; híbrida em sequência com variáveis compartilhadas. `NodeExecutor` roda nós `skill` do motor de workflows (uso independente do agente) e delega os demais tipos.

## Limites
- **Importação por Git não está disponível** (`POST /skills/import-git` responde 501 `unsupported`): depende do adaptador GitHub (T15-01).
- A IA real das skills de IA/híbridas não foi exercitada (testes usam um `AIRunner` de mentira); a ligação com os clientes CLI é do T09-01 + motor.
- Importar um pacote ainda não exige reautenticação (não carrega credenciais); a decisão pode mudar se pacotes passarem a trazer segredos.
