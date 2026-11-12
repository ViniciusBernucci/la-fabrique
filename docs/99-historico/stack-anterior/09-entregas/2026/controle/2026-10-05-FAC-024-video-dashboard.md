# FAC-024 — Vídeo decorativo e área compacta na home
Data: 2026-10-05. Estado IMPLEMENTADO / AWAITING_HUMAN.
Baseline 92627b8; revisão de código `eaf90013e8978a6562f930b98754f28d9a68c5d7`. Branch feat/fac-024-dashboard-video, worktree isolada /home/vinicius/le-fabrique-fac-024.

## Objetivo e critérios
Substituir a imagem central pelo vídeo fornecido, reduzir a área para cerca de 70% do tamanho atual, repetir com transição imperceptível e esconder controles. [Ticket READY](../../../08-desenvolvimento/tickets/controle/tickets/FAC-024.md). Interpretação comunicada: 70% da largura anterior, mantendo a proporção do vídeo; altura acompanha a nova proporção. Desktop 60% → 42% da coluna; mobile calc((100% - 24px) * 0.7). Menus/fontes/cartões existentes preservados.

## Implementação
DashboardLayout apresenta vídeo local `/videos/control-room-loop.mp4`, autoplay, muted, loop, playsInline e preload auto, sem atributo controls. Picture-in-picture e remote playback desabilitados; sem foco de teclado/semântica de conteúdo, pointer-events none e menu contextual bloqueado. O vídeo é decoração; navegação ocorre pelos menus/atalhos existentes. Hotspots invisíveis da antiga ilustração removidos porque suas coordenadas não correspondem à nova cena. Poster local `/images/control-room-poster.jpg` conserva a cena antes de carregar/reproduzir. O PNG original permanece preservado no repositório. Texto de rodapé passa a Prévia demonstrativa; dados continuam simulados.

Área centralizada, aspect ratio 16:9, objeto sem distorção. Ao sair da home o vídeo é desmontado, sem reprodução em outras telas; voltar inicia novamente. Nenhum comando visível de player, contrato/backend/migration ou pacote da aplicação alterado.

## Preparação da mídia e transição
Fonte original fornecida pelo responsável: MP4 H.264/AAC, 1280×720, 24fps, 10,01s, 4758941 bytes. Asset: H.264/yuv420p, 960×540, 24fps, 222 frames/9,25s, sem stream de áudio, 1774867 bytes (~63% menor). faststart permite começar sem baixar o arquivo inteiro; keyframe a cada segundo. Poster JPEG extraído do primeiro frame preparado.

Clipe começa no frame 18 (0,75s). Últimos 18 frames se misturam com os primeiros 18; xfade tem duração 17/24s entre timestamps, incluindo ambos extremos. O último frame corresponde ao frame anterior ao início, evitando corte/resíduo da cauda ao reiniciar. Transição incorporada no MP4, sem sobrepor players/timers no React. [Script de preparação](../evidencias/controle/evidencias/FAC-024/prepare-video.py), [análise reproduzível](../evidencias/controle/evidencias/FAC-024/analyze-media.py), [hashes/medidas](../evidencias/controle/evidencias/FAC-024/media-analysis.json).

FFmpeg 7.0.2-static extraído temporariamente do wheel pinned imageio-ffmpeg 0.6.0 em /tmp; ferramenta não instalada no aplicativo/sistema. Primeiro filtro precisou fps/timebase após trim para xfade aceitar CFR. Primeira comparação apontou resíduo de fade na última amostra; duração ajustada ao intervalo de 17 frames. Uma rodada de ajuste da mídia; nenhuma mudança adicional de aplicação para contornar teste.

## Verificação real
Typecheck web, build web, lint raiz e 46 testes web/18 arquivos PASS. Lint preserva o único aviso preexistente useOptionalChain da API. git diff --check PASS. Browser Chromium/Playwright mede reprodução real do MP4 local, flags de player e dois reinícios nativos; registra ausência de frames pretos e intervalos de renderização, não fixture de vídeo. Seis larguras 1536/1280/1024/768/390/320px sem overflow; proporções desktop/mobile verificadas. Menu/template/login preservados; zero API na entrada da home; erro/pageerror ausente. Capturas desktop/mobile inspecionadas.

Diferença média RGB da fronteira final→início em thumbnails 64×36: 0.7808 níveis/255, abaixo do percentil 95 de frames vizinhos (1.1367). Esse indicador comprova ausência de salto grande de imagem no asset; combinado com apresentação real em dois loops no navegador. Não é promessa de desempenho idêntico em todo dispositivo.

[Resultados do navegador](../evidencias/controle/evidencias/FAC-024/browser-results.txt), [harness](../evidencias/controle/evidencias/FAC-024/browser-check.mjs), [desktop](../evidencias/controle/evidencias/FAC-024/desktop.png), [mobile](../evidencias/controle/evidencias/FAC-024/mobile.png), [testes](../evidencias/controle/evidencias/FAC-024/tests.txt), [typecheck](../evidencias/controle/evidencias/FAC-024/typecheck.txt), [build](../evidencias/controle/evidencias/FAC-024/build.txt), [lint](../evidencias/controle/evidencias/FAC-024/lint.txt), [metadados do asset](../evidencias/controle/evidencias/FAC-024/media-info.txt).

## Limites e rollback
Autoplay mudo é suportado no navegador verificado; políticas de economia de bateria/autoplay do dispositivo podem impedir movimento ou gerar atraso. Poster mantém a decoração durante carregamento. Continuidade é preparada no arquivo e verificada localmente, sujeita ao decoder/carga do navegador. Sem controles visíveis conforme pedido. O painel continua demonstrativo, não representa atividade real dos agentes.
Rollback: reverter `eaf90013e8978a6562f930b98754f28d9a68c5d7`, restaurando PNG/CSS/hotspots anteriores; sem dados ou migration a desfazer. Fonte original preservada; não executar rollback destrutivo. Sem push/deploy/provider.

## Docs, uso de IA e aceite
README raiz/controle, INDEX, CHANGELOG, BACKLOG/ticket atualizados e [lesson sobre vídeo decorativo](../../../10-lessons/14-video-decorativo-loop.md). Stack aprovada preservada. Codex da sessão, mesmo writer, zero subagentes/handoff/clientes adicionais; modelo/cota/custo não comprovados. Merge developer/limpeza pós-merge autorizados anteriormente na sessão; registro de execução após realizar. AWAITING_HUMAN para revisão visual exata, DONE somente após aceite.

## Integração e limpeza realizadas
Fast-forward local em developer até 3ad0357, sem conflitos. Build web no destino PASS, mesmos bundles; reprodução real no painel habitual/5173 PASS: dois reinícios, fronteiras de 66,7/33,3ms, sem frame preto, mesma validação de navegação e seis larguras. [Resultado developer](../evidencias/controle/evidencias/FAC-024/browser-results-developer.txt). Capturas atualizadas pela execução na árvore principal.
Worktree limpa, commit ancestral de developer confirmado; preview próprio PID 2397245 encerrado após conferir cwd. git worktree remove /home/vinicius/le-fabrique-fac-024 executado sem force. Diretório ausente, dependências principais preservadas; git worktree list contém somente developer. Branch feature preservada; sem push/deploy. Aceite visual permanece pendente.
