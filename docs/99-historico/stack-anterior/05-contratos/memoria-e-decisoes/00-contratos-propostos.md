> Leitura: [Índice didático](../../02-INDEX.md) · [Próximo assunto →](../../08-desenvolvimento/planos-memoria-e-decisoes/00-README.md)

# Contratos comuns propostos — memória e decisões

Status: PROPOSTO PARA AVALIAÇÃO, implementação PLANEJADA. Base observada `896ef0d2c804b80711d5ad00ad96695eaaf9f2d7`. Este capítulo define semântica comum, não schemas executáveis ou rotas já existentes. [Planos](../../08-desenvolvimento/planos-memoria-e-decisoes/00-README.md) detalham contratos por feature; [Zod atual](../../../packages/contracts/src/index.ts) permanece fonte canônica do software implementado.

## Finalidade e envelope de identidade

Toda consulta precisa declarar qual conhecimento e código usa. Propõe-se envelope compartilhado com `schemaVersion`, `requestId`, `taskId/taskVersion`, `runId/attemptId` quando aplicáveis, `baseRevision`, `codeRevision`, `memoryRevision`, `overlayId/overlayHash`, `policyVersion` e `configurationVersion`. Identificadores externos e revisão são validados pelo servidor; texto de modelo não define ownership. Campo ausente não deve ser preenchido com SHA fictício.

`baseRevision` identifica o início; `codeRevision` identifica o código avaliado quando houver commit. Alterações ainda não commitadas usam hash do patch e manifesto de untracked sanitizado. `memoryRevision` identifica o conjunto de conhecimento aplicável, incluindo regra aceita ainda não incorporada ao código. Essa condição é explícita por item.

## Referência de fonte

`SourceReference` proposto contém ID, caminho relativo autorizado, revisão Git, SHA-256 dos bytes, tipo de fonte, seção/intervalo quando aplicável e hash do trecho. Fonte binária/segredo/symlink é excluída segundo regras atuais. Um hash válido prova identidade, não veracidade, autorização ou suficiência.

Fonte pode estar indisponível ou em revisão diferente: retornar erro/lacuna, nunca reconstruir um texto a partir da lembrança da LLM. Contratos normativos precisam de original recuperável além de resumo explicativo.

## Estados sem alterar enums existentes

| Entidade lógica | Dimensões propostas | Autoridade |
|---|---|---|
| Conhecimento | Observação/proposta/aceita; vigente/substituída; incorporada ou pendente; conflito | Responsável aceita regra; worker observa código |
| Índice | Pendente/processando/completo/incompleto/falhou | Worker comprova cobertura do inventário |
| Pack | Candidato/válido/bloqueado; lacunas/omissões | Validador determinístico sobre obrigações conhecidas |
| Decisão | Resolvida/incerta/bloqueada | Estratégia sugerida + validação de política |
| Chamada externa | Não enviada/enviada/conhecida/desconhecida | Evidência de transporte e fornecedor |

As dimensões são separadas. Conhecimento aceito pode aguardar incorporação. Resultado de modelo resolvido não significa tarefa DONE. Novos enums dependem de schema/migration futura; não substituir Ticket/Run/Attempt atuais.

## Operações, efeitos e erros propostos

| Operação | Entrada mínima | Resultado/efeito | Falhas relevantes |
|---|---|---|---|
| Propor conhecimento | Fonte/revisão, afirmação, natureza e evidência | Nova versão candidata | Fonte ausente, escopo inválido |
| Aceitar conhecimento | ID/hash esperado, revisão, responsável | Aceite e publicação vinculados | Hash obsoleto, evidência insuficiente |
| Indexar | Revisão/overlay, extrator e escopo | Geração candidata/publicável | Fonte inválida, geração incompleta |
| Consultar impacto | Revisão, componentes e limites | Relações, obrigatórios e lacunas | Índice obsoleto, limite atingido |
| Montar/expandir pack | Tarefa/papel/revisão/política e limites | Pack imutável e manifesto | Obrigatório ausente, conflito, orçamento de conteúdo |
| Decidir | Tipo, candidatos permitidos, política e evidências | Sugestão auditada ou escalonamento | Bloqueio, insuficiência, resposta inválida |
| Configurar JEV | Versão esperada e parâmetros pelo sistema | Configuração validada ou não pronta | Credencial/limite/tarifa ausente |
| Rotular exemplo | Candidato/hash, rótulo e evidência | Rótulo validado/histórico | Evidência ausente, versão divergente |

Nomes de erros deverão ser normalizados na implementação: revisão obsoleta, fonte indisponível, conflito relevante, obrigatório ausente, limite de recurso, configuração incompleta, serviço indisponível, resultado desconhecido. Mapeamento HTTP é proposta pendente; não inventar endpoint existente.

## Idempotência, ordenação e consistência

Repetir mesma operação com mesma chave e entrada não duplica aceite ou geração. Mesma chave com entrada diferente é recusada. Atualização compara versão esperada; resultado antigo não substitui uma revisão nova. Publicação e evento usam transação/outbox quando pertencem ao mesmo banco. Entrega duplicada da fila é reconciliada.

Aceite publica memória antes de merge, conforme requisito confirmado. Seleção cruza autoridade da regra com aplicabilidade à revisão de código; desacordo vira divergência, não alteração automática do código. Índice não concluído fica identificado; reconstrução posterior não apaga aceite.

Idempotência interna não prova idempotência do JEV. Depois de envio com resposta desconhecida, reserva de orçamento não é liberada como se nada tivesse ocorrido. Contrato de reconciliação fornecedor precisa ser validado antes de repetição paga automática.

## Segurança, limites e compatibilidade

API valida identidade/configuração; worker executa extração e chamadas; agentes não recebem acesso geral ao banco ou credenciais. Instalações oficiais de LLM permanecem subscription-only. Exceção JEV tem configuração separada, allowlist de tipos de decisão, orçamento e uso auditados. Não alterar `monthlyApiBudget: 0` de runtimes oficiais para ligar JEV.

Controles de paths, sanitização e proveniência não garantem que nenhuma informação sensível exista: revisar dataset/trechos, rejeitar padrões proibidos e registrar lacunas. Prompt injection em fonte ou resposta não muda política. Limites por payload, chamada, tarefa, período e host ficam configuráveis; valores devem ser preenchidos ou medidos.

Contratos v1 atuais continuam válidos. Definir v2/contrato novo e negociação de consumidor antes de rollout. Mudanças de modelo, extrator, chunking, política e configuração invalidam resultados pertinentes. Rollback preserva registros e permite baseline apenas quando cumpre as obrigações vigentes.

## Cenários verificáveis para implementação posterior

Testar revisão divergente, fonte removida, hash errado, proposta alterada antes do aceite, índice de outra branch, atualização duplicada, publicação atrasada, obrigatório acima do limite, segredo em fonte, opção de modelo não permitida, credencial JEV ausente, dois jobs concorrendo por orçamento e resposta externa desconhecida. Evidências devem identificar revisão/configuração e distinguir teste sintético de execução real.

Nenhum teste de aplicação foi executado para estes contratos propostos. Formatos finais de persistência, retenção e override em tentativa ativa aguardam avaliação conforme [pendências](../../08-desenvolvimento/planos-memoria-e-decisoes/01-decisoes-e-dependencias.md).
