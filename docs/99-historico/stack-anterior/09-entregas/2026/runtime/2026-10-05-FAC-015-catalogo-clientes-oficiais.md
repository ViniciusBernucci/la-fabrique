# FAC-015 — Descoberta de modelos no cliente oficial

O worker consulta metadados Codex após provar autenticação ChatGPT na identidade privada da instalação. Usa initialize/account-read/model-list via stdio, sem inferência, com validação runtime, paginação limitada e saída sanitizada. Somente modelos não ocultos são incorporados à configuração. Claude mantém modelos explícitos; o status da assinatura não é tratado como catálogo. Gates de execução/confinamento permanecem obrigatórios.

[Comportamento, fontes, testes, limitações e rollback](../configuracao/2026-10-05-FAC-015-contas-autenticadas-modelos.md). Estado AWAITING_HUMAN: clientes reais AUTH_REQUIRED, catálogo autenticado pendente. Não houve sessão pessoal reutilizada nem exportação de credenciais.
