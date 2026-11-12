> Leitura: [← Anterior](09-INVENTARIO.md) · [Índice didático](../02-INDEX.md) · [Próximo →](11-VALIDACAO.md)

# Mapa da migração real

Fonte repo a2cc5e0 + pacote recebido em reorganizacao-documental. Snapshot antes de transformar; cada documento/seção possui hash/origem/destino/tratamento. Os JSON são fonte detalhada; este capítulo orienta navegação, sem copiar os registros.

- [Mapa por seção](MAPA-SECOES-REPO.json): cobertura completa dos Markdown, inclusive preâmbulos e trechos sem heading.
- [Destinos locais](DESTINOS-REPO.json): cada path antigo→canônico; bridges só depois de conteúdo copiado.
- [Inventário](INVENTARIO-REPO.json): docs, regras ocultas, PDF/ZIP, schemas/API/migrations/config/tests/CI disponível e artefatos; hashes e duplicações.
- [Manifesto de preservação](MANIFESTO-FONTES.json): snapshots repo/pacote, com SHA-256 e bytes.
- [Mapa original do pacote/PDF por página](../99-historico/pacote-recebido/docs/00-governanca/MAPA-MIGRACAO.md): mapeamento histórico íntegro; não se afirma que seu espelho era repo executável.

## Tratamento por classe

| Origem | Destino / tratamento |
|---|---|
| README/setup/status local | Manual, ambiente-local e controle-mvp com errata; README original no snapshot |
| Políticas/guias/regra | docs/00-governanca; wrappers preservam especificidades e exigem leitura explícita |
| READMEs por domínio | Módulos/operação atuais + referências locais integrais com nota de revisão |
| Relatos datados FAC/OPS/PLAN | docs/09-entregas/<ano>/<domínio>; links editoriais, conteúdo/IDs/revisões intactos |
| Evidências binárias/patch/log/browser scripts | docs/09-entregas/2026/evidencias/<domínio>; bytes íntegros, originais preservados |
| Tickets locais | docs/08-desenvolvimento/tickets/<domínio>; não executados por esta tarefa |
| Lessons aplicadas | docs/10-lessons; exemplos/revisões íntegros, índice não declara ausência fictícia |
| ADRs locais | docs/06-decisoes; status/decisão original intactos, colisão ADR-003 do pacote identificada |
| Backlog/changelog reais | Raiz preservada integralmente, links editoriais + DOC-MV-001; pacote não os sobrescreve |
| ESPEC/plano/piloto/matriz/fontes | Capítulos canônicos com conteúdo local integral e atalhos raiz |
| C4/plano/contratos genéricos do pacote | AS-IS substitui declarações de repo ausente; alvo/propostas preservados em capítulos separados ou snapshots com justificativa |
| PDF 45 páginas, capítulos recuperados e ZIP | Histórico íntegro + mapeamento página/origem; extrair texto não prova layout |
| Código/config/schemas/tests não realocados | Inventário/hash e links como evidência; sem mudança de comportamento |
| Exemplos UI de gate | Somente paths Markdown novos; política configurável sem novo enforcement |

## Conciliação por seção

Sections de documentos locais realocados são transferidas integralmente, com links rebaseados; notas editoriais delimitam frases de estado superadas. Regras vigentes de stack/topologia/assinatura/writer/lease/fence/gates entram em políticas/módulos/contratos atuais. Seção de proposta substituída não é autoridade corrente: fica íntegra no pacote histórico e no alvo quando há direção útil distinta. Nenhum snapshot é “corrigido”.

## Limpeza autorizada posterior

[MD removidos](MD-REMOVIDOS.json) registra cada bridge individual redundante apagado pelo pedido explícito do usuário, com destino canônico e snapshot/hash previamente conferidos. Atalhos de raiz e entradas documentacoes/INDEX, política/domínios README e lessons/INDEX permanecem essenciais. Não existe exclusão de conteúdo útil ou snapshot para apagar conflito.

## DOC-MV-002 — caminhos atuais após organização da raiz

[Realocação da raiz](REALOCACAO-RAIZ.json) e [retirada de documentacoes](REMOCAO-DOCUMENTACOES.json) suplementam a migração inicial, com snapshots/hash e destino por arquivo. Backlog atual em docs/08-desenvolvimento/09-backlog.md; changelog em docs/04-CHANGELOG.md; guia em docs/00-governanca/GUIA-DE-INTEGRACAO.md. Documentacoes não existe mais como pasta corrente; seu nome em origem/histórico identifica proveniência.
