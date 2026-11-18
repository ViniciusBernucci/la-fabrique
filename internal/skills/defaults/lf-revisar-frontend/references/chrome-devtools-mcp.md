# Chrome DevTools MCP no review visual

## Vínculo com o navegador da execução

O servidor usado pela skill é o `chrome-devtools-mcp` do projeto Chrome DevTools. No Claude Code, o plugin do Marketplace é uma forma de disponibilizar o servidor. Nos demais executores, o runtime pode expor o mesmo servidor pelo cliente MCP ou adaptador autorizado. A configuração do executor precisa demonstrar que a sessão está vinculada à execução antes de permitir chamadas. [Plugin do Claude Code](https://claude.com/plugins/chrome-devtools-mcp) · [Servidor oficial](https://github.com/ChromeDevTools/chrome-devtools-mcp)

- A fábrica provisiona uma instância em container isolado, com perfil efêmero, rede privada da execução e acesso restrito à aplicação e aos serviços autorizados. Use Chrome ou Chrome for Testing, que têm suporte oficial do servidor. Outro binário Chromium requer comprovação de compatibilidade no preflight. [Compatibilidade](https://github.com/ChromeDevTools/chrome-devtools-mcp#disclaimers)
- Se o runtime iniciar o navegador, configure o MCP com `--browser-url` ou `--ws-endpoint` apontando para o endpoint DevTools privado atribuído à execução. O endpoint DevTools conecta o servidor ao navegador. A URL da aplicação é o destino que o navegador abre. Restrinja o endpoint DevTools aos componentes autorizados e deixe o runtime fornecê-lo. [Configuração do Chrome DevTools](https://developer.chrome.com/docs/devtools/agents/get-started/configuration/)
- Se o servidor iniciar o navegador, a fábrica deve executá-lo no isolamento da tentativa e fornecer perfil efêmero. A opção `--isolated` cria um diretório de perfil temporário. O isolamento de rede, processos, volumes e permissões continua a cargo do runtime. Desative a conexão automática a navegadores ativos e reutilize somente a sessão identificada no despacho. [Opções de inicialização](https://developer.chrome.com/docs/devtools/agents/get-started/configuration/)
- O runtime gerencia instalação, versão e lista de ferramentas permitidas. Para o review comum, a configuração pode desativar telemetria e consultas CrUX com `--no-usage-statistics` e `--no-performance-crux`. A skill consome a configuração disponibilizada. Se o vínculo não for verificável, devolva o bloqueio ao workflow. [Opções e telemetria](https://github.com/ChromeDevTools/chrome-devtools-mcp)

## Ferramentas por cenário

Use os nomes ou aliases que o executor expõe para o servidor oficial. Confira as ferramentas habilitadas em cada tentativa. [Referência oficial de ferramentas](https://github.com/ChromeDevTools/chrome-devtools-mcp/blob/main/docs/tool-reference.md)

| Necessidade | Ferramentas e uso |
|---|---|
| Abrir a rota de teste | `new_page` ou `navigate_page`, conforme o contrato da sessão. Restrinja navegação e redirecionamentos às origens autorizadas. |
| Localizar e interagir | `take_snapshot` para inspecionar os controles. Use `click`, `fill`, `fill_form` ou `press_key` para percorrer o fluxo. Atualize o snapshot quando a interface mudar ou o identificador ficar obsoleto. |
| Observar resultado assíncrono | Use `wait_for` quando um texto esperado sinalizar que a interface chegou ao estado verificável. Registre timeout ou estado diferente do esperado. |
| Verificar viewport e layout | `emulate` ou `resize_page` conforme a capacidade disponível. Use `take_screenshot` no estado relevante e associe a imagem à rota, ao viewport e ao critério. |
| Diagnosticar comportamento | `list_console_messages` e `list_network_requests` para correlacionar falhas com a ação executada. Colete apenas o trecho necessário para a evidência. |

Prefira a interação pela interface para provar o comportamento. Inspeção de DOM ou JavaScript pode ajudar no diagnóstico dentro das permissões concedidas. Evite alterar o estado interno da aplicação para fazer um cenário passar.

## Evidências

Se o MCP devolver a captura na resposta, use o mecanismo de artefatos do runtime para registrar a referência. Se o executor habilitar gravação via `filePath`, use apenas o diretório de artefatos autorizado, conforme interpretado pelo servidor MCP. Preserve dados sintéticos e exclua segredos, cookies, tokens, cabeçalhos de autenticação, endpoints privados do DevTools e dados reais das evidências. Registre somente IDs de artefato confirmados pelo runtime. [Opções de captura](https://github.com/ChromeDevTools/chrome-devtools-mcp/blob/main/docs/tool-reference.md#take_screenshot)
