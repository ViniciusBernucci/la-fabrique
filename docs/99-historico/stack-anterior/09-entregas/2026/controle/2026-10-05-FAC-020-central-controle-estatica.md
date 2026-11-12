# FAC-020 — Central de controle estática
Data: 2026-10-05 (Europe/Berlin; verificação final às 22:25).
Status: IMPLEMENTADO / AWAITING_HUMAN.
Domínio: controle web.
Base SHA: 8285173. Revisão do código: `904d0b5c4619bc2ab518d2d1c0d2912ea166b84a`.
Branch: `feat/fac-020-home-dashboard`; worktree exclusiva `/home/vinicius/le-fabrique-fac-020`.

## Objetivo e critérios de aceite
Pedido do responsável: tela inicial estática o mais parecida possível com a referência local fornecida, com notificações, mensagens e links. Usar dados simulados quando as funcionalidades/dados não estiverem disponíveis. Ticket [FAC-020](../../../08-desenvolvimento/tickets/controle/tickets/FAC-020.md) registrado READY antes da implementação. Preservar React/TypeScript/Vite e autenticação das funcionalidades existentes.

## O que foi implementado
- `HomeDashboard.tsx`: navegação lateral, busca de atalhos, perfil demonstrativo, escritório cartoon, quatro indicadores, nove acessos rápidos, mensagem importante, quatro notificações, cinco agentes e quatro projetos demonstrativos.
- `home-dashboard.css`: estilos exclusivos da home e layout responsivo. Desktop de referência: sidebar 234px, topbar 66px, resumo lateral 366px; dispositivos pequenos colapsam o menu para ícones e empilham os cartões.
- `App.tsx`: home como entrada padrão; links existentes levam à Operação/Configurações, passando pelo formulário de token quando necessário. Botões permitem voltar à home sem perder a sessão administrativa em memória.
- Ícones e avatares são SVGs locais. O escritório usa o PNG fornecido, mantido integralmente em `public/images/control-room-reference.png`, com enquadramento em CSS e seis regiões clicáveis sobre os rótulos. Não foi gerada uma nova ilustração ou declarado que o escritório é vetorial nativo.
- As áreas futuras abrem prévia em `dialog` nativo, identificada como demonstração. Busca filtra atalhos; Ctrl/Cmd+K foca o campo. Escape/fechamento devolvem foco ao botão de origem.

## Funcionamento
A entrada estática não solicita token nem chama API. Os dados são fixtures locais da apresentação e não entram nos contratos, cadastros ou execução. Cabeçalho/workspace/rodapé identificam demonstração. Percentuais são atividade simulada, não cota de assinatura. JARVIS/Marketing/Agentes/Configurações levam ao Centro de Configurações; Desenvolvimento/Projetos/Novo Projeto/projetos recentes levam à Operação. Projetos recentes não selecionam um registro real com o mesmo nome.

A primeira entrada administrativa continua exigindo login, mesmo que exista um token armazenado: o estado `authenticated` só é promovido pela validação existente. A home não libera writer, não inicia provider, não dispara jobs, nem aprova tickets. Voltar à home após autenticação mantém os mecanismos de polling já existentes da aplicação; a ausência de chamadas verificada no browser se refere à entrada estática antes do login.

## Alterações e diff
[Patch textual sanitizado](../evidencias/controle/evidencias/FAC-020/implementation.patch), associado à revisão `904d0b5`. O patch usa contexto zero (`git diff --unified=0`) para não introduzir whitespace de linhas de contexto no artefato e inclui código/CSS/testes; PNGs são revisáveis pelos próprios arquivos e hashes, não por patch textual.

SHA-256 do recurso fonte: `88fffabdb345ead38cf810b7f3b2bff13fe9d304da8449059ca14945cb0a11ad`.
Captura desktop: `35d0974e5bf62406f8c8ff29b4705e5854c162a19e02b043dd80033a418f4542`.
Captura mobile: `ba06e323e23bb2315c71bb257cc00c18d28e1ffc90ef065265967ce2753f4f0b`.

## Verificação
Baseline `8285173`: typecheck web e 40 testes/15 arquivos passaram. Lint raiz passou com um aviso preexistente `useOptionalChain` em `apps/api/src/orchestration/run-delivery.service.ts:209`.

Revisão final do código:

| Check | Resultado real |
| --- | --- |
| `npm run typecheck -w @le-fabrique/web` | PASS |
| `npm run test -w @le-fabrique/web` | PASS: 42 testes/16 arquivos |
| `npm run build -w @le-fabrique/web` | PASS: Vite 8.3.1, 132 módulos |
| `npm run lint` | PASS: 250 arquivos; mesmo único aviso preexistente |
| `git diff --check` / `git diff --cached --check` | PASS |
| `git apply --check --unidiff-zero implementation.patch` no baseline original | PASS: patch textual aplicável, sem efetuar alterações |
| Chromium 153.0.8010.12 / Playwright 1.63.0 | PASS: home sem API, busca/atalho/empty state, dialog/Escape/foco, login 401 sintético, retorno à home |
| Responsividade | PASS: sem overflow horizontal em 1536, 1280, 1024, 768, 390 e 320px |
| Inspeção visual | Executada nas [capturas desktop](../evidencias/controle/evidencias/FAC-020/desktop.png) e [mobile](../evidencias/controle/evidencias/FAC-020/mobile.png); composição próxima da referência |

[Script reproduzível](../evidencias/controle/evidencias/FAC-020/browser-check.mjs) e [resultado integral sanitizado](../evidencias/controle/evidencias/FAC-020/browser-results.json). Para executar, disponibilizar Playwright no ambiente de verificação ou definir `PLAYWRIGHT_MODULE` para seu módulo; `PREVIEW_URL` aceita URL de preview. Comando desta sessão:

```bash
LD_LIBRARY_PATH=/tmp/extracted/usr/lib/x86_64-linux-gnu \
PLAYWRIGHT_MODULE=/home/vinicius/.npm/_npx/420ff84f11983ee5/node_modules/playwright/index.mjs \
node documentacoes/controle/evidencias/FAC-020/browser-check.mjs
```

Bibliotecas ausentes do Chromium foram baixadas/extraídas em `/tmp` para o browser de verificação, sem instalar pacotes no host ou adicionar dependências/alterar lockfile da aplicação. O browser inicial falhou por dependências de ambiente; uma primeira rodada corrigiu semântica de meter/CSS e a segunda corrigiu overflow a 768px. Checks finais passaram. O lint de especificidade descendente está suprimido somente no CSS da home, com justificativa: seletores escopados em componentes distintos e overrides responsivos intencionais. Outras regras continuam ativas.

Preview dev desta sessão: `http://127.0.0.1:5174/`, iniciado na worktree isolada. Login real/DB/worker não foram exercitados; API foi interceptada com 401 sintético para provar que o frontend preserva o bloqueio. Testes/backend/build de API e worker não executados neste ticket exclusivamente web.

## Riscos e limitações
Dados simulados não comprovam prontidão de contas/agentes/projetos. A referência central é PNG (~2 MiB), mantém texto/marca NÚCLEOS da arte original e não é uma ilustração SVG editável; os elementos externos usam La fabrique. Rótulos ficam menores no mobile. Avatares vetoriais simplificados diferem da referência. Navegação é estado interno, sem novas URLs públicas. Aceite visual humano pendente.

## Rollback
Antes da integração, basta voltar a usar a worktree/branch `developer`; ela permanece no baseline e sem alterações desta entrega. Após eventual integração autorizada, reverter o commit de código `904d0b5` restaura a entrada administrativa anterior. Nenhuma migration ou dado precisa ser revertido. Não foi realizado merge, deploy ou atualização do serviço original na porta 5173.

## Documentação atualizada
README raiz, README controle, INDEX documental, CHANGELOG, BACKLOG, ticket, relatório, capturas, resultado do browser, script e patch. ADRs/contratos/backend/operação não sofreram decisão estrutural nem alteração; conteúdo da home permanece local.

## Lessons
[Modais e estados de interface](../../../10-lessons/13-modal-edicao-configuracao.md): separar demonstração de dados administrativos; `dialog`/Escape/restauração de foco verificados em browser. [Baseline, regressão e review](../../../10-lessons/11-baseline-regressao-review.md): comparar aviso anterior, testes reais e fidelidade visual; renderização estática não substitui browser nem aceite.

## Uso de IA e custos
Provider da sessão: Codex. CLI instalado observado: `codex-cli 0.159.2`; `codex exec --help` consultado, sem lançar outra execução. Hierarquia desta tarefa: sistema/developer → instruções fornecidas pelo usuário → AGENTS.md local; nenhum AGENTS mais específico em apps/web. Não foram lidos/exportados tokens nem dados de autenticação. Modelo efetivo, modo/plano de cobrança, uso total e cota atual não foram comprovados pelo cliente nesta tarefa. Nenhuma chamada adicional a provider, API de IA, fallback pago, extra usage ou autorecharge habilitada. Custo incremental não calculado; downloads de ferramentas não são evidência de consumo de modelo. Um writer nesta worktree nova; zero handoffs, subagentes, merges ou deploys.

## Pendências e aceite
AWAITING_HUMAN: responsável deve revisar a composição/capturas e o código `904d0b5`. Revisão documental é o commit posterior deste relatório. Nenhum DONE presumido. Integração em developer e publicação permanecem dependentes de autorização aplicável.

## Atualização posterior — FAC-020A / FAC-021
O responsável autorizou integração em developer e remoção da worktree, e solicitou correção do nome para La fabrique e cena em 60%. Código atual cde0cc6 integrado; evidências anteriores descrevem sua revisão histórica. Correção nominal deste documento identificada pelo FAC-021; capturas/patches originais preservados. [Estado atual](2026-10-05-FAC-021-nome-escala-home.md).
