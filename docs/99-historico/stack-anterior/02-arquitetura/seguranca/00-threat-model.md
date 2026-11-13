> Leitura: [Índice didático](../../02-INDEX.md) · [Próximo →](01-rede-e-egress.md)

# Threat model e risco residual

Data: 2026-10-06 (America/Sao_Paulo). Status da arquitetura e controles: **PLANEJADO**. Implementação e eficácia: **NÃO VERIFICADAS**. Este documento especifica trabalho futuro; não comprova instalação, configuração ou execução.

## Ativos e adversário

Ativos: dados dos tenants, fontes Git, checkpoints, artefatos, credenciais de fornecedor, secrets do controle, banco, Redis, integrações, identidade do host, disponibilidade e veracidade do aceite. Adversário pode controlar todo conteúdo de um projeto/ticket/README/AGENTS de cliente, package scripts, página, MCP sugerido, log e output do modelo. Assumir prompt injection com sucesso e execução arbitrária dentro do sandbox, inclusive persistência e exfiltração tentadas. Ameaça inclui cliente A malicioso e falhas acidentais de associação A/B.

Raiz de confiança: administração do host, configuração/binários aprovados, kernel/perfil de isolamento, policy engine, broker, Executor e mecanismos de autenticação. Modelo não é parte dessa raiz. Provider Runtime administrado é confiável apenas para o contrato restrito; comprometer seu processo autenticado pode comprometer a auth daquela instalação, logo superfície local deve ser mínima.

## Fronteiras e ataques

| Ameaça | Barreira planejada | Prova exigida |
|---|---|---|
| Prompt pede outro cliente/controle | nenhum recurso/capability/rota correspondente | ataque via pedido de tool e comando direto negado |
| IDOR/confused deputy | identidade derivada, cadeia/FK/RLS e revalidação | trocar IDs em APIs/filas/instalação/artefatos |
| Script lê auth/environment | credencial só runtime, ferramentas sem shell local | canários em runtime inacessíveis a sandbox e modelo |
| Config/hooks/MCP maliciosos | descoberta desativada, perfis administrados | repo tenta habilitar shell/endpoint externo |
| Escape de path/coleta | roots por descritor e validação sem links externos | traversal, symlink race, archive slip |
| SSRF/lateral/metadata | firewall/egress effective-IP deny | IPv4/IPv6, redirecionamento, rebinding, IP público do controle |
| Roubo/replay de capability | TTL, audience, channel binding, revoke, fencing | reuse em attempt/tenant diferente e após cancel |
| Sessão/cache contaminado | namespace por tenant/projeto/attempt | segredo sintético A não aparece em B nem A2 |
| Processos após handoff | cgroup kill + quiescência antes de novo writer | partição/lease/crash; sem dupla escrita |
| Artefato/log injeta comando/UI | parsing/sanitização/escape, sem ação automática | HTML/ANSI/control chars e evento administrativo forjado |
| DoS/fork bomb/disco | cgroups, quotas, admission e reservas | teste controlado sob carga preserva API/banco |
| Tenant cruza dados por SQL/cache | RLS e auth de projeto, queries parametrizadas | pool alternado e consultas sem filtro |

## Prompt injection

Conteúdo de projeto e web tem proveniência e é dado não confiável. Políticas e manifestos aprovados ficam RO e fora do alcance de edição do modelo. Detectores, regras no prompt e secret scanners reduzem risco, mas não constituem fronteira. A prova exige tentar ações proibidas mesmo quando o modelo não as pede; nenhuma suite depende de o modelo “resistir”. Não ampliar permissão para resolver TOOL_DENIED. Não enviar dados internos da fábrica ao modelo como contexto auxiliar.

## Riscos residuais e decisão de admissão

VPS compartilhada não garante impossibilidade universal: escape do kernel/runtime, root comprometido, supply chain dos binários, falha do broker, side channels e DoS do host podem atravessar limites. RLS não protege contra DBA nem SQL injection com autoridade de configuração; HOME separado não protege processos do mesmo UID; assinatura do envelope não corrige autorização errada. Registros de testes são evidência dos cenários executados, não prova de ausência de vulnerabilidade.

Piloto: código próprio e dados sintéticos, container endurecido, um executor. Multi-tenant hostil: bloquear admissão até perfil mais forte (gVisor/microVM avaliado na mesma VPS), suite adversarial e revisão de segurança; registrar limitações restantes. Se VPS não suporta perfil requerido, suspender esse nível de uso e apresentar alternativa dedicada para decisão, sem trocar topologia automaticamente. Separação física futura e alta disponibilidade continuam fora desta implementação inicial.

Fornecedor externo vê o contexto autorizado; política de retenção/uso depende da modalidade validada, sem promessa universal. Segredos do cliente também devem ser excluídos do snapshot. Ausência de segredo da fábrica no contexto é controle de origem; redaction não remove dados já enviados anteriormente.

## Resposta a incidente

Kill switch global/por tenant: negar novos claims, revogar capabilities, interromper processos, congelar artefatos em quarentena, preservar evidências sanitizadas. Revogar/renovar auth afetada por fluxo oficial e credenciais internas por mecanismo administrativo. Investigar alcance real sem afirmar isolamento intacto. Reativar só após causa corrigida, nova configuração testada e aceite; restore não restaura automaticamente confiança nem sessões.


## Origem desta edição

[Versão original preservada](../../99-historico/originais/output/le-fabrique-multitenant/documentacoes/seguranca/THREAT-MODEL.md). Migração editorial de paths em 2026-10-06; conteúdo de engenharia continua proposto.
