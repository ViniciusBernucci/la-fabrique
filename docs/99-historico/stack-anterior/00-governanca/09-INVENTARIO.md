> Leitura: [← Anterior](08-AUDITORIA.md) · [Índice didático](../02-INDEX.md) · [Próximo →](10-MAPA-MIGRACAO.md)

# Inventário completo da integração

[INVENTARIO-REPO.json](INVENTARIO-REPO.json) contém cada path/tipo/assunto/status/bytes/SHA-256/seções/destino. [DUPLICACOES.json](DUPLICACOES.json) agrupa identidade byte a byte; igualdade de título não prova igualdade de conteúdo. [Mapa por seção](MAPA-SECOES-REPO.json) contém hash de cada trecho e tratamento; [manifesto](MANIFESTO-FONTES.json) preserva materiais realocados.

Scope: git ls-files da revisão a2cc5e0 e todos os arquivos do pacote fornecido, inclusive ocultos, PDF/ZIP e fontes históricas. Excluídos .env/credenciais, dependências, .git e outputs gerados. Fontes sources históricas/sincronizadas somente lidas, sem mutação. Inventário de código/manifests/migrations/tests não prova execução. Links de fontes físicas usam os arquivos reais; nenhum src/modules suposto.
