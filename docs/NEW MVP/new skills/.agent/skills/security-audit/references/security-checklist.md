# Lista de verificação de segurança

Use-o como um registro de cobertura, e não como um substituto para rastrear a situação real do projeto.
fluxos de dados e autoridade. Marque cada item aplicável, não aplicável com razão,
revisado com evidências ou descobertas.

## 1. Escopo e arquitetura

- Fixe o commit/intervalo auditado e o modo de implantação.
- Identifique público, autenticado, administrador, interno, local, plug-in, CI e
  limites de confiança de dependência.
- Identifique dados confidenciais, credenciais, capacidade de execução, integridade,
  disponibilidade e ativos de cobrança.
- Desenhe entrada -> analisar -> validar -> autorizar -> efeito colateral -> persistir ->
  caminhos de saída/log.
- Identifique trabalhos em segundo plano e pontos de entrada alternativos que ignoram o primário
  caminho da solicitação.

## 2. Autenticação

- Use bibliotecas/protocolos de autenticação estabelecidos.
- Valide a assinatura/MAC antes de confiar nas declarações.
- Valide emissor, público, assunto, vencimento, não antes e algoritmo.
- Evite confusão de algoritmos, tokens não assinados, travessia de ID de chave/SSRF, repetição,
  fixação, downgrade e fallback inseguro.
- Use CSPRNG para tokens e entropia suficiente.
- Armazene tokens/senhas com segurança; nunca registre-os.
- Suporta expiração, rotação, revogação, logout e comprometimento de credenciais.
- Compare segredos fixos em tempo constante quando relevante.
- Vincule sessões/retornos de chamada/fluxos de dispositivos ao principal inicial.

## 3. Autorização e multilocação

- Negar por padrão em um limite centralizado de capacidade/política.
- Autentique e autorize cada rota, método, mensagem, trabalho em segundo plano e
  ponto de entrada de protocolo alternativo.
- Verifique a propriedade do objeto e o escopo do locatário/espaço de trabalho/projeto em cada leitura,
  pesquisa, mutação, exportação, backup, restauração e operação destrutiva.
- Falha no fechamento por identidade ausente, parcial, ambígua, obsoleta ou conflitante.
- Mantenha as chaves de cache/índice/arquivo/armazenamento de objetos com escopo total.
- Prevenir IDOR/BOLA, atribuição em massa, função/autopromoção, deputado confuso,
  reutilização de capacidade e corridas de verificação/uso.
- Verifique a separação administrador/raiz para caminhos UI e API/auxiliar.
- Garanta contagens, erros de existência, tempo, registros, classificação de pesquisa, incorporações e
  os resumos gerados não vazam os dados de outro locatário.

## 4. Entrada e injeção

- Normalize uma vez em um limite digitado; validar após decodificação/canonização.
- Comprimento, profundidade, contagem, intervalos numéricos, recursão e codificação vinculados.
- Use consultas parametrizadas e construtores estruturados.
- Passe o subprocesso argv separadamente; evite a avaliação do shell.
- Escape para o contexto de saída exato (HTML, JS, URL, cabeçalho, CSV/fórmula, log,
  terminal, modelo).
- Restringir caminhos de arquivos em raízes canônicas; lidar com links simbólicos e janelas de corrida.
- Valide as entradas do arquivo antes da extração; índices e totais de expansão do limite.
- Restringir esquemas de URL, hosts, portas, redirecionamentos, resolução de DNS/IP e privacidade
  alvos de rede para buscas no lado do servidor.
- Evite desserialização insegura de objetos/nativos e ativação de tipo polimórfico.
- Trate prompts/texto recuperado/saída de ferramenta como dados; impor ferramenta/capacidade
  política fora do modelo e higienizar o contexto de saída.
- Impedir CRLF/cabeçalho, cabeçalho de host, redirecionamento aberto, regex, XPath/LDAP e consulta
  Injeção DSL quando esses coletores existem.

## 5. Sistema de arquivos, processos e plugins

- Use usuários com privilégios mínimos e permissões de criação restritivas.
- Evite caminhos temporários previsíveis; abra atomicamente e resista às corridas de link simbólico/hardlink.
- Preserve a propriedade canônica e o escopo em leitura/gravação/exclusão/mover/restauração.
- Valide os caminhos executáveis ​​e a origem do plugin.
- Limpe o ambiente herdado, cwd, descritores de arquivo e credenciais.
- Limite o tempo/saída/recursos do subprocesso e lide com cancelamento/limpeza.
- Não conceda aos plugins/hooks mais recursos do que o necessário.
- Trate compiladores, macros, scripts de construção, ganchos de ciclo de vida de pacotes, descoberta de testes,
  e migrações como execução de código.
- Verifique canais de atualização, somas de verificação/assinaturas, reversão e comportamento de downgrade.

## 6. Rede e web

- Vincule-se à interface menos exposta por padrão.
- Exija autenticação antes de analisar corpos de solicitação caros/confidenciais.
- Aplicar limites de corpo/cabeçalho/tempo/simultaneidade/taxa.
- Valide cabeçalhos de host/proxy e configuração de proxy confiável.
- Configure o CORS de forma restrita; use defesas CSRF para mutação autenticada por cookie.
- Defina sinalizadores de cookies seguros e cabeçalhos de segurança do navegador, quando aplicável.
- Verifique assinaturas de webhook em bytes brutos; evitar a repetição.
- Não vaze rastreamentos de pilha, segredos, caminhos, existência de locatário ou URLs internos.
- Validar certificados/nomes de host TLS; tornar explícito o uso remoto de texto simples.
- Evite o contrabando/dessincronização de solicitações nos limites de proxy/backend, quando aplicável.

## 7. Persistência, consistência e ciclo de vida

- Valide antes da mutação e autorize dentro da transação/corrida correta
  limite.
- Confirme índices derivados com dados canônicos ou exponha um estado de recuperação confiável.
- Use gravações de arquivos atômicos e padrões duráveis ​​de renomeação/fsync quando necessário.
- Tenha um modelo de propriedade claro para redatores simultâneos.
- Tornar as operações destrutivas explícitas, com escopo definido, confirmadas, auditadas e
  recuperável.
- Verifique os caminhos de encaminhamento e reversão/recuperação de migrações em dados representativos.
- Valide backups/restaurações, caminhos de arquivo, versões, propriedade e integridade.
- Preservar a atribuição de auditoria; não deixe que os usuários forjem outro ator.
- Defina o comportamento de retenção/exclusão para dados canônicos, caches, logs, exportações,
  backups, incorporações e contexto do modelo.

## 8. Segredos e privacidade

- Mantenha segredos reais fora da fonte, histórico, acessórios, documentos, imagens, artefatos,
  logs, erros, métricas, telemetria, linhas de comando e listagens de processos.
- Centralize a resolução de segredo/configuração e distinga valores ausentes de vazios.
- Minimize a coleta de dados e o contexto do modelo/provedor.
- Sanitize antes da persistência/transporte/registro, não depois.
- Edite campos aninhados e estruturados, não apenas padrões de strings óbvios.
- Proteja exportações/backups e evite destinos padrão inseguros.
- Tornar visível a divulgação de fornecedores transfronteiriços/externos quando relevante.
- Teste a exclusão/retenção e a separação de usuário/inquilino.

## 9. Criptografia

- Use primitivos e protocolos padrão mantidos.
- Use CSPRNG e nonces/IVs exclusivos conforme necessário.
- Separe as chaves por finalidade e gire com segurança.
- Autenticar texto cifrado e metadados; não invente formatos de criptografia.
- Verifique as assinaturas antes de usar e fixe a raiz/proveniência de confiança pretendida.
- Evite hashes, cifras, modos, desvios de certificados e operações silenciosas inseguras.
  rebaixamento/substituição.

## 10. Disponibilidade e abuso

- Tamanhos de solicitação de limite, contagens de coleção, profundidade do gráfico, trabalho de regex, descompactação,
  saída gerada, cardinalidade de log e crescimento do disco.
- Filas, tarefas, threads, conexões, pools vinculados e trabalho por usuário/global.
- Aplique tempos limite, cancelamento, limites de repetição, espera exponencial e jitter.
- Evite tentar novamente efeitos colaterais não idempotentes sem desduplicação.
- Evite a privação/impasse de bloqueio e transações longas em caminhos ativos.
- Autenticação, pesquisa, geração, upload, exportação e webhook caros com limite de taxa
  caminhos conforme apropriado.
- Faça com que as dependências degradadas falhem de forma previsível, sem exaustão em cascata.

## 11. Cadeia de suprimentos, CI e liberação

- Explique cada adição de dependência e alteração de registro/fonte.
- Revise arquivos de bloqueio, somas de verificação, revisões git, dependências de caminho, recursos, scripts de construção,
  ganchos do ciclo de vida do pacote, código do fornecedor e licenças.
- Investigar typosquatting, mudanças de mantenedor/proveniência, obsoletos/não mantidos
  pacotes e avisos.
- Mantenha as permissões de CI mínimas e fixe ações de terceiros de acordo com o projeto
  política.
- Nunca execute código PR não confiável com segredos de repositório ou grave tokens.
- Auditoria `pull_request_target`, matrizes dinâmicas, interpolação de shell, artefato
  nomes/caminhos, caches e entradas de fluxo de trabalho reutilizáveis.
- Separar construção da publicação; publique apenas commits testados e marcados com precisão.
- Verifique a consistência da versão/tag, somas de verificação/assinaturas/atestados, artefato
  origem, destino do registro e reversão.
- Proteja credenciais de implantação/liberação de logs e códigos controlados por contribuidores.

## 12. Indicadores de códigos maliciosos

Estas são pistas, não provas. Determine a intenção e a acessibilidade.

- Novos destinos de rede, DNS, telemetria, webhooks, túneis ou uploads.
- Descoberta de metadados de credencial/home/navegador/nuvem.
- Carregamento shell/eval/dinâmico ou execução baixada.
- Blobs base64/hex/comprimidos/criptografados ou caminhos de decodificação e execução em estágios.
- Controles bidi Unicode, identificadores homoglifos, condições invisíveis, minificados
  código, arquivos gerados sem fonte, binários inexplicáveis.
- Comportamento acionado por tempo/usuário/host/CI/região e ativação atrasada.
- Contas ocultas, chaves/tokens estáticos, desvios de depuração/administração, fallback permissivo.
- Persistência através de arquivos de inicialização, cron/services, ganchos de pacotes, plugins,
  fluxos de trabalho, imagens ou canais de atualização.
- Desativar ferramentas de segurança, testes, registro, auditoria, TLS, verificações de assinatura, autenticação,
  sandbox ou limites de recursos.
- Testes/simulações que ocultam efeitos colaterais reais ou afirmam comportamento vazio.
- Limpeza destrutiva fora das raízes do projeto ou comportamento anti-análise.

## 13. Qualidade das evidências

- Distinguir a acessibilidade do código dos acessos grep.
- Declare os pré-requisitos do invasor e o controle sobre cada entrada.
- Prove o coletor sensível e o controle ausente/com falha.
- Incluir um caso de controle legítimo nos testes.
- Verifique novamente todos os chamadores alternativos e caminhos de falha/substituição.
- Verifique as correções no commit final exato e no formato de implantação.
- Declare o risco residual em vez de converter incógnitas em uma passagem.
