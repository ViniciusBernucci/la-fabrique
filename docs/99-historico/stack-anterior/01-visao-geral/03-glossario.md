> Leitura: [← Anterior](02-escopo.md) · [Índice didático](../02-INDEX.md) · [Próximo →](04-mapa-do-sistema.md)

# Glossário

Revisão 2026-10-07, base a2cc5e0. Conceitos propostos são explicitamente distinguidos de entidades físicas atuais.

## Tenant

PROPOSTO: unidade de ownership e isolamento; não é apenas prefixo. Ausente no schema atual.

## Project

Repositório/ref e definição versionada de paths/checks/contexto/documentação. Tenant é direção posterior.

## Ticket

Pedido de trabalho com escopo, risco, limites e critérios verificáveis.

## Run

Execução de um ticket; guarda base/revisão, política e estado.

## Step

Conceito de etapa Developer/Reviewer; não há modelo físico Step no Prisma atual.

## Attempt

Tentativa de Run com sequência/fence/worker/lease/stop; funções/chamadas são observações no resultado.

## Worker

Serviço autenticado que recebe jobs e coordena execução; não é o modelo.

## Executor Manager

Componente confiável de lifecycle, admissão e coleta; não é ferramenta da IA.

## Provider

Fornecedor/cliente de IA (Codex, Claude ou Antigravity); não é um papel obrigatório.

## ProviderInstallation

Instalação configurada no JSON FactorySettings, com observações de verificação/login. Ownership por tenant é proposta posterior.

## Provider Runtime

Cliente oficial com store privado por instalação e perfil nativo; broker exclusivo é alvo posterior.

## Tool Broker

PROPOSTO: autoriza operações restritas fora do modelo e as executa na sandbox; sem acesso a auth/DB/Redis/controle. Nenhum broker real nesta base.

## Capability

PROPOSTA: autorização temporária por canal/attempt/escopo; não é credencial de fornecedor.

## Sandbox

Ambiente restrito de checks (SandboxRunner) e perfis nativos de ferramentas CLI distintos. Eficácia requer ensaio sob identidade real.

## Checkpoint

Estado recuperável com revisão, patch/untracked, hashes e prova de writer parado.

## Artifact

Arquivo/resultado validado e imutável por hash, com ownership e retenção.

## ContextManifest

Proveniência do contexto: fontes/hashes/revisão/omissões/instruções.

## Lease

Prazo de posse de uma execução; expiração não prova término de processo.

## Fencing token

Versão de autoridade que rejeita eventos antigos; não mata writer local.

## Quiescência

Confirmação externa de fim da árvore/processos antes de coleta ou novo writer.

## Outbox

Registro transacional de dispatch no banco, publicado na fila com idempotência.

## Approval

Aceite associado à revisão exata e políticas/configuração relevantes.

## UsageObservation

Uso observado com unidade/fonte/horário; desconhecido é nullable.

## LedgerEntry

PROPOSTO: registro financeiro decimal de assinatura/extras/API; não há Ledger físico implementado.

## C4

Modelo de arquitetura por contexto, containers e componentes; container não significa Docker.

## ADR

Registro de decisão com contexto, alternativas, consequências e condições de revisão.

## Manual Vivo

Navegação didática do estado documentado, ligada aos capítulos canônicos.

## Delivery Record

Registro histórico de uma entrega; não substitui documentação do módulo.

## R0–R4

Escala de risco: documental/cosmético; lógica local; integração; auth/pagamentos/segredos; produção/infra/migração irreversível.

## Proveniência

- [Entidades v2](../99-historico/originais/sources/ESPEC-MVP.md)
- [Identidade v3](../03-modulos/tenant-isolation/00-README.md)
