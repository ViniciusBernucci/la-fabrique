> Leitura: [← Anterior](12-documentacao-proveniencia.md) · [Índice didático](../02-INDEX.md) · [Próximo →](14-video-decorativo-loop.md)

# Edição em modal com rascunho separado

FAC-013 em SettingsPanel.tsx mantém editingInstallation separado de draft. Abrir usa structuredClone; editar não muda a linha ou referências de funções. Cancelar/Escape descarta o clone e conta nova só entra na lista após persistência bem-sucedida. saveInstallation reconcilia catálogo e atribuições antes da API versionada; só fecha após retorno positivo, preservando entrada quando a requisição falha.

Dialog nativo aberto por showModal fornece foco modal/fundo inerte pelo browser; aria-labelledby identifica o formulário. O estado saving bloqueia cancelamento e campos durante a requisição. A validação automatizada existente cobre build/tipos/view models, mas foco, Escape e retorno do foco ainda requerem prova de interação no navegador. Não confundir renderização estática com essa prova.

[Implementação e evidências](../09-entregas/2026/configuracao/2026-10-05-FAC-013-modal-contas-ia.md).

FAC-014 extrai SettingsModal com título por useId e cleanup do dialog, reutilizado nas três seções. editingAssignment/editingGithub isolam campos do draft global; ações de integração/PR continuam independentes do cancelamento do formulário. Fechar um modal descarta configuração local, mas não é rollback de pedidos já enviados ao worker. [Evidências](../09-entregas/2026/configuracao/2026-10-05-FAC-014-modais-funcionarios-github.md).

FAC-016: mensagens globais podem ficar atrás do dialog nativo. SettingsPanel mantém envio/erros no modal com role=status/alert e traduz resultado por estado em providerVerificationFeedback. Confirmar API/worker separadamente evita tratar falha operacional como clique perdido. Atualização de código pelo watcher não recarrega ambiente herdado do supervisor; a causa conhecida deve ter orientação fixa sem expor erro bruto.

FAC-019: abas em SettingsPanel agrupam o mesmo estado persistido sem duplicar formulários; hidden preserva rascunhos/polling, aria-controls/selected identifica painel e setas/Home/End acompanham foco. Lista de modelos encaminha ao modal de conta na aba Contas. Typecheck/build não provam interação visual do navegador.


FAC-020: `HomeDashboard` usa dialog nativo apenas para prévias demonstrativas; dados fictícios não são importados para o estado administrativo. `preview` abre o dialog por efeito, `onClose` limpa a seleção e formulário `method="dialog"`/Escape encerram a prévia. A evidência Playwright comprova fechamento e restauração de foco ao botão Financeiro. Links existentes saem da home e usam autenticação real; clicar num projeto fictício não assume identidade de um projeto persistido. [Relatório](../09-entregas/2026/controle/2026-10-05-FAC-020-central-controle-estatica.md).

FAC-022: `App` monta DashboardLayout uma vez e muda apenas o slot central; o browser mantém referências aos nós sidebar/topbar e prova identidade em home/login/controle/configurações. Isso preserva o chrome/estado do template; o rascunho de SettingsPanel continua sujeito ao ciclo de vida do próprio painel. Fontes escopadas ao slot também alcançam o dialog nativo, medido em 16px/12px sem salvar a fixture. [Relatório](../09-entregas/2026/controle/2026-10-05-FAC-022-template-tipografia.md).


FAC-023: TeamsPanel mantém clone de agente/skill separado do draft de settings, fecha somente após PUT positivo e mostra erro 409 dentro do modal. removeProjectSkill/upsertProjectSkill e pruneTeamAgentInstallations limpam referências ao excluir/mover skill ou retirar conta/modelo. Browser prova oito agentes, recarga e cancelamento após conflito com HTTP fixture; não é prova de escrita PostgreSQL real.


FAC-026: separar botão da linha e switch evita controle interativo aninhado. A alteração de enabled confirma PUT versionado antes de mostrar novo estado; conflito mantém valor e erro. Ao transitar de dialog informativo para configuração, foco nativo pode tentar voltar a botão de dialog já desmontado; PresetAgents preserva referência à linha e restaura foco ao terminar configuring. Browser mede retorno, gravação e cancelamento, com fixture explícita. FAC-027 mantém recolhimento no template e usa uma variável CSS para width/margem em todos os breakpoints; storage opcional não deve bloquear navegação.


FAC-028: retirar texto/moldura visual de um botão não exige retirar aria-label, title ou foco de teclado. CSS transparente no hover evita caixa implícita da regra global de botões; browser verifica estilo computado e o fluxo já existente.


FAC-029: alinhar à borda requer ajustar margem/padding, além de justify-content. Browser compara getBoundingClientRect do SVG e do menu nos dois estados; diferença de 1px representa a borda real do sidebar.


FAC-030: estado preservado pelo template durante navegação é diferente de persistência entre aberturas. Novo pedido definiu início sempre recolhido; useState(true) com atualização funcional preserva a navegação sem consultar/escrever preferência antiga. Browser semeia chave antiga false e comprova padrão no reload, além de troca de tela preservada.


FAC-031: alinhamento visual pode depender do estado do template. Regra específica is-menu-collapsed centraliza o SVG sem aplicar padding do estado expandido; browser mede centro e distância da borda, considerando a borda de 1px do container.
