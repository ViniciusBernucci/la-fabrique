> Leitura: [← Anterior](01-as-is.md) · [Índice didático](../02-INDEX.md) · [Próximo →](03-containers.md)

# C4 nível 1 — contexto atual

Base a2cc5e0, estrutura inspecionada, eficácia de implantação NÃO VERIFICADA nesta migração.

```mermaid
flowchart LR
  U[Operador administrativo] -->|navegação, tickets e aceite| F[La fabrique]
  F -->|cliente oficial e contexto autorizado| AI[Fornecedores de inferência]
  F -->|Git e gh no worker confiável| G[Repositórios Git / GitHub]
  F -.->|backup offline; externalização pendente| S[Armazenamento externo]
```

Inferência ocorre no fornecedor. Cadastro de conta não prova autenticação/eligibilidade. GitHub tem fluxos implementados sob gate; integração real exige evidência própria. Home inicial é demonstrativa. Nenhum sistema externo concede autoridade por output de modelo. [Contexto alvo posterior](10-contexto-alvo.md).
