> Leitura: [← Anterior](13-modal-edicao-configuracao.md) · [Índice didático](../02-INDEX.md) · [Próximo assunto →](../00-governanca/00-README.md)

# Vídeo decorativo com fronteira contínua
No FAC-024, só adicionar loop repetiria um corte do asset fornecido. prepare-video.py desloca o início em 18 frames e mistura a cauda com a cabeça; a duração do fade usa 17 intervalos entre 18 frames, incluindo os extremos. Análise de frames detectou o resíduo de uma duração arredondada de 0,75s e confirmou fronteira mais suave após ajuste. Hashes associam a prova ao MP4 exato.

DashboardLayout usa autoplay/muted/playsInline, poster local e sem controls, interação ou foco no vídeo. Preparar transição no MP4 evita temporizadores/decoders duplos na página; faststart, áudio removido e resolução adequada reduziram tráfego. Browser verifica reprodução e dois reinícios com requestVideoFrameCallback; diferença entre último/primeiro frame não prova tempo de apresentação, e o tempo no browser não garante desempenho em todos os dispositivos. [Relatório e scripts](../09-entregas/2026/controle/2026-10-05-FAC-024-video-dashboard.md).

FAC-025: verificação técnica do loop não substitui aceite visual do responsável. Após rejeição, restaurada imagem estática e removidos assets do public para que não entrem no build, preservando fonte/evidências históricas e tamanho aprovado anteriormente. Browser confirma ausência de requests de mídia, além de ausência do player. [Retirada e revisão](../09-entregas/2026/controle/2026-10-06-FAC-025-retirar-video-dashboard.md).
