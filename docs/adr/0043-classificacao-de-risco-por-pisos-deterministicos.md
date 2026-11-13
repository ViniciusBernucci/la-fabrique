# ADR-0043 — Classificação de risco com pisos determinísticos

- **Estado:** ACEITA · **Ticket:** [T14-02](../NEW%20MVP/03-BACKLOG.md#t14-02) · **Épico:** E14 · **Relacionadas:** [ADR-0035](0035-plano-versionado-da-feature.md), [ADR-0040](0040-checks-baseline-e-classificacao-de-falhas.md)

## Decisão
1. **Nível final = máximo entre a sugestão da IA e os pisos determinísticos.** Os pisos só **elevam**: uma sugestão baixa não rebaixa um arquivo crítico, e uma sugestão alta não é reduzida pelos pisos. Sugestão ausente ou inválida é tratada como **alta** (falha fechada).
2. **Pisos por caminho** (glob com `*` no segmento e `**` entre diretórios) e por **tamanho do diff** (linhas e arquivos alterados), por projeto, com padrão seguro: migrations/SQL, autenticação, segurança/criptografia, segredos, `.env`, CI, Dockerfile/compose → **alto**; manifestos e locks de dependência e configuração → **médio**; a partir de 200 linhas ou 10 arquivos → médio, 800 ou 30 → alto. A política do projeto (`risk_policies`) substitui a padrão e é validada (piso só médio/alto, padrão relativo sem `..`, limiares coerentes).
3. **Duas fases de avaliação:** no **planejamento**, com os caminhos previstos da tarefa (`FromTask`); depois da **implementação** e de cada **correção**, com o **diff real**. Na reavaliação só valem os arquivos realmente alterados: o nível pode cair se o diff não tocou o que estava previsto, e sobe se tocou algo crítico.
4. **Justificativa auditável:** cada classificação grava as razões (`ai`, `rule`, `size`) com o arquivo e a regra que exigiu cada nível, a revisão e os arquivos; o histórico fica por tarefa e a mais recente vale.
5. **Os três níveis escolhem verificações previstas** (`risk.Plan`): baixo = `lint`, `build`, `test`; médio = o mesmo + revisão padrão + E2E se houver UI; alto = o básico + `secret_scan`, `dependency_audit`, `static_analysis` + revisão independente de segurança + E2E obrigatório. Os nomes que o projeto não sabe executar saem em `not_applicable` (precisam de justificativa), nunca omitidos em silêncio.

## Limites
- Não há geração do diff (git) nem ligação ao ciclo de execução: quem chama informa a sugestão da IA, a revisão e os arquivos. O agente que produz a estimativa é o planejador (T10-01 define o campo `risk`).
- Os checks de segurança são nomes: executá-los exige comandos no perfil do projeto (T05-03) e aplicá-los é o T14-01/T14-03; a revisão e o E2E de verdade são T14-03 e T14-04.
- Sem rotas nem tela para editar a política.
