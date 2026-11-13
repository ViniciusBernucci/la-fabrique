# FAC-012Y — Handoff automático configurado

Data: 2026-10-03. Status: IMPLEMENTADO / AWAITING_HUMAN. Baseline `af5545a`, READY `c908b1d`; código `f66a9238dfad143600cce823d550be7859f450fd`. Branch `feat/fac-012y-configured-handoff`, worktree `/home/vinicius/le-fabrique-fac-012y`.

## Objetivo, critérios e funcionamento

Centro de Configurações oferece zero a duas alternativas de instalação/modelo por função, ordenadas e vazias por padrão. Selecionar e salvar autoriza somente alternativas de assinatura; não altera proteções financeiras. Contrato runtime rejeita primário ausente, contas duplicadas, IDs desconhecidos/desabilitados, modelos fora do catálogo e mais de duas alternativas. UI remove referências inválidas ao excluir/desabilitar instalação ou editar catálogo/atribuição. Campos novos opcionais mantêm compatibilidade de configurações/resultados anteriores sem mudar digests por defaults artificiais.

Router consulta configuração atual e escolhe somente candidatos explícitos elegíveis AVAILABLE com adapter suportado. Conta primária indisponível pode ser ignorada antes da primeira chamada; não há processo anterior nesse caso. Segurança da factory, ambiente e permissão são gates independentes: erro de preparação não autoriza tentar outra conta. Reviewer continua READ_ONLY. Não há conta/modelo fixos, migração ou fallback pago.

Quando uma chamada terminal conhecida informa AUTH_REQUIRED/RATE_LIMITED/PROVIDER_BUSY, workflow exige quiescência e checks parados, captura snapshot posterior à parada na base exata, cria outra worktree limpa, restaura com verificação de hashes e reconstrói contexto. Revalida rota atual após restauração e mantém o runtime selecionado para a próxima chamada, evitando registrar uma conta e executar outra. IDs de instalações indisponíveis são excluídos nesta tentativa, não reclassificados globalmente por suposição.

Developer continua sem consumir rodada de correção por indisponibilidade. Reviewer troca dentro da revisão, sem reexecutar Developer nem checks de código byte-equivalente restaurado. Máximo dois handoffs no workflow; chamadas de cada função, orçamento global de chamadas/tempo/trocas e AbortSignal não são reiniciados. Limite pode impedir a chamada do destino depois de restore: registro de handoff prova preparação/restauração, não chamada bem-sucedida; observações de runtime mostram chamadas efetivas. Sem alternativa/limite de função, espera WAITING_PROVIDER como X.

Setup/restauração que falha após snapshot conhecido retorna FAILED com evidência da origem, sem chamar destino. Captura inválida/unknown continua propagando falha; não inventa parada nem libera writer. Transferência é patch/untracked/contexto externo, nunca sessão privada/tokens/raciocínio. Mesmo worker conserva lease/fence e exclusão global durante etapas sequenciais. Novo job/retomada humana continua exigindo stop e fence novo; trocar cliente dentro da tentativa não entrega autoridade a outro writer.

Resultado público/journal carregam metadados mínimos role, instalação origem/destino, execução origem, motivo, snapshot/hash e horário. Painel mostra histórico escapado; campos opcionais são preservados pelo contrato/result report existente, sem novas tabelas/endpoints.

## Diff e checks

Diff sanitizado: `git show f66a9238dfad143600cce823d550be7859f450fd -- apps packages/contracts/src/index.ts`. 12 arquivos, 751 inserções/98 remoções. Fixtures sintéticas sem credenciais. Implementação/fixtures somente nos caminhos autorizados; evidências/documentação separadas.

- `npm ci --ignore-scripts`: dependências privadas; audit zero vulnerabilidades.
- `npm run db:generate`: geração local, sem banco/migration.
- `npm test`: 411 passaram (launcher 2, contracts 34, runtime 46, API 139, worker 172, web 18).
- `npm run typecheck`: todos os workspaces passaram, incluindo fixtures finais.
- `npm run lint`: 198 arquivos, sem fixes.
- `npm run build`: os cinco workspaces passaram; teste extra de budget acrescentado depois, validado na suíte/typecheck finais, implementação inalterada.
- `git diff --check`: passou.

Suíte inicial 410, final 411 com teste extra de budget. Nenhuma falha de checks/regressão observada; patches auxiliares corrigiram contexto de heading documental antes da validação final, sem evidência falsa. Typecheck inicial precedeu integração final e foi repetido na revisão exata.

Novos testes verificam Developer/Reviewer em outra worktree após restore, ordem de chamadas/contexto, falha de integridade sem novo agente, orçamento global não reiniciado, ausência de fallback implícito, exclusão de conta já indisponível, cinco configurações inválidas, UI vazia por padrão/pruning e histórico público. Não são prova de autenticação/cota/confinamento real.

## Limitações, rollback e aceite

Consumer permanece desligado; não houve piloto/login/IA/provider/API/extras/créditos/autorecharge/serviços/filas/banco/push/deploy. Claude escrita ainda bloqueada pela factory até prova de confinamento granular; selecioná-lo como alternativa de escrita não contorna o bloqueio. Gate da documentação técnica do projeto externo ainda pendente. Autenticação/preflight, capacidade/backup e implantação são manuais. Configuração pode mudar entre etapas; cada rota reconfirma snapshot atual, não promete revogação instantânea de processo já iniciado.

Snapshots de handoff ficam locais durante a tentativa e o protocolo existente entrega o resultado/bundle final. Crash anterior à publicação continua UNKNOWN/BLOCKED_RECOVERY, sem promessa de retomada automática do snapshot intermediário. Não há limpeza automática de workspaces/evidências de execução.

Rollback somente com writer comprovadamente parado e ticket autorizado: reverter código compatível preservando settings/resultados/journal/snapshots; versões anteriores strict podem rejeitar campos novos, portanto não apagar dados para fazê-las aceitar. Manter reader/consumer compatível para recuperação. Aceite exato humano pendente, não DONE.

## Documentação, lessons e IA

READMEs runtime/handoff/controle/operação/planejamento, ADR-003, índice/changelog/backlog, README raiz, CONTROLE-MVP e lesson lifecycle atualizados. Distinção concreta entre troca de cliente sob lease contínuo e transferência de writer com novo fence documentada. Nenhuma chamada de IA da fábrica; custo/tokens/modelo efetivo do assistente não observáveis.
