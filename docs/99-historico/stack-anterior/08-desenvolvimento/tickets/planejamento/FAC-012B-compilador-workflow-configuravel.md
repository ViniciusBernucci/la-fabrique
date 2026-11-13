# FAC-012B — Compilador da especificação para workflow

Status: DONE

## Objetivo

Compilar o snapshot imutável FAC-012A em `DeveloperWorkflowRequest`, sem ativar o consumer BullMQ, cadastrar projeto/piloto, escolher provider ou executar checkout/comandos. Checkout local, política de contexto/limites e allowlist de executáveis entram como dados confiáveis injetados pelo worker; seleção de modelo segue o Centro de Configurações.

## Escopo e critérios de aceite

1. Adicionar compilador puro e contrato de configuração confiável no worker.
2. Exigir correspondência exata entre projeto e repositório local resolvido (ID, URL e SHA-base); divergência falha fechado.
3. Traduzir objetivo, critérios e instruções da definição para pedido válido do workflow.
4. Converter checks configurados somente quando nome, executável e argv coincidirem exatamente com allowlist confiável; nunca interpretar shell ou executar o comando.
5. Exigir fontes de contexto e limites/política válidos fornecidos pela configuração confiável, sem inventar defaults específicos de projeto.
6. Preservar `modelRequested` como valor resolvido externo opcional; não codificar conta, provider ou modelo.
7. Testes provam correspondência, rejeição de desvios e validação runtime. O consumer real continua desligado.
8. Lint, typecheck, testes, build e `git diff --check` passam.

## Entrega

Código: `e7b9bf60e03089c502255d19edbb5d1e31725d2c`. Revisao documental aceita: `3e28363b0642f8e05840bb649b681e3837e1e015`. Aceite explicito do responsavel em 2026-10-02. O consumer continua desligado ate cumprir os gates de seguranca pendentes.

## Baseline, caminhos e limites

Base: `3ef3d543b98fb48226714787315f4de947fa6dd9`. Branch/worktree: `feat/fac-012b-workflow-compiler`, `/home/vinicius/le-fabrique-fac-012b`. Caminhos: `apps/worker/src`, testes e documentação de operação/runtime. Sem migration, rede, banco/Redis reais, checkout, cliente de IA ou alteração de configuração externa. Um writer; até duas rodadas de correção; custo adicional zero.

## Provider elegível

Nenhum: implementação determinística com fixtures; nenhum adapter será invocado.

## Dependências e limites

FAC-001A, FAC-003A, FAC-009 e FAC-012A. Perfil ainda não fornece clone/repositório real; esta entrega define e verifica a fronteira de entrada, mas não a materializa nem habilita execução.
