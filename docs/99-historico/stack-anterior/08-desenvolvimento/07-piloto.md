> Leitura: [← Anterior](06-controle-mvp.md) · [Índice didático](../02-INDEX.md) · [Próximo →](08-tech-lead-handoff.md)

# Contrato do piloto

## Decisão obrigatória da stack - revisão 2.3
A stack da própria La fabrique está APROVADA: React + TypeScript + Vite no painel; NestJS + TypeScript na API; worker Node.js + TypeScript em processo separado; PostgreSQL; Redis + BullMQ; Docker Compose na mesma VPS. Não solicitar nova escolha ou confirmação da stack. Não iniciar a fábrica em PHP/Laravel, Angular ou .NET. Esta decisão substitui propostas anteriores.
Monorepo: `apps/web`, `apps/api`, `apps/worker`, `packages/contracts`. Contratos compartilhados precisam de validação em runtime. API não executa clientes, builds ou testes; o worker executa esses trabalhos com isolamento, limites e um writer inicial.
Ao trabalhar na própria fábrica, aplicar esta stack. A regra de preservar a stack existente aplica-se somente a projetos EXTERNOS cadastrados para desenvolvimento pela fábrica; ela não altera a stack da La fabrique. Se o repositório da fábrica contiver implementação anterior incompatível, registrar a divergência e planejar a adaptação por etapas; não apagar código existente nem reabrir a escolha tecnológica.
Versões exatas e comandos devem ser fixados conforme compatibilidade no bootstrap; isso não é uma nova decisão de stack. Repositório e funcionalidade do piloto externo permanecem pendentes quando não fornecidos.

Status: DEFERRED — o responsável escolherá e configurará o projeto pela interface da La fabrique quando desejar experimentar um projeto externo, em etapa separada da conclusão do MVP. O repositorio, a stack externa, os caminhos e os checks nao serao inseridos no codigo da fabrica. Este contrato não bloqueia desenvolvimento, testes, conclusão do software ou preparação operacional do MVP. A fábrica deve aceitar qualquer projeto configurado posteriormente pela interface; não precisa desenvolver um piloto para poder ser concluída.
- Repositório/acesso, base branch/SHA: A DEFINIR pelo cadastro de projeto e verificacoes do software.
- Funcionalidade pequena e critérios verificáveis: A DEFINIR.
- Stack da fábrica: React/Vite, NestJS e worker Node, todos em TypeScript; PostgreSQL e Redis/BullMQ. Decisão encerrada.
- Se o piloto for projeto externo: inspecionar somente a stack desse projeto.
- Versões/checks/baseline: A DEFINIR na definicao versionada do projeto, a partir do repositorio real.
- Caminhos permitidos/proibidos: A DEFINIR na interface de projeto.
- Infraestrutura: VPS única; perfil Bom recomendado 8 vCPU/16 GB/200 GB; perfil contratado A DEFINIR. Clientes oficiais autenticados na VPS; compatibilidade A VERIFICAR.
- Providers/planos/modos de autenticação/modelos: registrar no preflight real.
- Limites propostos: um writer, 30 min/tentativa, duas correções, dois handoffs.
- API: desativada; orçamento mensal zero. Extra usage/créditos/autorecharge: desativados nos fornecedores.
- Dados somente sintéticos; sem merge/deploy automático.
Preferir filtro, validação ou exibição num módulo existente com testes. Excluir autenticação/pagamento/migração irreversível na primeira amostra. Não assumir qual sistema será usado; pode ser escolhido pelo responsável.
