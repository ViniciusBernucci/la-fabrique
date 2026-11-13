# DOC-MIGRACAO — pacote Manual Vivo

Data: 2026-10-06, America/Sao_Paulo. **Artefatos documentais CRIADOS**; software PLANEJADO/NÃO VERIFICADO. Estado do aceite: pendente; ID local do pacote, não ticket real promovido a DONE. Não há revisão de código da aplicação disponível; hashes das entradas e verificação documental identificam o trabalho.

## Objetivo, problema e solução

Reorganizar documentação dispersa e instruções duplicadas sem perda de informação. Criado pacote docs-as-code com Manual, três níveis C4, módulos/features/contratos/decisões/operação/desenvolvimento e histórico separado. PDF v2.1, fontes sincronizadas e pacote v3 preservados integralmente; 28 capítulos/anexo recuperados.

## Funcionamento e arquivos

docs/README é entrada didática; INDEX lista documentos; políticas em docs/00-governanca são fonte comum; wrappers de raiz exigem leitura explícita; contratos e módulos apontam fontes especializadas. Entregas/lessons têm propósitos distintos. Guia instrui integração em repo real sem sobrescrever regras/código.

## Decisões, dados e API

Stack corrente documentada React/NestJS/worker Node-TS; stack antiga preservada no histórico. DOC-ADR-001 é proposta de governança. Nenhum contrato/API/DB/migration foi implementado; divergência de POST runs e estados ficou explicitamente auditada.

## Diff, origem e verificações

Patch documental é o pacote inteiro criado em output/reorganizacao-documental; nada de sources/ ou pacote anterior foi editado. MANIFESTO-FONTES associa snapshots/hashes a entradas; VALIDACAO registra checks estruturais e preservação realmente executados. Não há SHA de código inventado. Nenhum teste de aplicação, CLI, RLS, firewall ou SEC executado.

## Segurança, risco e rollback

Não foram lidas credenciais nem acessados serviços/produção. Risco principal é integração cega em repo real: mesclar regras, identificar colisões/contratos/código antes de promover estado. Rollback deste pacote é retirar somente o diretório novo; rollback no repo real reverte apenas patch/commit documental após revisar dependentes, preservando mudanças prévias.

## Documentação e lessons

Todos os capítulos estão no [índice](../../02-INDEX.md); [mapa de migração](../../00-governanca/10-MAPA-MIGRACAO.md) registra fontes e destinos. Não foi inventada lesson de implementação: nenhum conceito de software foi aplicado aqui. Preservação por hashes foi aplicada na própria entrega documental e está evidenciada na governança.

## Uso/custos e pendências

Não executados providers do runtime da fábrica, logo uso/versões/planos deles são desconhecidos. Nenhum gasto/contratação autorizado ou realizado. Repo real/VPS/preflight/IDs/rotas/esquemas/gates aguardam integração e validação; [pendências](../../00-governanca/07-PENDENCIAS.md).

## Aceite

Reorganização do material disponível entregue; instalação no repo da aplicação NÃO EXECUTADA. Aceite humano não foi inferido. Nenhum ticket funcional DONE.
