# FAC-016 — Retorno da verificação Codex no modal

Estado AWAITING_HUMAN. Baseline f075add; código ad78c84 em developer conforme autorização do responsável. Nenhum provider usado para inferência; nenhum login, migration, alteração de plano ou reinício de processo realizado pelo agente.

## Problema e correção

O clique chegava à API/outbox/worker e terminava FAILED em milissegundos. Inspeção limitada às variáveis operacionais mostrou que os processos antigos não tinham WORKER_PROVIDER_ROOT nem WORKER_CODEX_BINARY. A raiz havia sido provisionada no .env, mas o supervisor ainda não tinha sido reiniciado. O modal mostrava somente status técnico e a mensagem global ficava fora do dialog.

O modal agora mostra envio imediato, espera pelo worker, processamento e resultado em português com role=status/aria-live. Desabilita novos pedidos durante envio/verificação ativa; falha ao solicitar ou consultar resultado aparece dentro do modal com role=alert. A exceção conhecida de raiz ausente recebe mensagem fixa com setup/restart; erros desconhecidos continuam sanitizados, sem logs brutos ou tokens. Estado AUTH_REQUIRED explica necessidade de conectar assinatura com conta habilitada/salva.

## Evidências

- Histórico real: três pedidos Codex de 18:46 UTC terminaram FAILED; clientes/processos antigos sem variáveis operacionais confirmados, sem ler/publicar segredos.
- Teste real após atualização: POST local `/api/settings/installations/codex-default/verifications` retornou 201, ID `83098492-a06f-4cd4-9b5c-f665e796f8e3`; worker concluiu COMPLETED/AUTH_REQUIRED com `codex-cli 0.159.2`, modelos vazios. O worker passou a acessar o cliente; não atribuímos o reinício ao agente.
- Leitura posterior `/api/settings`: versão 15, codex-default habilitado e AUTH_REQUIRED. Alteração/habilitação do usuário preservada.
- 35 testes web e 209 worker passaram; build web (inclui TypeScript), typecheck/build worker e lint passaram. Um warning anterior em run-delivery.service.ts permanece. Novos casos verificam mensagens por estado, diagnóstico da raiz e não exposição de exceção privada.
- Sem navegador automatizado disponível: interação visual/login real depende do usuário. Nenhuma inferência ou catálogo autenticado comprovado.

## Uso e rollback

Recarregue a página, abra o modal Codex e use Verificar conta e modelos. Com a conta já habilitada e AUTH_REQUIRED, clique em Conectar assinatura Codex e autorize o fluxo oficial. Se reaparecer a falta de raiz, rode providers:setup e reinicie npm run dev; apenas editar .env não atualiza processos antigos.

Reverter ad78c84 restaura os textos e feedback anteriores sem modificar contas, sessões privadas ou schema. [Diff sanitizado](../evidencias/configuracao/evidencias/FAC-016-diff.patch), formato sem contexto, compatível com git apply --unidiff-zero sobre baseline apropriado. Aceite visual/humano pendente; documentação e lesson atualizadas.
