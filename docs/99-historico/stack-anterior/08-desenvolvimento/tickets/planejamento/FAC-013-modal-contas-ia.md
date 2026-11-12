# FAC-013 — Lista de contas IA com configuração em modal

Status: AWAITING_HUMAN. Data: 2026-10-05. Baseline `5f642d2`; developer diretamente por instrução do responsável. Provider elegível: nenhum. Escopo web/configuração/documentação; sem banco, migrations, login ou execução IA. Até duas rodadas de correção.

## Objetivo e critérios

Listar contas de forma compacta mostrando nome/provider/estado; clicar em Configurar ou Adicionar conta abre modal com os campos existentes. Edição local ao modal, cancelar/Escape descarta, salvar persiste via API existente e fecha só após sucesso. Preservar ações de verificação/login/remoção, regras de atribuição/modelos e limites de até 20 contas. Dialog nativo com título acessível, foco modal e retorno de foco. Layout responsivo. Checks web, lint, documentação/lessons/diff sanitizado; não DONE sem aceite humano exato.

## Resultado

Código `54475ba`; checks web/lint passaram. Validação manual do modal pendente. [Evidências](../../../09-entregas/2026/configuracao/2026-10-05-FAC-013-modal-contas-ia.md).
