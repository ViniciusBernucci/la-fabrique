# ADR-0041 — Matriz de evidências derivada das fontes

- **Estado:** ACEITA · **Ticket:** [T17-01](../NEW%20MVP/03-BACKLOG.md#t17-01) · **Épico:** E17 · **Relacionadas:** [05-ACEITE-E-PILOTO](../NEW%20MVP/05-ACEITE-E-PILOTO.md), [04-RASTREABILIDADE](../NEW%20MVP/04-RASTREABILIDADE.md)

## Decisão
1. **A matriz não é escrita à mão:** `make matriz` (`cmd/matriz`, `internal/acceptance`) a gera de `05-ACEITE-E-PILOTO.md` (casos P01–P18 e seus épicos), `03-BACKLOG.md` (critério de aceite de cada ticket) e `07-CHECKLIST.md` (estado e entregas). Sai em `docs/aceite/02-matriz-de-evidencias.md`.
2. **Duas visões:** (A) por caso P01–P18, com os épicos, a evidência mínima exigida, a **evidência de software hoje** (tickets feitos/parciais/pendentes por épico, com os abertos listados) e a coluna **evidência no projeto real**; (B) por épico e ticket, com o critério do ticket, o estado e os registros de entrega.
3. **Evidência de software não substitui a do piloto:** a coluna do projeto real sai sempre "pendente" até alguém registrá-la; o aceite exige o projeto real (D95).
4. **Rastreabilidade verificada por teste:** todo ticket do checklist existe no backlog e vice-versa, tem critério de aceite e pertence a um épico; todo caso aponta para épicos existentes e todo épico é coberto por ao menos um caso; toda referência de ticket da matriz de decisões (D01–D95) existe. O teste também compara o arquivo versionado com o gerado: **depois de alterar o checklist, rode `make matriz`** e inclua o arquivo no commit.
5. **Ficha de preparação pré-preenchida** (`docs/aceite/01-ficha-de-preparacao.md`): um campo por linha da ficha de 05, com o que o repositório prova e o que fica **A PREENCHER**, mais uma tabela de pendências F01–F12 com o motivo de cada uma não ter sido feita pelo agente. Um teste confere que todos os campos e todas as pendências existem.

## Limites
- A seleção do projeto e da feature reais, as contas, as credenciais, a VPS e o repositório do piloto são decisão ou acesso do responsável: **o contrato não está completo** enquanto F01–F12 estiverem abertas, e T17-01 fica parcial.
- A matriz mede cobertura por ticket, não prova que o critério foi atendido no piloto; isso é o T17-02/T17-03.
