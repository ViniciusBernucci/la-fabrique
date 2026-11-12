# FAC-019 — Abas de configurações

IMPLEMENTADO / AWAITING_HUMAN. Código 0182a4fa02dfaeb008cad2079084b251a317b12b, baseline d6802fe com FAC-018; developer autorizado. Pedido: Contas, Integrações IA, Equipes nessa ordem.

Contas é a aba inicial e reúne contas de IA/GitHub com seus modais. Integrações IA apresenta modelos configurados por conta e proteções financeiras; Configurar encaminha ao mesmo modal da conta. Equipes apresenta os funcionários digitais e mantém modal de atribuições. Resumo e salvamento continuam comuns. Abas usam tablist/tab/tabpanel, seleção acessível e navegação com setas/Home/End; painéis hidden preservam rascunhos e consultas em andamento. Não há alteração de schema, endpoints ou auth pelo ticket.

Verificação: 40 testes web passaram após alteração; typecheck/build monorepo e lint passaram, com único aviso preexistente em run-delivery.service.ts:209. git diff --check passou. Interação visual/foco no navegador NÃO VERIFICADA; aceite humano da revisão pendente. Não foram adicionados testes que reproduzem marcação para esta mudança reversível.

[Diff sanitizado conjunto FAC-018/FAC-019](../evidencias/configuracao/evidencias/FAC-018-diff.patch). Rollback: restaurar SettingsPanel.tsx/styles.css da revisão anterior depois de separar alterações FAC-018 no mesmo arquivo; rebuild web. Nenhum dado persistido depende dessas abas. Conceito registrado em lessons/modal-edicao-configuracao.md.
