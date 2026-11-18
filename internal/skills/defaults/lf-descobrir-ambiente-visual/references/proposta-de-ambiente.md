# Proposta de ambiente visual

Use as chaves aceitas pelo workflow quando houver schema configurado. As chaves desta referência são um formato inicial para uso manual ou avaliação. Identificadores oficiais e a versão já validada vêm do despacho. Uma proposta gerada por inspeção estática ainda precisa de revisão humana e validação isolada antes do uso.

## Campos

| Campo | Conteúdo |
|---|---|
| `origem` | Referências recebidas de projeto e repositório, revisão inspecionada, aplicação ou rota alvo e referência da configuração validada anterior, se houver. Se a revisão observada divergir da revisão solicitada, registre a divergência antes de apresentar qualquer conclusão. |
| `resultado` | `PROPOSTA` quando há uma configuração candidata fundamentada e sem lacunas essenciais para revisão. `INCOMPLETA` quando falta uma escolha, evidência ou pré-requisito essencial. `REUSO_SUGERIDO` quando a configuração anterior é compatível com a revisão inspecionada e pode seguir para o preflight. São resultados da descoberta. |
| `servicos` | Serviço e papel, diretório relativo, configuração de execução ou comando e argumentos, dependências, porta interna quando aplicável, alias privado e verificação de prontidão. Inclua serviços auxiliares somente quando forem necessários ao fluxo visual. |
| `navegador` | Rota de entrada para o Chromium isolado via MCP, URL privada ou alias lógico, dependências necessárias para carregar e interagir com a tela e destinos de redirecionamento relevantes. Diferencie URL usada pelo navegador e URL usada entre serviços. Indique proxy e origem da API quando afetarem a navegação. |
| `preparo` | Passos na ordem requerida para construir ou iniciar o ambiente, aplicar migrations de teste, carregar fixtures e provisionar a identidade de teste. Indique em qual serviço cada passo ocorre, o que deve existir antes dele e se pode ser repetido com segurança. |
| `variaveis` | Nomes, serviço consumidor e classificação. Para credenciais, indique a referência ao cofre fornecida pela fábrica ou o vínculo que falta. Registre valores literais apenas se forem públicos, sintéticos e declarados para teste. |
| `limpeza` | Recursos que a execução deve liberar e sua identificação por execução. Inclua estratégia para início parcial, falha ou cancelamento quando houver diferença. Limite remoções a recursos efêmeros da execução cuja identidade o runtime possa comprovar. |
| `fontes` | Para cada campo relevante, caminho relativo e chave, seção ou linha que sustenta a escolha. Cite arquivos inspecionados na revisão informada. Registre `DECLARADO`, `INFERIDO` ou `PENDENTE` por campo, com justificativa para inferências e divergências. |
| `lacunas` | Informação que falta, impacto no review, evidência já consultada, alternativas quando existirem e próxima decisão necessária. Identifique se impede a validação. Campos essenciais desconhecidos ficam pendentes. |
| `comparacao` | Para uma configuração existente, liste escolhas manuais preservadas, alterações sugeridas, fontes que mudaram e verificações afetadas. Se a compatibilidade não puder ser estabelecida, registre o motivo. |

## Prontidão e rede

- Separe serviço iniciado, serviço pronto e aplicação acessível pelo navegador. Uma dependência de inicialização, um processo ativo ou uma porta aberta podem ser insuficientes para provar que migrations, login ou rotas já funcionam. Proponha o check apoiado pelo repositório. Quando não houver evidência, indique qual verificação precisa ser definida.
- Para cada conexão relevante, registre quem inicia a chamada, qual destino usa e como o endereço é resolvido dentro do isolamento. Em uma aplicação com proxy, a rota `/api` pode ser chamada pelo navegador e encaminhada pelo frontend à API em outro container. Se o código cliente chamar a API diretamente, o navegador precisa alcançá-la pela rede aprovada. Marque DNS, TLS, CORS ou callbacks de login como lacunas quando a configuração disponível não os explicar.
- Trate destinos remotos, mounts de host, acesso a daemon de containers e operações destrutivas como efeitos a verificar. Indique a restrição em vez de transpor a configuração de produção. Use referências de imagens ou serviços declarados pelo projeto quando aplicáveis e deixe alterações de privilégios ou rede para a configuração autorizada do runtime.

## Quando reexaminar

Compare os arquivos que justificam a configuração anterior com a revisão examinada. Considere mudanças em scripts e hooks, serviços, imagens, dependências fixadas, portas, rotas, variáveis, migrations, fixtures, login e rede. Considere também novas fontes que alterem a interpretação da configuração. Identifique o efeito concreto sobre a validação. Um novo commit, por si só, não comprova que o ambiente ficou incompatível.

Antes de devolver, confira se cada serviço necessário tem caminho de inicialização e prontidão propostos ou uma lacuna registrada. Verifique se o preparo respeita a ordem dos serviços, se a URL faz sentido a partir do navegador e se a limpeza se limita à execução. A proposta deve indicar o que a Sentinela precisa comprovar em runtime, sem apresentar leitura estática como teste executado.
