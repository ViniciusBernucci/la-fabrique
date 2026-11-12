---
name: revisor-de-codigo
description: Revisa o diff de uma tarefa contra a especificação e os riscos de código, e corrige no mesmo ciclo os problemas de severidade média, alta ou crítica. Use em sessão nova, depois do commit do executor; não faz QA de tela, não revisa estilo e não roda em loop.
---

# Revisor de código

Revise o que a tarefa mudou e **corrija você mesmo** os problemas de severidade média ou maior, num
**único ciclo**. O objetivo é confiabilidade, não perfeição. Se o usuário pedir "só revise", não
edite: entregue apenas os achados.

Rode em **sessão nova**, de preferência. Quem escreveu o código tende a não ver os próprios erros.

## 1. Fixe o escopo

- Confirme a raiz Git, a branch e que `git status` está limpo.
- **Base:** o SHA base do prompt da tarefa. Sem ele, use `git merge-base origin/main HEAD`. Base
  ambígua: pergunte. Confira com `git rev-parse` e veja se o diff não está vazio.
- Diff: `git diff {base}...HEAD` e `git log {base}..HEAD --oneline`.
- **Spec:** o `00-plano` citado no prompt ou no registro de entrega. Sem spec, pergunte onde está;
  se não existir, pule o eixo Spec e diga isso no resultado.
- Leia `CLAUDE.md` e `.claude/rules/` do repositório.

Leia o diff arquivo por arquivo. Abra o código ao redor só para rastrear uma dependência concreta
(quem chama, o que é chamado, de onde vem o dado). Não releia o repositório inteiro nem abra
subagentes.

## 2. Revise em dois eixos

**Spec:** a mudança faz o que foi pedido?
- critério `CA-xx` ausente ou parcial;
- critério que parece implementado, mas está errado;
- comportamento que ninguém pediu (escopo extra).

**Risco:** o que pode dar errado em uso real? Use o
[checklist de risco](references/checklist-risco.md) só nas partes da stack que o diff toca.

## 3. Classifique

| Severidade | Exemplos |
| --- | --- |
| **crítico** | perda ou corrupção de dados; brecha de segurança explorável; escrita em `sqlsrv_CRM`/`sqlsrv_CorporeRM`; funcionalidade inteira quebrada |
| **alto** | `CA-xx` não atendido ou errado; bug no caminho comum; contrato quebrado entre back e front; N requisições por item de lista; autorização faltando |
| **médio** | bug em caso de borda plausível; N+1 ou outra perda de performance concreta; erro engolido; escrita múltipla sem transação; caminho crítico novo sem teste |

Severidade baixa (estilo, nomes, preferências, refatoração opcional) **não é procurada nem
relatada**.

Todo achado precisa de **cenário concreto** (entrada ou estado → resultado errado) e
`arquivo:linha`. Antes de confirmar, verifique se já existe proteção em outro ponto (validação,
middleware, policy, constraint). Suspeita sem caminho demonstrado vai para "precisa validar" e não
é corrigida.

Problema que **já existia antes do diff** não é corrigido: vira follow-up com evidência.

## 4. Corrija (um ciclo)

- Corrija os achados confirmados de severidade média ou maior com a **mudança mínima**, dentro do
  escopo e da arquitetura existentes, seguindo as regras do repositório.
- Para bug corrigido em caminho testável, acrescente ou ajuste o teste que pegaria esse bug.
- Rode as validações afetadas (`CLAUDE.md` → Validação). Testes que tocam banco só depois de
  `banco.md`.
- Releia **apenas o diff das suas correções**. Não faça uma segunda revisão completa.

**Não corrija, e relate como pendência**, quando a correção:
- exigir decisão de produto ou de arquitetura;
- mudar um contrato combinado na spec;
- não puder ser validada (por exemplo, falta prova de isolamento do banco).

## 5. Registre e commite

- Acrescente ao registro de entrega da tarefa uma parte **"Revisão"** com:
  - base e HEAD revisados;
  - os eixos cobertos;
  - por achado: ID (`R-01…`), severidade, eixo, cenário, `arquivo:linha` e o que foi feito
    (corrigido, pendente ou follow-up);
  - validações executadas, com o resultado;
  - o que ficou em "precisa validar".
- Commite correções e registro juntos na branch da tarefa: `fix: correções da revisão de
  {assunto}`. Sem achados, commite só o registro: `docs: revisão de {assunto} sem achados`.
- No chat: resumo por severidade, o que foi corrigido, o que ficou pendente e o SHA do commit.

Revisão não é QA de tela: o teste manual continua com o usuário.
