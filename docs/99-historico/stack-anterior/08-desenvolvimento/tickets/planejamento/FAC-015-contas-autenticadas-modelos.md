# FAC-015 — Contas oficiais autenticadas e modelos acessíveis

Status: AWAITING_HUMAN (execução iniciada em READY). Data: 2026-10-05. Baseline `a9aea9e`; developer diretamente conforme responsável. Provider elegível: nenhum para inferência; clientes oficiais somente login/status/catalogo. Sem API keys/extras/fallback. Até duas rodadas de correção.

## Objetivo e critérios

Configurar identidades privadas dos clientes no ambiente dev, integrar catálogo Codex via app-server model/list após auth ChatGPT e refletir modelos nas escolhas por função. Preservar fluxo oficial de login Codex. Oferecer comando de login Claude por ID de instalação sob a mesma identidade privada; confirmação humana no browser/terminal, sem exportar tokens. Verificação posterior publica apenas estado/modelos/metadados. Claude não fornece catálogo garantido via status: nomes configurados continuam explícitos e sem alegar entitlement. Antigravity sem adapter continua indisponível para execução.

Escopo worker/API/web/scripts/configuração/docs, sem migrations, liberação do consumer/writer ou prompts de IA. Prover setup idempotente local sem sobrescrever .env; explicitar passo dependente do usuário. Novos modelos observados persistidos no catálogo e visíveis em funcionários; dados runtime validados. Checks relevantes, docs/lesson/diff/evidência reais. Não DONE sem aceite exato, nem afirmar login real sem prova.

Entrega implementada e checks automatizados concluídos; aceite exato/login humano pendentes. [Relatório](../../../09-entregas/2026/configuracao/2026-10-05-FAC-015-contas-autenticadas-modelos.md).
