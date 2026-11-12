# GitHub: repositório, PRs e proteção

Remoto atual: `origin` → `github.com/ViniciusBernucci/le-fabrique` (privado a confirmar). Estado de [T01-02](../NEW%20MVP/03-BACKLOG.md#t01-02): **parcial**.

Feito: template de PR (`.github/pull_request_template.md`), convenções ([01-convencoes.md](../desenvolvimento/01-convencoes.md)).
**NOT_RUN** (a `gh` não estava autenticada e alterar o repositório remoto exige sua autorização):
1. Confirmar visibilidade privada e o plano. *Proteção de branch/rulesets em repositório privado exige GitHub Pro/Team/Enterprise; no plano Free isso não está disponível — registre aqui qual plano vale.*
2. Proteger `main`: PR obrigatório, checks obrigatórios (após o CI do T02-03), sem push direto; com um único administrador, decidir se o proprietário pode contornar (exceção documentada).
3. PR de teste passando pelo fluxo; conferir permissões da identidade usada.
