# Entrega FAC-011 — Centro de configuracoes

Data: 2026-10-01

Status: DONE

## Objetivo e revisoes

Transformar contas de IA, modelos, atribuicoes por funcionario e GitHub em configuracao administravel antes do segundo adapter, sem solicitar credenciais antecipadamente.

- Base do trabalho: `0186bcf5420d941b50f38e4a77f119aaa8faa678`.
- Ticket READY: `b37b020ea4f2e634e3eeb3c8434fafae5953a318`.
- Implementacao inicial: `b119a4e56f2518fc292f916737ababb01bd0ad00`.
- Bootstrap atomico: `2737ab89d7bbf91cb3377df03080370e6d470b6c`.
- Revisao funcional corrigida e verificada: `9a948d968c072b8b5a5841d8eff4d8c8c5e343e6`.
- Branch/worktree: `feat/fac-011-settings-center`, `/home/vinicius/le-fabrique-fac-011`.

## Implementacao

- `packages/contracts`: schemas Zod estritos de instalacoes, catalogos de modelos, funcoes, atribuicoes, GitHub, protecoes financeiras e update versionado.
- `apps/api`: modulo `settings`, defaults conservadores, singleton PostgreSQL, `GET/PUT /api/settings` sob `AdminAuthGuard` e concorrencia otimista.
- Prisma: migration aditiva `20261001000100_fac_011_factory_settings`; nenhuma migration foi aplicada a ambiente real nesta entrega.
- `apps/web`: navegacao Operacao/Configuracoes, resumo, cadastro de multiplas contas, selecao de provider/modelos, atribuicao por funcionario, limites e metadados GitHub.
- Seguranca: nenhum campo de segredo; `SUBSCRIPTION_CLI` e `GH_CLI`; merge/API/extras/creditos/autorecharge/fallback fixados desligados.

## Criterios e evidencias

1. Contratos rejeitam campos desconhecidos como `token`, modelos fora do catalogo e relacoes inconsistentes.
2. O servico cria defaults desabilitados, `AUTH_REQUIRED`, sem modelos e atualiza somente com a versao esperada.
3. O painel distingue cliente habilitado de cliente comprovadamente `AVAILABLE` e permite configurar cada funcao separadamente.
4. Remover uma conta limpa suas atribuicoes no draft antes de salvar.
5. Desabilitar conta ou remover modelo limpa atribuicoes invalidas; modelos duplicados sao normalizados na UI e rejeitados pelo contrato.
6. Estados `AVAILABLE`/`CONNECTED` e demais observacoes nao podem ser forjados por `PUT /api/settings`; pertencem ao worker futuro.
7. GitHub persiste somente metadados; autenticacao e segredo nao atravessam o novo contrato.
8. Politicas financeiras proibidas sao literais `false` no schema e aparecem como somente leitura.

## Checks reais

Executados no worktree com Node/npm do repositorio e dependencias do `package-lock.json`:

- `npm ci`: 211 pacotes instalados, 0 vulnerabilidades reportadas.
- `npm run lint`: passou, 85 arquivos verificados.
- `npm run typecheck`: passou em contracts, runtime, API, worker e web.
- `npm test`: passou, 70 testes em 19 arquivos.
- `npm run build`: passou; web gerou bundle Vite de producao.
- `DATABASE_URL=postgresql://fixture:fixture@127.0.0.1:5432/fixture npm exec -w @le-fabrique/api -- prisma validate --schema prisma/schema.prisma`: schema valido, sem conexao ao banco.
- `git diff --check`: passou.

A primeira tentativa de `db:generate` encontrou o worktree sem dependencias; `npm ci` resolveu a pre-condicao. A primeira validacao Prisma sem `DATABASE_URL` parou antes de validar; a repeticao usou URL sintetica somente para parsing. Nao houve deploy, migration real, chamada de provider, login, GitHub remoto ou consumo de cota.

## Limites e riscos

- A UI teve build e view model testados, mas nao houve ensaio visual em browser/E2E nesta revisao.
- Estado, versao e modelos do provider ainda precisam vir de preflight confiavel; o cadastro nao torna uma conta elegivel.
- O endpoint usa o token administrativo atual em `sessionStorage`; autenticacao multiusuario continua futura.
- O fluxo de login, armazenamento oficial de credencial, descoberta e troca de conta nao esta implementado.
- A intencao de criar PR e configuravel, mas nenhuma operacao GitHub foi implementada; merge permanece bloqueado.

## Referencia Orca

Foram reaproveitados conceitos publicos de organizacao por secoes, ativacao de agentes, contas separadas e integracoes. Nenhum codigo ou asset foi copiado. A politica de permissoes amplas do Orca nao foi adotada porque conflita com os limites da La fabrique.

## Rollback e aceite

Antes de deploy, o rollback funcional e reverter, nesta ordem, `9a948d968c072b8b5a5841d8eff4d8c8c5e343e6`, `2737ab89d7bbf91cb3377df03080370e6d470b6c` e `b119a4e56f2518fc292f916737ababb01bd0ad00`; a migration ainda nao foi aplicada. Se ela ja tiver sido aplicada em outro ambiente, preservar a tabela/dados e fazer migration compensatoria; nao apagar configuracoes manualmente. O responsavel aceitou a revisao documental exata `e41ec1e45273aba3205f266e8753dfca305c5952` em 2026-10-01.
