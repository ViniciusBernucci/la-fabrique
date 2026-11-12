> Leitura: [Índice didático](../../02-INDEX.md) · [Próximo →](01-referencia-v2.3.md)

# Painel de controle

IMPLEMENTADO no código base `a2cc5e0`; verificação operacional NÃO VERIFICADA nesta migração. Responsabilidade lógica, não nova classe/microserviço.

## Finalidade e limites

Oferecer navegação e operação administrativa da fábrica.

## Fluxo e comportamento

Menu principal: Painel, Projetos, Tarefas, Agentes de IA, Núcleos, Pipeline, Revisões, Repositórios, Histórico, Escritório, Usuários e Configurações, nessa ordem. Núcleos, Pipeline, Revisões, Repositórios e Histórico são prévias, sem novos endpoints. Tarefas abre o painel demonstrativo FAC-033 sem exigir login; Projetos abre controle; Agentes de IA e Configurações abrem configurações; Usuários usa a prévia de áreas futuras; Escritório abre o escritório virtual FAC-042. Os atalhos da home são independentes do menu. [Entrega FAC-032](../../09-entregas/2026/2026-10-07-FAC-032-menu-inicial-reduzido.md).

Home demonstrativa com núcleo SVG animado antes de login → sessão Bearer → projetos/tickets/runs/operação; menu abre recolhido, mantém expansão durante navegação e reinicia recolhido no reload.

Exemplo didático: ticket com critério “validar um campo” só avança com a base/definição/checks aprovados; request inválido não vira comando autorizado. O exemplo não é execução desta migração.

## Apresentação Centro de Comando — FAC-034

Tokens canônicos em [theme-hud.css](../../../apps/web/src/theme-hud.css), importados antes de styles.css; fundo com grade, títulos Orbitron, UI Rajdhani, rótulos técnicos JetBrains Mono, superfícies/bordas ciano. Login, projetos/operação, execuções, tarefas, configurações e dialogs compartilham o tema. Cores fixas ficam nos tokens; não usar paleta isolada por tela.

[OrchestrationCore](../../../apps/web/src/OrchestrationCore.tsx) substitui a imagem da home: JARVIS ao centro e os quatro outros agentes já existentes, rede determinística, anéis/ondas/partículas em CSS. Movimento reduzido desativa animações e transições. Relógio mostra a hora local, com timer limpo no unmount; status/diagnóstico continuam dados simulados, não telemetria ou orçamento. O atalho ENVIAR abre Tarefas, sem executar ou persistir diretivas. Projetos recentes permanecem projetos, sem serem convertidos em runs fictícias.

Frame mantém props/HomeDestination, busca ⌘K/Ctrl+K, skip link, aria-current, menu recolhível e prévia. Sidebar expandida mede 226px; recolhida 66px. Trilho de 78px oferece Geral/Modelos/Integrações/Acessos/Alertas/Equipe para a mesma tela de configurações; em telas estreitas vira faixa inferior. Textos existentes são preservados, maiúsculas de títulos por CSS. Estado atual IMPLEMENTADO no diff sobre de5dbfa; checks e limites visuais na [entrega FAC-034](../../09-entregas/2026/2026-10-07-FAC-034-centro-comando.md). Revisão independente/aceite visual pendentes.

## Visual sóbrio e seletor Novo/Antigo — FAC-043

O seletor Novo/Antigo na barra superior troca o visual de todas as telas sem mudar comportamento; a escolha fica no navegador e o padrão é "Novo". Novo = [theme-sobrio.css](../../../apps/web/src/theme-sobrio.css) sobre os mesmos tokens `--hud-*` (grafite, acento azul, Geist, sem brilho/grade, sem trilho direito) e painel inicial [SobrioHome](../../../apps/web/src/SobrioHome.tsx) com decisão pendente, indicadores e Orquestração em destaque. Antigo = HUD da seção seguinte. O orquestrador chama-se Núcleos em ambos. [Entrega FAC-043](../../09-entregas/2026/2026-10-25-FAC-043-visual-sobrio.md).

## Escritório virtual — FAC-042

Escritório 2D isométrico em Canvas 2D, carregado sob demanda. Agentes trabalhando ficam sentados digitando na própria mesa; entregas fazem o agente ir até a mesa do destinatário; livres circulam, tomam café ou leem; 10 mesas, vazias disponíveis. Somente visual, sem cliques. Laço a 30 fps apenas com o painel visível e a aba ativa; movimento reduzido mostra posições finais. Dados simulados pela interface `OfficeFeed`, preparada para eventos reais posteriores. Código em [apps/web/src/office](../../../apps/web/src/office/office-model.ts); [entrega FAC-042](../../09-entregas/2026/2026-10-24-FAC-042-escritorio-virtual.md).

## Estados, dados e regras

App/DashboardLayout; seta central no recolhido e padding direito 12px no expandido. Home simulada; admin protegido, sem token no bundle. Veja [estados por entidade](../../05-contratos/schemas/01-workflow.md) e [contratos canônicos](../../05-contratos/00-README.md).

## Falhas, recuperação e segurança

Payload/versão/base inválidos são recusados antes de efeito pertinente; auth/cota não autorizam fallback pago. Writer unknown permanece bloqueado; não repetir execução ou liberar lease por idade. Credenciais não entram em DTO/contexto/log/artefato; API não executa código do projeto. Limites de cada contrato e gates de provider continuam necessários.

## Código e evidência

[apps/web/src/App.tsx](../../../apps/web/src/App.tsx) e [apps/web/src/DashboardLayout.tsx](../../../apps/web/src/DashboardLayout.tsx). Testes versionados ao lado dessas implementações e evidências por FAC-020–031 no [histórico](../../09-entregas/00-README.md). Não reexecutados aqui. [ADR-003 aceito](../../06-decisoes/ADR-003-stack-typescript.md) e [feature relacionada](../../04-features/01-control-settings.md).

## Limitações e direção posterior

[AS-IS versus alvo](../../02-arquitetura/01-as-is.md) separa controles atuais de tenancy/RLS/broker propostos. Presença de implementação/teste versionado não comprova instalação ou eficácia sob UID de serviço.

## Referência integral local

[Detalhes por incremento com nota editorial](01-referencia-v2.3.md). Evidências/contratos históricos permanecem completos; afirmações de estado vencidas são delimitadas pela auditoria.

## Sequência de leitura deste capítulo

1. [referencia-v2.3](01-referencia-v2.3.md).
