# ADR-0067 — Catálogo inicial de modelos (Claude e Codex) e lista por provedor no agente

- **Estado:** ACEITA · **Relacionadas:** [ADR-0021](0021-modelos-e-gateways-de-api.md), [ADR-0013](0013-catalogo-de-agentes-e-sobrescritas.md), [ADR-0065](0065-login-das-contas-de-assinatura-de-ia-pelo-painel.md)

## Contexto
No cadastro de agentes o campo **Modelo** era texto livre, e o catálogo de modelos (ADR-0021) nascia **vazio** e sem tela. Pior: a verificação de compatibilidade exige que o modelo esteja cadastrado para o provedor, então qualquer modelo digitado reprovava o agente ("não está cadastrado"). A pessoa precisava saber o nome exato e cadastrá-lo pela API.

## Decisão
1. **Catálogo inicial instalado ao subir a API** (`models.InstallDefaults`), só com o que falta: um modelo que já existe (mesmo desativado ou editado) não é tocado. Contém 12 modelos:
   - **Claude (4):** `claude-fable-5-2`, `claude-opus-5-5`, `claude-sonnet-5-5`, `claude-haiku-5-5`. Janela de contexto e saída máxima ficam em 0 (não informadas): não se inventa o que não foi observado.
   - **Codex (8):** os modelos de visibilidade "list" do cache do próprio cliente (`gpt-6.1-sol`, `gpt-6-astra`, `gpt-6.1-terra`, `gpt-6-sol`, `gpt-6-luna`, `gpt-5.6-sol`, `gpt-5.6-terra`, `gpt-5.6-luna`), com a janela de contexto que ele informa. Os ocultos (`gpt-reserve`, `codex-auto-review`) ficam de fora.
   Todos ativos, com ferramentas e streaming, e passam por `models.Check` com um agente que tem ferramentas.
2. **O campo Modelo do agente vira uma lista** (`ModelSelect`), carregada de `GET /api/models`:
   - mostra só os modelos **ativos do provedor** escolhido, ordenados pelo rótulo; trocar o provedor troca a lista e descarta o modelo anterior;
   - nos provedores de assinatura, a primeira opção é **Padrão da conta** (vazio);
   - no provedor `api`, filtra pelo **gateway da conta** escolhida (openrouter ou vercel);
   - **Outro (digitar o nome)** permite um modelo fora do catálogo; se a lista falhar ao carregar, o campo continua utilizável.

## Limites
- **Nada foi verificado executando os clientes.** Os nomes do Claude seguem a família atual conhecida; os do Codex vêm do cache local do cliente em 2026-11-16 e podem variar por conta e no tempo. Ajustar o catálogo (desativar, acrescentar) não exige mudar código; porém não há tela de cadastro de modelos ainda, só a API (`POST /api/models`).
- **Gateways de API (OpenRouter e Vercel) não têm modelos semeados:** o catálogo deles é grande e muda, e não foi possível conferi-lo aqui. A lista do provedor `api` mostra o que for cadastrado.
- O campo "Modelo" do nó de workflow (sobrescrita) continua texto livre.
