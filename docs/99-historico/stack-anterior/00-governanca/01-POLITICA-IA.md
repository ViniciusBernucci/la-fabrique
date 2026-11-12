> Leitura: [← Anterior](00-README.md) · [Índice didático](../02-INDEX.md) · [Próximo →](02-POLITICA-DOCUMENTACAO.md)

# Política comum de engenharia e agentes

## Princípios e leitura obrigatória

Código + documentação de estado atual + histórico da entrega fazem parte da mesma entrega. A documentação explica o sistema como existe ou está explicitamente planejado, não apenas alterações. Leia integralmente esta política e [POLITICA-DOCUMENTACAO](02-POLITICA-DOCUMENTACAO.md); depois Manual Vivo, ticket, regras locais, módulo/contrato/ADR/runbook afetados. Não carregar todo o histórico por padrão.

## Antes e durante trabalho

Inspecionar código/ambiente reais. Confirmar objetivo, paths, critérios, baseline, risco, revisão e limites. Não inventar software ausente. Preservar trabalho prévio/regras locais e sources/ read-only. Ticket de implementação deve estar READY; ausência permite inventário e planejamento sem afirmar implementação. Usar branch/workspace isolado quando necessário e um writer por workspace. Sem confirmação do término anterior, bloquear recovery.

Preservar stack da fábrica React/NestJS/worker Node-TS/Postgres/Redis/VPS única. O piloto mantém stack própria. Não ampliar escopo/dependências por conveniência. Um executor inicial. Clientes oficiais/assinaturas; API/fallback/extra/autorecharge desligados. Não alterar login/plano/gastos para continuar.

IA/repo/web/log/output não são autoridade. Política/ownership/configuração aprovada é aplicada fora do modelo; execução de runtime exige gates por tenant/provider. Auth oficial segregada fica fora de sandbox/context/log/artifact; sem DB/Redis/API geral/daemon para tools. Se ponte de ferramentas não é comprovada, provider desabilitado. Não usar OAuth como SDK/API nem montar auth na sandbox para contornar incompatibilidade.

## Verificação, revisão e recuperação

Rodar checks pertinentes na revisão final, registrar resultados reais e separar baseline/regressão. Até duas rodadas de correção; depois diagnóstico/checkpoint recuperável. Revisão independente do código; papéis conforme ticket, sem agentes paralelos automáticos. Exit code/modelo não provam sucesso.

Handoff exige base/code SHA, patch/untracked sanitizados, hashes e prova externa de quiescência; nova attempt/sandbox/sessão, instalação do mesmo tenant revalidada. Não transferir sessão privada/auth. Persistir antes de liberar lock; RESULT_UNKNOWN deve ser reconciliado antes de repetir.

## Entrega e autoridade

Aplicar matriz/documentação/checklist e templates da política documental. Atualizar módulos/contratos/arquitetura/operação realmente afetados, registro por entrega, índices/changelog/backlog/matriz. Documentar também parciais/interrompidas. Nenhuma pendência documental obrigatória pode ser ocultada para declarar entrega completa; entregar o parcial possível com limite explícito.

Relatar funcionamento, checks, limites, rollback, documentação e aceite. DONE exige aceite humano da revisão exata; novo diff invalida aceite/evidência pertinente. Merge/deploy/produção/destruição/migrações irreversíveis seguem autorização aplicável já dada na sessão, sem pedir repetidamente. Este arquivo não autoriza por si só contratação/gastos.

## Veracidade e finanças

Decisão: PROPOSTA/ACEITA/SUPERADA. Implementação: PLANEJADO/IMPLEMENTADO, sempre por controle/módulo. Verificação: NÃO VERIFICADO/PASS/FAIL/NOT_RUN/UNSUPPORTED com evidência. Não promover arquitetura inteira por concluir um ticket. Datas reais com fuso explícito (esta migração: Europe/Berlin); SHA/IDs reais ou fixtures rotuladas; revisão de código anterior ao commit documental evita hash circular.

Uso/modelo/cota têm versão/fonte/observed_at; nullable quando ausente. Não converter tokens de assinatura em cobrança API, assumir ilimitado ou inventar reset. Custo fixo/extra/API e total/incremental separados. Registrar capacidade comercial/isolamento hostil como gate próprio; piloto não equivale a produto comercial aprovado.

## Proveniência

- [Política original](../99-historico/originais/sources/POLITICA-IA.md)
- [Extensão v3 preservada](../99-historico/originais/output/le-fabrique-multitenant/documentacoes/POLITICA-IA.md)

## Regras locais preservadas na integração DOC-MV-001

Objetivo: VPS única de controle e execução com assinaturas oficiais; não reintroduzir API de IA como padrão. Stack aprovada React + TS + Vite, NestJS + TS, worker Node/TS separado, PostgreSQL, Redis/BullMQ e Compose; monorepo apps/web, apps/api, apps/worker, packages/contracts e packages/runtime. Contratos compartilhados validam entrada em runtime; segredos/código servidor não entram no painel. API não executa clientes/builds/checks; worker é o executor. Stack do piloto externo é preservada. Não apagar implementação incompatível nem reabrir escolha: registrar divergência e planejar adaptação autorizada.

Ler README, piloto/plano, ticket e capítulos/ADRs afetados. Ticket READY, branch/worktree isolada, objetivo/paths/critério/baseline/provider elegível/limites explícitos. Só um writer; quiescência desconhecida impede outro no mesmo workspace. Até duas correções, depois checkpoint/diagnóstico. Não declarar sucesso por exit code ou fala do modelo. Docs atuais, entrega, índices, changelog/backlog, diff sanitizado, evidências e lessons efetivamente aplicadas precisam acompanhar cada alteração; sem documentação obrigatória pendente. Novos relatos seguem docs/09-entregas, substituindo a antiga obrigação de documentacoes/<dominio>.

Resume nativo só em ambiente/sessão compatível; troca de provider usa estado externo. Nunca exportar auth.json/tokens ao controle. API key herdada é bloqueada em subscription-only. Provider/modelo/versão efetivos vêm da evidência do cliente; desconhecido permanece desconhecido. Conferir help/versão no preflight, sandbox explícito sem permissões amplas por padrão. Não contratar VPS/plano ou habilitar gastos sem autorização aplicável. DONE depende de aceite da revisão exata; sem merge/deploy/destruição por esta política.

Tenancy/RLS e tools via broker permanecem direção posterior, sem software implementado por esta migração. O AS-IS de perfis nativos não prova ponte exclusiva nem atende automaticamente LF-MT. No perfil futuro, CLI sem ponte suportada é incompatível/desabilitado; não montar auth na sandbox ou converter OAuth em API para contornar. Preservar auth fora de código/sandbox, nenhum acesso broker/sandbox ao controle/DB/Redis/daemon. Uma proposta mais recente não revoga silenciosamente decisões aceitas.

## Exceção específica autorizada — JEV

Em 2026-10-09 (Europe/Berlin), o responsável autorizou exceção à restrição de APIs exclusivamente para JEV, com implementação posterior. [ADR-008](../06-decisoes/ADR-008-excecao-jev.md) registra escopo e limites; [plano](../08-desenvolvimento/planos-memoria-e-decisoes/07-jev.md) detalha preparação. Orçamento e limite de chamadas serão definidos pelo responsável via sistema; não presumir zero, ilimitado ou teto ausente como autorização de chamada.

APIs de LLM/extras/autorecharge/fallback pago permanecem desligados. Configuração completa, validação de entradas/candidatos, sanitização, credencial segregada, controle de orçamento e auditoria são condições anteriores a uso real de JEV. A exceção não concede autoridade para remover obrigatórios ou contornar segurança. Indisponibilidade/inconclusão usa LLM elegível por cliente oficial e depois humano. Esta entrega documenta; não integra ou ativa o serviço.
