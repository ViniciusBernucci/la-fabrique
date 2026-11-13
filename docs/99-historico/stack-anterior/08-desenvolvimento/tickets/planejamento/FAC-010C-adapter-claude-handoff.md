# FAC-010C — Adapter Claude e handoff seguro

Status: DONE

## Objetivo

Adicionar um segundo adapter subscription-only para Claude Code e um handoff local que transfere somente contexto, snapshot verificavel e feedback sanitizado entre providers, respeitando selecao de conta/modelo do Centro de Configuracoes, quiescencia e limites do RuntimeGuard.

## Escopo

- Generalizar contratos de runtime para `codex` e `claude`, com validacao Zod.
- Adapter Claude Code por CLI oficial, argv fixo, prompt por stdin, JSONL, shell desligado, ferramentas/permissoes conservadoras, timeout, log limit e cancelamento da arvore.
- Bloquear chaves, tokens e modos API/Console/Bedrock/Vertex herdados no processo Claude.
- Resolver rota de funcionario somente para instalacao habilitada, `AVAILABLE`, provider/modelo compativeis e protecoes financeiras desligadas.
- Coordenador de handoff que exige termino do provider anterior, autoriza troca no RuntimeGuard, cria worktree na mesma base, restaura snapshot por hash e inicia o provider de destino sem sessao privada de origem.
- Fixtures sinteticas para sucesso, auth, limite, cancelamento, output invalido, rota inelegivel e handoff adulterado.

## Fora do escopo

- Executar login, prompt ou preflight real no Claude Code deslogado.
- Tornar Claude elegivel sem `auth status`, modelo observado e confirmacao humana do plano/extras.
- Usar Anthropic Console/API, chave, gateway, Bedrock, Vertex, Foundry, fallback pago ou `--dangerously-skip-permissions`.
- Ativar o consumidor real BullMQ antes do perfil confiavel de repositorio/comandos.
- Transferir ID de sessao, token, cookie, cache OAuth, output bruto ou raciocinio privado entre providers.
- Merge, deploy, piloto externo ou migration.

## Criterios de aceite

1. Adapter Claude valida entrada/saida em runtime e usa somente flags fixas compatíveis com a CLI observada `2.1.285`; modelo configurado entra como valor de `--model`, nunca como comando.
2. Ambiente do processo remove credenciais/modos API conhecidos; autenticacao ambigua falha fechado e nao declara `AVAILABLE`.
3. Read-only e workspace-write possuem allowlists distintas; prompts interativos, persistencia de sessao, MCP nao configurado, browser e bypass irrestrito ficam desligados.
4. JSONL e reduzido a eventos/resultados sanitizados; erro, uso e modelo permanecem nullable quando nao comprovados; output bruto nao e devolvido.
5. Timeout, cancelamento e log limit encerram a arvore antes do resultado; somente uma execucao ativa por adapter.
6. Rota usa a instalacao/modelo escolhidos para o funcionario e rejeita conta desabilitada, nao `AVAILABLE`, modelo ausente ou protecao financeira alterada.
7. Handoff exige resultado terminal do processo anterior, snapshot integro na mesma base e autorizacao do RuntimeGuard; restaura em nova worktree e nao usa resume/sessao do provider anterior.
8. Limites de tentativa/troca continuam aplicados; falha de auth/cota produz erro normalizado e estado apto a checkpoint/`WAITING_PROVIDER`, sem fallback pago.
9. Lint, typecheck, testes, build e `git diff --check` passam somente com fixtures.

## Baseline, provider e limites

- Base aceita: `0b1544dba3d8c2fef96fd61ac0ad816d312fd94c` (FAC-010B aceito e registrado).
- Branch/worktree: `feat/fac-010c-claude-handoff`, `/home/vinicius/le-fabrique-fac-010-handoff`.
- Claude Code observado: `2.1.285`; `auth status --json` retorna `loggedIn=false`, `authMethod=none`, `apiProvider=firstParty`.
- A CLI local confirma `auth login --claudeai` como assinatura padrao e `--console` como billing de API proibido.
- Um writer; ate duas tentativas e duas trocas; timeout fornecido pelo job, limitado a 30 minutos; log conforme contrato, maximo 10 MiB.
- Budget API zero; API/extras/creditos/autorecharge/fallback pago desligados.

## Fontes e aceite

Referencias oficiais: CLI reference e setup do Claude Code em `docs.anthropic.com`. Fixtures nao comprovam assinatura real. A entrega permanecera `AWAITING_HUMAN` ate aceite da revisao exata; elegibilidade operacional do Claude permanecera pendente ate configuracao posterior pelo software e confirmacao humana da conta.

Implementacao funcional verificada em `3420e241644fa64d7c525939c3cfebf407518681`. Entrega aceita explicitamente pelo responsavel em 2026-10-01 na revisao documental exata `0748f029a8a62ce891486dc5751c207bdd6e56db`. Claude permanece inelegivel enquanto a conta real estiver deslogada.
