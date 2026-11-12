> Leitura: [← Anterior](06-RUNBOOK.md) · [Índice didático](../../02-INDEX.md) · [Próximo →](08-ENTREGA.md)

# HANDOFF — TICKET / RUN

**Template: campos devem ser preenchidos com evidência real; headings vazios aqui são intencionais e não representam documento concluído.**

Data/fuso; PARCIAL/INTERROMPIDO/PRONTO PARA CONTINUAR; provider/versão/modelo conhecido; motivo/limite observado sem segredo.

## Contrato
Objetivo, critérios, tenant/projeto autorizados, escopo/paths proibidos.
## Estado recuperável
Base/code SHA, branch/workspace, policy/config, fencing, prova EXTERNA de writer parado; patch/untracked artifact IDs/hashes/exclusões.
## O que foi feito e verificado
Arquivos/decisões/fontes; checks/revisão/resultados; NOT_RUN; baseline/regressões.
## Próximas ações
Passos prioritários e diagnóstico; não reaplicar patch presente.
## Documentação
Relato parcial, estado atual/docs pendentes, lessons.
## Uso e limites
Fonte/horário/cota/reset conhecido; tempo/tokens quando observáveis; unknown.
## Gate de retomada
Membership/instalação do mesmo tenant/política revalidados; nova attempt/sandbox/sessão; sem auth/transcript privado. Sem quiescência, BLOCKED_RECOVERY.

## Campos locais preservados

O template local anterior também exige objetivo/atual-esperado, escopo/paths proibidos, risco/orçamento/dependências, funcionamento, diff sanitizado, checks por revisão e baseline/regressões, rollback, docs/lessons, provider/modelo/auth/billing sem segredo, custos/uso unknown e aceite exato. Cada item deve aparecer na seção pertinente; não gerar cópia adicional por domínio.
