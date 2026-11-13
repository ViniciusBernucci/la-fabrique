# ADR-0021 — Catálogo de modelos e gateways de API (OpenRouter e Vercel AI Gateway)

- **Estado:** ACEITA · **Ticket:** [T09-02](../NEW%20MVP/03-BACKLOG.md#t09-02) · **Épico:** E09 · **Relacionadas:** [ADR-0005](0005-autenticacao-sessoes-e-cofre-de-conexoes.md), [ADR-0011](0011-clientes-cli-de-ia-no-sandbox.md), [ADR-0013](0013-catalogo-de-agentes-e-sobrescritas.md), [ADR-0015](0015-catalogo-de-ferramentas.md)

## Decisão
1. **Modelos são cadastrados à mão** (`models`): provedor (`claude`, `codex`, `api`), gateway (só no `api`: `openrouter` ou `vercel`), nome, capacidades (`supports_tools`, `supports_streaming`), janela de contexto, saída máxima e parâmetros padrão (`temperature` 0–2, `top_p` 0–1, `max_tokens` dentro da saída máxima). Identidade única por provedor + gateway + nome; estado `active`/`disabled` com versão (controle otimista).
2. **Compatibilidade (`models.Check`)** devolve um veredito com os **motivos**, nunca um "não" mudo. Modelo só serve quando: está cadastrado para o provedor (e para o gateway da conta), está ativo e, se o agente tem ferramentas, suporta ferramentas. No provedor `api` a conta precisa ser `api`, ativa e com **destino igual a um gateway suportado**; o gateway da conta decide em qual catálogo o modelo é procurado. CLI sem modelo explícito usa o padrão da conta; `api` sem modelo é incompatível.
3. **Sobreposição verificável:** a configuração efetiva do nó (`GET /api/workflows/{id}/nodes/{node}/effective`) ganhou `model_check`, calculado sobre o modelo, a conta e as ferramentas **efetivos** (já com as sobrescritas do nó) e mostra a origem (`agent`/`node`) de cada valor. Nenhuma sobrescrita altera o agente do catálogo.
4. **Gateways (`internal/gateways`):** um cliente OpenAI-compatível (`/chat/completions`) serve os dois gateways (endereços oficiais fixos; `BaseURL` só em teste). HTTPS obrigatório, sem credenciais na URL, sem redirecionamento, destino interno recusado. Erros têm mensagem fixa (sem corpo remoto, URL ou token). A credencial vem do cofre por `gateways.Open` (conta `api` ativa) e nunca é devolvida ou logada.
5. **Ferramentas controladas:** o laço (`gateways.Run`) anuncia ao modelo **só as ferramentas que o agente tem** e executa cada pedido por um `Executor`. `RegistryExecutor` liga ao catálogo (ADR-0015): concessão, escopo, validação e **política de risco** continuam valendo. Pedido de ferramenta fora do agente ou com argumentos que não são objeto JSON é recusado sem executar; falha da ferramenta volta ao modelo redigida e limitada (300 caracteres). Limites: 8 rodadas, 24 chamadas, resultado de 16 KiB.
6. **Elegibilidade na execução:** `Run` confere de novo (modelo `api`, ativo, com suporte a ferramentas quando há ferramentas) **antes do primeiro pedido** ao gateway.

## Limites
- **OpenRouter e Vercel AI Gateway reais: NOT_RUN** (exigem conta do responsável). A prova usa um gateway de mentira com o protocolo de chat completions e o **sandbox Docker real** para as ferramentas.
- Sem streaming (`supports_streaming` é só declarativo), sem tela no painel e o laço ainda **não** está ligado ao worker (T07/T11 consomem `gateways.Run`).
- A verificação de compatibilidade não é imposta ao salvar o agente/workflow (é informada no efetivo); o bloqueio na execução é de `gateways.Run`.
- Custo/uso (`Usage`) é o informado pelo gateway; sem tabela de preços.
