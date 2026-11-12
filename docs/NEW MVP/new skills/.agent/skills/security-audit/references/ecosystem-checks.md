# Verificações de segurança específicas do ecossistema

Carregue apenas seções cujos sinais existam no projeto auditado. Propriedade do projeto
instruções e CI têm precedência sobre comandos de exemplo. Inspecionar configuração da ferramenta
e supressões antes de interpretar os resultados.

## Ferrugem (`Cargo.toml`)

- Revise `build.rs`, macros proc, aliases/config de `cargo`, dependências de git/caminho,
  recursos, `links`, bibliotecas nativas e código de fornecedor antes da construção.
- Pesquise `unsafe` e alterações de lint proibidas/permitidas; inspecionar qualquer limite FFI.
- Revise `Command`, invocação de shell, `std::env`, manipulação de caminho temporário/sistema de arquivos,
  desserializadores personalizados serde/enums não marcados, regexes, análise de arquivo e assíncrono
  limites de tarefa/fila.
- Verifique conversões de números inteiros, alocação de comprimentos não confiáveis, bloqueio de trabalho em
  caminhos assíncronos, cancelamento, ordem de bloqueio e panic/`unwrap` em caminhos de solicitação.
- Use portas configuradas como `cargo clippy`, `cargo test`, `cargo deny`,
  `cargo audit` e testes de fuzz/propriedade isoladamente após revisão do script de construção.

## JavaScript/TypeScript (`package.json`)

- Use o gerenciador de pacotes/lockfile comprometido. Revise `preinstall`/`postinstall`,
  scripts, substituições/resoluções de pacotes, URLs de git/arquivos, novos registros e
  surpresas transitivas do lockfile antes da instalação.
- Inspecione `child_process`, `eval`/`Function`, importações dinâmicas, protótipo
  poluição, mesclagem de objetos/atribuição em massa, expressões regulares, caminhos do sistema de arquivos, busca de URL,
  coletores de modelo/DOM, serialização e mapas de origem.
- Para servidores Node, audite a confiança do proxy, limites do corpo, CORS/CSRF/cookies, redirecionamentos,
  manipulação de cabeçalho, caminhos de upload, SSRF e limpeza assíncrona de promessas/erros.
- Para aplicativos de navegador, audite coletores XSS, armazenamento de token, verificações de origem postMessage,
  CSP, scripts de terceiros, service workers, retornos de chamada OAuth e do lado do cliente
  pressupostos de autorização.
- Execute comandos lint/type/test/audit/SAST configurados somente após o ciclo de vida/plugin
  análise. Não presuma que a gravidade do `npm audit` é igual à capacidade de exploração.

## Python (`pyproject.toml`, `setup.cfg`, `setup.py`)

- Revise o back-end de construção, ganchos de configuração, plug-ins, dependências editáveis/caminho/VCS,
  índices, restrições/arquivos de bloqueio e extensões nativas antes da instalação.
- Inspecione `subprocess`/shell, `eval`/`exec`, pickle/yaml/desserialização de objeto,
  renderização de modelo, consultas brutas ORM, manipulação de sistema de arquivos/arquivo, URLs/SSRF,
  regexes e importações dinâmicas.
- Modos de depuração da estrutura de auditoria, chaves secretas, configurações de host/CORS/CSRF/sessão,
  autorização de objeto, manipulação de upload, redirecionamentos e cabeçalhos de proxy quando o
  existe uma estrutura web correspondente.
- Use linters/verificações de tipo/testes configurados, além de ferramentas como auditoria de dependência
  ou Bandit somente quando já confiável/configurado ou instalado a partir de um confiável
  fonte isoladamente.

## Vá (`go.mod`)

- Revise as substituições de módulos, fontes de módulos personalizados/privados, código gerado,
  cgo, assembly, `//go:linkname`, plug-ins e comandos `go:generate`. Não corra
  geradores automaticamente.
- Inspecione `os/exec`, wrappers de shell, pacotes de modelos, `unsafe`, sistema de arquivos e
  caminhos de arquivo, clientes/redirecionamentos de URL, SQL, regexes, limites de goroutine/canal,
  cancelamento de contexto, condições de corrida e conversões de inteiro/alocação.
- Use `gofmt`, `go vet`, `go test`, ferramentas race/fuzz/vuln quando aplicável e
  seguro após revisão do gerador/construção.

## Ruby/Rails (`Gemfile`, `.gemspec`, arquivos Rails)

- Revise fontes de gemas, gemas git/path, extensões nativas, ganchos de instalação, tarefas Rake,
  inicializadores, mecanismos e alterações no arquivo de bloqueio antes da execução do Bundler.
- Inspecione `system`/backticks/Open3, eval/send/constantize, YAML/Marshal, SQL bruto,
  Segurança ERB/HTML, redirecionamentos/URLs, caminhos de sistema de arquivos/arquivo, expressões regulares e massa
  atribuição.
- Somente em projetos Rails, filtros de autenticação de auditoria, política/escopo, CSRF,
  chaves de sessão/cookie/criptografia, armazenamento ativo/uploads, IDs assinados, trabalhos,
  Action Cable, configuração de host/proxy e produção específica do ambiente
  configurações.
- Use testes de projeto/lint além de Brakeman configurado e ferramentas de auditoria de dependência.

## JVM (`pom.xml`, `build.gradle`, `build.gradle.kts`)

- Revise plug-ins Maven/Gradle, repositórios, scripts de inicialização/configurações, wrappers,
  processadores de anotação, código gerado e bloqueio/verificação de dependência
  antes de construir.
- Inspecione a execução do processo, carregamento de reflexão/classe, scripts Java/Kotlin,
  desserialização de objeto/XML/YAML, linguagens SpEL/expressão, JNDI, SQL, modelo
  saída, manipulação de sistema de arquivos/arquivo, SSRF, expressões regulares e limites de thread/pool.
- Para estruturas web reais, filtro de auditoria/cobertura da cadeia de segurança, método/objeto
  autorização, CSRF/CORS, endpoints de atuador/depuração, divulgação de erros, upload
  limites e cabeçalhos de proxy.

## .NET (`*.sln`, `*.csproj`, `Directory.Build.*`)

- Revise fontes/arquivos de bloqueio do NuGet, destinos/tarefas do MSBuild, analisadores/fonte
  geradores, eventos pós-construção e bibliotecas nativas antes da construção.
- Inspecione a execução do processo, reflexão/compilação dinâmica, código inseguro,
  desserialização, XML, SQL, Razor/HTML, sistema de arquivos/caminhos de arquivo, URLs/SSRF,
  regexes, cancelamento assíncrono e propriedade de recursos.
- Somente para projetos ASP.NET, audite a ordem do middleware, a autorização do endpoint,
  antifalsificação, CORS/cookies, chaves de proteção de dados, cabeçalhos encaminhados, modelo
  atribuição de ligação/massa, uploads e limites de taxa/corpo.

## PHP/Compositor (`composer.json`)

- Revise scripts/plugins do Composer, repositórios, pacotes de caminho/VCS, carregamento automático
  alterações e integridade do arquivo de bloqueio antes da instalação.
- Inspecione a execução de comandos, inclua/exija caminhos, desserialize, SQL, modelo
  escape, comportamento de upload/caminho/arquivamento de arquivo, URLs/SSRF, redirecionamentos, sessões,
  e expressões regulares.
- Aplicar apenas verificações de autenticação/CSRF/atribuição em massa/depuração/armazenamento específicas da estrutura
  quando essa estrutura estiver presente.

## Elixir/Erlang (`mix.exs`, `rebar.config`)

- Revise aliases/tarefas mistas, dependências hexadecimais/git/caminho, compiladores, NIFs/portas, versões,
  e configuração de tempo de execução antes da construção.
- Inspecione `System.cmd`, criação dinâmica de átomos, decodificação de termos inseguros, Ecto
  fragmentos, segurança do modelo, caminhos do sistema de arquivos, clientes URL, processo/caixa de correio
  limites, loops de supervisão/reinicialização, propriedade ETS e nó distribuído
  cookies/TLS.
- Somente para projetos Phoenix, audite plugs/pipelines, CSRF, soquetes/canais,
  Autorização do LiveView, uploads, segredos de endpoint, cabeçalhos de proxy e taxa/corpo
  limites.

## Shell, Make, CI, contêineres e infraestrutura

Aplicar sempre que existirem arquivos correspondentes, independentemente do idioma do aplicativo.

- Shell: citar expansões, evitar `eval`, separar dados de comandos, proteger
  arquivos temporários, globbing, `IFS`, traps, pipelines, execução de curl/download,
  caminhos destrutivos, vazamento de ambiente e mudanças de privilégios.
- Make/task runners: os alvos podem ser executados durante a configuração/teste; inspecionar inclui,
  expansão do shell, ferramentas baixadas, ambiente e limpeza.
- GitHub Actions/CI: menos permissões, sem segredos com checkout não confiável,
  `pull_request_target` cauteloso, dados de eventos de cotação, ações de fixação por política,
  isolar caches/artefatos, proteger ambientes e separar compilação/publicação.
- Docker: bases/resumos confiáveis, não raiz, recursos mínimos, sem soquete de host ou
  montagens sensíveis, argumentos/segredos de construção seguros, escopo de cópia, verificação de integridade, rede,
  necessidades somente leitura/rootfs/temp e origem de artefato de vários estágios.
- Kubernetes/IaC: RBAC/contas de serviço, namespaces/caminhos de host/privilegiados,
  capacidades, seccomp, segredos, política de rede, exposição pública, metadados
  acesso, segredos de estado/plano, criptografia e deriva destrutiva.

## Nativo/móvel/desktop

Aplicar somente quando existirem destinos nativos/móveis/desktop.

- Revise bibliotecas inseguras/FFI/nativas, esquemas IPC/URL personalizados, soquetes locais,
  assinatura de atualização automática, links diretos, visualizações na web, associações de arquivos, permissões,
  armazenamento de chaves/credenciais, sandbox/direitos e empacotamento de plataforma.
- Trate as pontes de pré-carga/IPC e webview do Electron/Tauri como autorização
  limites; valide o remetente/origem e exponha uma API mínima digitada.
- Verifique caminhos e modelos de permissão específicos da plataforma em cada sistema operacional compatível;
  não projete garantias do Linux no Windows/macOS ou vice-versa.
