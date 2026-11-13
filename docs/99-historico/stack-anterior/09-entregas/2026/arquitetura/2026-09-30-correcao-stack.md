# Entrega: correção consistente da stack

Status: documentação atualizada, revisão 2.3.

Problema: anexos 2.1 sem a decisão de stack e arquitetura com instrução residual de confirmação.

Implementado: React/Vite no painel, NestJS na API e worker Node separado, todos em TypeScript, explicitados no início dos guias, plano, backlog, especificação, piloto e política. Preservação de stack restrita aos projetos externos. VPS única e requisitos anteriores preservados. PDF e ZIP gerados dos mesmos Markdown.

Validação: conferência de todos os guias e busca por instruções residuais; nenhum código da aplicação ou teste de runtime foi executado.

Diff resumido:
```diff
- Stack/versões/checks/baseline: inspecionar repo real.
+ Stack da fábrica aprovada: React + NestJS + worker Node em TypeScript.
- Stack da fábrica proposta pode ser confirmada antes de bootstrap.
+ Não solicitar nova escolha ou confirmação da stack.
- Preservar stack e padrões do repo alvo.
+ Preservar stack existente somente em projetos externos cadastrados.
```

Uso: substituir os documentos antigos pelo kit completo, preservando código e regras locais específicas. Rollback documental: recuperar a revisão anterior apenas para consulta histórica, sem adotá-la como instrução vigente.
