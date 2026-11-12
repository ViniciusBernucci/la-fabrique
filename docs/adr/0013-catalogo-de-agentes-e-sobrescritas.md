# ADR-0013 — Catálogo de agentes e sobrescritas por nó

- **Estado:** ACEITA · **Ticket:** [T08-01](../NEW%20MVP/03-BACKLOG.md#t08-01) · **Épico:** E08 · **Decisões:** D10–13, D74, D78–80

## Decisão
1. **Catálogo global** (`agents`): nome (slug), instruções, provedor (`claude`, `codex` ou `api`), **conta como referência** a uma conexão do cofre, modelo, skills e ferramentas por nome, cadeia de fallback (validada aqui, usada no T09-04) e tempo máximo. Controle otimista por versão. Nenhum segredo trafega: só o ID da conexão.
2. **Compatibilidade da conta:** `claude`/`codex` exigem conexão `cli` ativa cujo destino é o próprio cliente; `api` exige conexão `api`. Vale na gravação do agente, no fallback e na conta sobrescrita pelo nó.
3. **Sobrescritas ficam no nó do workflow, nunca na base:** o nó `agent` ganhou `model`, `connection`, `tools`, `instructions_extra` e `timeout_minutes` opcionais. `Resolve` calcula a **configuração efetiva** (base + nó) numa cópia: a base não é alterada e cada valor registra sua origem (`agent` ou `node`).
4. **Privilégio mínimo:** o nó só pode **reduzir** as ferramentas do agente (subconjunto da base). Ampliar é recusado (`invalid_override`). Instruções adicionais são anexadas; modelo, conta e tempo substituem.
5. **Efetivo sanitizado:** `GET /workflows/{id}/nodes/{node}/effective` mostra a configuração efetiva do **rascunho salvo** (sem segredos), com a versão do agente usada. Erros: agente ausente (`agent_not_found`), sobrescrita inválida (`invalid_override`), conta incompatível/revogada (`connection_incompatible`).
6. O workflow referencia o agente **pelo nome**; renomear um agente exige ajustar os nós (sem reescrita automática).

## Limites
- A configuração efetiva congelada por execução (snapshot) é do T07-03; hoje ela é calculada sob demanda a partir do catálogo atual.
- Skills e ferramentas são nomes livres até os catálogos T08-02/T08-03; a allowlist de MCP por agente é do T08-04.
- Não há exclusão de agente.
