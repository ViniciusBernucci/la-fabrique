> Leitura: [← Anterior](02-POLITICA-DOCUMENTACAO.md) · [Índice didático](../02-INDEX.md) · [Próximo →](04-CHECKLIST-ENTREGA.md)

# Integração do Manual Vivo neste repositório

Migração inicial executada sobre a2cc5e0 na branch docs/manual-vivo-inicial; a worktree temporária foi removida após integração. O caminho atual é /home/vinicius/le-fabrique/docs. A migração 607d6ed foi integrada em developer. O pacote original está preservado em docs/99-historico/pacote-recebido; a pasta redundante reorganizacao-documental foi removida após comparação íntegra dos 230 arquivos. Não foi copiado por cima do backlog/código/configuração real.

[Manual](../01-README.md), [políticas](00-README.md), [mapa](10-MAPA-MIGRACAO.md), [auditoria](08-AUDITORIA.md), [entrega](../09-entregas/2026/2026-10-07-DOC-MV-001-migracao-manual-vivo.md). Originais/hash/documentos/seções estão em governança e docs/99-historico. Atalhos essenciais antigos foram mantidos após conteúdo canônico completo; bridges individuais redundantes foram removidos pelo pedido posterior do usuário.

## Revisar e adotar

1. Abrir guias e as duas políticas explicitamente. Conferir AS-IS/alvo, snapshots/destinos/seções, contratos/controllers/ADRs e pendências.
2. Rodar python3 scripts/validar-documentacao.py; para comparar fontes intactas usar --reference-root /home/vinicius/le-fabrique. Após integrar paths de origem podem conter bridges; aí validar snapshots, não comparar source realocado contra o original.
3. Revisar commit/diff e evidências finais. Aceite humano da revisão exata precede DONE documental. O merge inicial para developer já foi autorizado e executado; push, deploy e ativação seguem autorização aplicável.
4. Novos tickets usam prompts/PROMPT-INICIAL.md e prompts/PROMPT-ENTREGA.md. prompts/implementacao LF-MT permanecem planejamento futuro; não executar pelo ato de migrar docs.

## Carregamento e rollback

Autoload de nova sessão Codex/Claude/Antigravity NÃO VERIFICADO; versões só --version observadas. Fallback é abrir guias/políticas e registrar fontes/hashes, sem ampliar permissões/autenticar/ler auth para testar. Fixtures têm regras locais preservadas.

Rollback revisa dependentes e reverte só commit/patch documental; snapshot permite restaurar paths alterados sem reset destrutivo ou overwrite de trabalho anterior. Nenhum DB/serviço/credencial foi migrado. [Guia original do pacote](../99-historico/pacote-recebido/GUIA-DE-INTEGRACAO.md) permanece íntegro e histórico.
