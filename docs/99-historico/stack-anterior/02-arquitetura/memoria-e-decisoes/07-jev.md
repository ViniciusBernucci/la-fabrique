> Leitura: [← Anterior](06-modelo-local.md) · [Índice didático](../../02-INDEX.md) · [Próximo →](08-aprendizado.md)

# JEV: julgamento entre alternativas permitidas

## Papel do JEV

JEV é um serviço externo de decisões estruturadas da TypeSafe. Nesta arquitetura, recebe um estado pequeno e perguntas delimitadas. Um estado é a informação necessária à pergunta; não é todo o repositório. Sua integração é futura, mas já está prevista uma exceção específica à restrição de APIs.

A [introdução oficial](https://docs.typesafe.ai/introduction) descreve três primitivas: **Choice**, escolher entre opções; **Score**, avaliar por uma escala; **Noul**, julgar uma afirmação numa saída entre zero e um. São formatos de pergunta, não permissões para agir.

Exemplo didático: o motor encontra dois documentos opcionais sobre retomada. Pode perguntar qual merece leitura primeiro. Não pergunta se pode ignorar confirmação de parada do writer: isso é requisito obrigatório.

## Como a chamada será controlada

O worker aplica regras e filtra candidatos; prepara estado sanitizado; confere configuração, credencial e limites; chama JEV; valida a resposta; revalida restrições; registra resultado e uso observado. Credencial nunca entra em pacote de contexto ou sandbox do agente.

O [quick start oficial](https://docs.typesafe.ai/introduction/quickstart) apresenta POST em `https://api.typesafe.ai/v1/systemone`, autenticação Bearer, campos state/model/questions e resposta com answers e uso. Esses dados foram consultados em 2026-10-09; a implementação futura precisa revalidar contrato e versão.

## Confiança e fallback

Uma distribuição de probabilidades expressa como o modelo distribui alternativas. O valor confidence resume concentração para Choice/Score; Noul tem outro formato. Conforme a [documentação de confiança](https://docs.typesafe.ai/confidence), confiança não deve ser tratada como porcentagem universal de acerto. Perguntas precisam de avaliação e limiares próprios.

Se JEV não estiver configurado, estiver indisponível ou não resolver a dúvida, usar LLM elegível. Se ainda houver dúvida, perguntar ao responsável. Não ativar API de LLM paga para fazer esse fallback.

## Configuração pelo sistema

O painel deverá apresentar ativação, modelo, referência da credencial, decisões permitidas, timeout, repetição e limites de gasto/chamadas. Nenhum teto foi escolhido nesta conversa: o responsável o preencherá no sistema. Configuração incompleta significa “não pronto”, e o fallback permanece possível.

Resposta perdida após envio pode ter consumido orçamento mesmo sem decisão disponível. Repetir às cegas pode duplicar cobrança. O plano prevê estado desconhecido, reserva de limite e reconciliação; suporte do fornecedor a idempotência precisa ser confirmado.

A exceção permite projetar uso do JEV, mas esta entrega não implementa integração nem realiza chamadas autenticadas. Leia o [plano](../../08-desenvolvimento/planos-memoria-e-decisoes/07-jev.md).
