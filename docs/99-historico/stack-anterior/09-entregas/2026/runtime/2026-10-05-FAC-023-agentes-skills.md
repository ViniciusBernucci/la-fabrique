# FAC-023 — Agentes e skills: runtime
Data: 2026-10-05. AWAITING_HUMAN. Código `a25638acde42ba50576908f1bf6725570a360a99`; baseline 30fbd91.
O cadastro aditivo é documentado integralmente no [relatório de configuração](../configuracao/2026-10-05-FAC-023-agentes-skills-projeto.md), incluindo critérios, contratos, checks, diff, limites, rollback e aceite.
Esquemas estritos aceitam cadastros opcionais, validam IDs/referências/projeto e conta/modelo. Snapshot interno transporta registro validado; router mantém funções existentes. 51 contratos/65 runtime/215 worker passaram. Não há instalação automática de skills nem ampliação de writers.
