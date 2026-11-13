# Ficha de preparação do aceite e do piloto

Pré-preenchida com o que o **repositório prova**; o que depende de decisão, acesso ou credencial do responsável está marcado **A PREENCHER** (nada foi inventado). Campos e roteiro: [05-ACEITE-E-PILOTO.md](../NEW%20MVP/05-ACEITE-E-PILOTO.md). Critérios rastreáveis por caso, épico e ticket: [02-matriz-de-evidencias.md](02-matriz-de-evidencias.md) (gerada; `make matriz`). Segredos **nunca** entram aqui nem no Git: só referência de conexão (`connection:<id>`).

## Campos (um por linha da ficha de 05-ACEITE-E-PILOTO.md)

| Campo | O que o repositório já fornece | A PREENCHER pelo responsável | Fecha com |
|---|---|---|---|
| **Responsável pelo aceite** | Um único administrador, com senha + TOTP (`make create-admin`); sem multiusuário no MVP (D01, ADR-0005). | Nome da pessoa e confirmação de que ela configura, decide exceções e faz o merge humano. | F01 |
| **Revisão da fábrica** | Commit: `git rev-parse HEAD`. Schema: última migration em `db/migrations` (`make migrate-status`). Stack: Go (`go.mod`), React 19 + Vite, PostgreSQL 17, Docker Compose. Imagens: [imagens-base](../operacao/imagens-base.md). | Commit, **digests** das imagens publicadas e data da revisão candidata (a CI/promoção ainda é parcial, T02-03). | F02 |
| **VPS e ambiente** | Scripts de inventário e benchmark (`scripts/vps-inventory.sh`, `scripts/vps-benchmark.sh`); backup e restauração com verificação: [backup-restauracao](../operacao/backup-restauracao.md) e `make restore-drill`; composição por ambiente: [ambientes-docker](../operacao/ambientes-docker.md). | Instância existente, **capacidade medida**, limites de concorrência e sandbox, local dos backups, TLS, namespace/volumes de homologação. | F03 |
| **Projeto e feature** | Cadastro em rascunho sem vínculos, descoberta de manifestos, plano versionado com critérios verificáveis (`/api/projects/{id}/features/{featureId}/plan`). | **Repositório existente**, branch base e SHA, feature real e sua chave externa, prioridade, escopo, critérios verificáveis, tarefas e dependências a demonstrar. | F04 |
| **Origem oficial** | Adaptadores Jira e ClickUp (fixtures), elegibilidade por marcador/estado, mapa de prioridade, verificação remota, sincronização manual e preflight (D89). | Jira **ou** ClickUp, projeto/lista, campo de elegibilidade, mapas de status e prioridade, conexão com permissões **verificadas na plataforma real**; ambiente de teste da outra origem. | F05 |
| **GitHub** | Branch, commit e push com prova; Draft PR e merge confirmado dependem do T15-03/T15-02. | Repositório, acesso restrito, branch de teste, identidades de revisão autorizadas; provar Draft PR no repositório e que a conta da fábrica **não** faz merge pelo produto. | F06 |
| **Configuração de desenvolvimento** | Perfil técnico (comandos com origem, serviços, referências de segredo), checks no baseline e na candidata com classificação de falhas. | Perfil da stack alvo, caminhos autorizados, comandos de instalação/checks/start e serviços auxiliares com dados de teste; **baseline e falhas existentes** registrados. | F07 |
| **Catálogos** | Agentes, skills, ferramentas (8 nativas), MCP com concessão por esquema pinado, modelos manuais, cadeia de fallback por agente. | Contas CLI (Claude Code, Codex) e API (OpenRouter, Vercel) **reais**, modelos, agentes, skills, MCP e cadeias de fallback; modo e escopos verificados. | F08 |
| **Orçamento e risco** | Tetos globais e por agente com reserva atômica, uso de API separado da cota de assinatura, autorizações com escopo e uso, painel `/usage`. | Valor dos tetos, limite por agente, fixture de indisponibilidade, regras de aprovação, prazos, usos e destinos. | F09 |
| **Workflows** | Editor visual, publicação imutável com configuração efetiva congelada, seleção por regras de dois níveis. | Versões, padrão e regras dos dois níveis, revisor, condições, loops, esperas, convergências, falha e aprovação do piloto. | F10 |
| **Ambientes de ensaio** | Sandbox por feature com supervisão e quarentena, fila local com vaga única por projeto. | Espaços isolados para segundo projeto, falhas, concorrência, revisão externa, restore e revogação. | F11 |
| **Evidências** | Matriz gerada com um espaço **por critério**: [02-matriz-de-evidencias.md](02-matriz-de-evidencias.md). Entregas e testes por ticket em `docs/entregas/`. | Caminho de armazenamento, política de acesso, responsável, prazo e índice dos arquivos/links; referências estáveis por revisão. | F12 |

## Usuários, conexões e critérios

- **Usuários:** um administrador (senha + TOTP; recuperação de uso único). Sem outros perfis.
- **Conexões necessárias** (todas só no cofre): GitHub (repositório do piloto), Jira ou ClickUp (origem real) e a outra em ambiente de teste, uma conta CLI e uma conta API por agente do piloto. Cada conexão precisa de permissão mínima e teste verificado.
- **Critérios:** cada critério é uma linha da matriz (caso P01–P18 → épicos → tickets → critério de aceite do ticket → evidência de software → evidência no projeto real).

## Pendências do contrato (para o responsável)

| ID | Decisão ou acesso necessário | Por quê não foi feito pelo agente |
|---|---|---|
| F01 | Identificar o responsável pelo aceite. | Pessoa e papel. |
| F02 | Fixar a revisão candidata e os digests das imagens. | A CI/promoção por revisão ainda é parcial (T02-03). |
| F03 | Medir a VPS e informar limites, backups e TLS. | Exige acesso à VPS (não disponível). |
| F04 | Escolher o repositório real e a feature do piloto. | Decisão de negócio do responsável. |
| F05 | Informar origem real, mapas e conexão com permissões. | Exige credenciais e plataforma reais. |
| F06 | Preparar o repositório do piloto no GitHub e provar Draft PR. | Exige credenciais e o adaptador de PR (T15-02). |
| F07 | Confirmar comandos e registrar o baseline do projeto real. | Depende de F04 e do acesso ao repositório. |
| F08 | Cadastrar contas CLI/API reais e modelos. | Exige credenciais e cotas reais. |
| F09 | Definir tetos, aprovações e prazos. | Gasto e política são do responsável. |
| F10 | Escolher o fluxo principal e os subfluxos do piloto. | Decisão do responsável sobre o processo. |
| F11 | Preparar ambientes isolados de ensaio. | Exige a VPS (F03). |
| F12 | Definir onde as evidências ficam e quem as acessa. | Política de acesso e prazo. |

**Estado de T17-01:** parcial. A matriz e a ficha estão prontas e verificadas; a seleção do projeto e da feature reais (F04) e os acessos (F03, F05, F06, F08) dependem do responsável. Enquanto F01–F12 não forem preenchidas, o contrato **não** está completo.
