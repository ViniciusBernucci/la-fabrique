# FAC-015 — Preparação das assinaturas na VPS

Execute `npm run providers:setup` para provisionar raiz privada fora do repositório e caminhos dos clientes; reinicie `npm run dev`. Codex autentica pela tela após conta salva/habilitada e verificação de estado. Claude autentica na VPS com `npm run providers:login -- ID_DA_CONTA`; autorização ocorre no fluxo oficial e verificação é enviada ao worker ao concluir.

Setup foi executado nesta VPS e repetido de forma idempotente. Clientes reais Codex 0.159.2/Claude 2.1.285 pedem autenticação nas novas identidades. Nenhuma execução de ticket/migration foi habilitada. Login humano, catálogo autenticado, permissões operacionais e validação visual permanecem pendentes.

[Passo a passo, evidências e rollback completo](../configuracao/2026-10-05-FAC-015-contas-autenticadas-modelos.md). Credenciais privadas são preservadas no rollback e não passam pela API ou painel. Estado AWAITING_HUMAN.
