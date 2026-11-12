> Leitura: [← Anterior](00-LEIA-ME-PRIMEIRO.md) · [Índice didático](02-INDEX.md) · [Próximo →](02-INDEX.md)

# Manual Vivo da Fábrica de Software

Revisão documental: 2026-10-07 (Europe/Berlin), código base a2cc5e0. Há implementação no monorepo; verificação operacional depende de cada controle. [AS-IS e alvo](02-arquitetura/01-as-is.md).

## O que é e qual problema resolve

A Fábrica organiza trabalho de desenvolvimento executado com agentes de IA. Um pedido vira ticket com escopo e critérios; uma execução produz código, checks, revisão e documentação; uma pessoa aceita a revisão específica. O objetivo é reduzir retrabalho e aproveitar assinaturas existentes, com limites e recuperação explícitos.

Neste repositório existem painel, API, worker, contratos Zod, runtime e migrations. O pacote fornece planejamento posterior de tenancy/broker: essa parte permanece PROPOSTA. Implementação, teste histórico, implantação e aceite são dimensões separadas.

## Como funciona em dois minutos

1. O usuário cadastra projeto e ticket com objetivo, caminhos autorizados e critérios verificáveis.
2. A API administrativa valida READY, definição e SHA e grava outbox; o claim interno cria/recupera a Run. Auth atual usa Bearer, sem tenancy.
3. O Context Builder seleciona fontes da revisão concedida e registra hashes/omissões.
4. Um worker recebe o job; o Executor prepara a sandbox e garante um único writer.
5. O runtime usa cliente oficial com identidade privada por instalação e perfis nativos de ferramentas; checks usam SandboxRunner. Broker exclusivo é alvo posterior, ainda ausente.
6. O executor coleta resultados, roda checks e encaminha a uma sessão independente de revisão.
7. Até duas rodadas de correção; documentação atual e registro de entrega passam pelo gate.
8. O workflow retorna AWAITING_HUMAN; a persistência aguarda aceite em VALIDATING. Novo diff invalida aceite e evidências afetadas.

Se o provider atingir limite, a fábrica confirma término do writer e preserva checkpoint, patch e arquivos não rastreados. Outro provider elegível configurado explicitamente retoma em uma tentativa nova. Sem provider disponível, aguarda; não habilita gasto extra.

## Arquitetura em uma leitura

React é o painel. NestJS organiza o plano de controle como monólito modular quando adequado. PostgreSQL persiste workflow/configuração; Redis participa de fila/cache, com outbox transacional no banco. Worker Node/TypeScript e Executor administram execução. Runtimes autenticados ficam separados do código do projeto, mesmo na mesma VPS. Inferência ocorre nos fornecedores externos.

[Contexto C4](02-arquitetura/02-contexto.md) mostra pessoas e sistemas externos; [containers C4](02-arquitetura/03-containers.md) mostra unidades de execução e armazenamento; [componentes C4](02-arquitetura/04-componentes.md) explica responsabilidades internas. Container C4 não significa necessariamente container Docker.

## Módulos e features

Projetos e tickets definem trabalho; runs/orchestrator coordenam estados; worker/executor/sandbox impõem lifecycle; runtime/providers escolhem instalações elegíveis; context builder limita entrada; tool broker controla ações; documentation gate e approvals vinculam conclusão a evidência; observabilidade/economia mostram resultados e desconhecidos. [Catálogo de módulos](03-modulos/00-README.md).

Features descrevem comportamento e critérios. FAC registra entregas e aceites reais; LF-MT registra gates propostos posteriores, sem alterar esses aceites. [Catálogo de features](04-features/00-README.md).

## Limites importantes para entender o desenho

Uma VPS é ponto único de falha; backup fica fora dela. Um executor global inicial e um writer por workspace. Container comum é perfil de piloto próprio com dados sintéticos; código hostil de terceiros requer avaliação de perfil mais forte. Mediação exclusiva de tools pelo broker é condição a provar: se o cliente não suportar, provider fica desabilitado. Não existe garantia universal contra kernel/root comprometido.

## Conta do usuário

FAC-038–041 adicionam cadastro/login e painel Minha conta, acessível pelo avatar ou menu Usuários: perfil, equipe, convites, projetos/permissões/execuções, integrações privadas, segurança/MFA, notificações, dados/exportação, plano/uso e atividade. Essa camada não concede acesso automático aos projetos/configurações globais descritos acima; sua autorização usa cookie próprio. [Funcionamento e limites](04-features/01-control-settings.md#painel-da-conta-do-usuário--fac-038). Implantação/aceite pendentes.

## Onde continuar

- [Como ler o sistema](00-LEIA-ME-PRIMEIRO.md): trilhas por objetivo.
- [Visão geral](01-visao-geral/00-README.md): escopo, objetivos e glossário.
- [Arquitetura](02-arquitetura/00-README.md): C4, dados, fluxos e segurança.
- [Contratos](05-contratos/00-README.md): API, eventos, filas, runtime e schemas.
- [Decisões](06-decisoes/00-README.md): razões e status de ADRs.
- [Operação](07-operacao/00-README.md): deploy, backup, restore e incidentes planejados.
- [Desenvolvimento](08-desenvolvimento/00-README.md): piloto, plano, testes e contribuição.
- [Entregas](09-entregas/00-README.md): o que mudou, com evidências reais.
- [Lessons](10-lessons/00-README.md): conhecimento aplicado, sem exemplos de implementação inventados.
- [Governança](00-governanca/00-README.md): política, auditoria, fontes e mapa de migração.

Este manual resume e aponta fontes canônicas. Contratos detalhados não devem ser copiados para vários capítulos.

## Proveniência

- [Planejamento v2.1](99-historico/originais/sources/README.md)
- [Proposta multi-tenant](99-historico/originais/output/le-fabrique-multitenant/README.md)
- [Contratos v3](99-historico/originais/output/le-fabrique-multitenant/ESPEC-MVP.md)

## Entender memória e decisões antes de avaliar os planos

[Guia didático passo a passo](02-arquitetura/memoria-e-decisoes/00-README.md) explica as oito capacidades propostas. [Planos separados](08-desenvolvimento/planos-memoria-e-decisoes/00-README.md) distinguem próxima versão e evolução futura. Nenhuma nova integração foi implementada por esse planejamento.
