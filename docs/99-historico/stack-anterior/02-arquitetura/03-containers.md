> Leitura: [← Anterior](02-contexto.md) · [Índice didático](../02-INDEX.md) · [Próximo →](04-componentes.md)

# C4 nível 2 — containers atuais

Base a2cc5e0. Unidades C4 descrevem processos/armazenamento, não necessariamente containers Docker. Código/Compose presentes, deploy efetivo NÃO VERIFICADO.

```mermaid
flowchart TB
  U[Navegador / painel React] -->|HTTP dev; HTTPS em implantação autorizada| P[Proxy Nginx / build web]
  subgraph V[VPS Linux única]
    P -->|API Bearer; SSE| A[API NestJS modular]
    A -->|Prisma / transações / outbox / eventos| D[(PostgreSQL)]
    A -->|dispatcher BullMQ / desafio efêmero| Q[(Redis)]
    W[Worker Node TS / supervisor host] -->|consumo BullMQ| Q
    W -->|Bearer interno: claim, lease, resultado, heartbeat| A
    W -->|lifecycle e rota por função| R[Cliente oficial / adapter]
    H[Auth privada por instalação fora de código] -->|somente cliente oficial| R
    R -->|ferramentas sob perfil nativo específico| T[Worktree do projeto]
    W -->|systemd, cgroups, namespaces| B[Sandbox de checks]
    T -->|workspace permitido| B
    W -->|snapshot e journal privados| E[(Evidências locais)]
  end
  R -->|inferência externa| AI[Fornecedor]
```

Outbox no PostgreSQL; Redis fila/cache efêmero não é autoridade de stop. API e Redis em loopback para worker host, PostgreSQL sem porta pública no Compose controle. Nginx bloqueia internal no canal público. Worker não é iniciado pelo Compose. Supervisor faz checkout/coleta fora da sandbox. Checks não acessam controle/auth/DB/Redis/daemon; perfil de ferramentas do cliente exige prova independente, sem inferir eficácia pelo desenho.

AS-IS não possui broker remoto. [Containers alvo](11-containers-alvo.md) conserva auth segregada/runtime por tenant/broker sem controle e sandbox efêmera, todos como proposta posterior. Cliente sem ponte suportada permanece incompatível com esse perfil; sem contorno OAuth/API/auth mount.
