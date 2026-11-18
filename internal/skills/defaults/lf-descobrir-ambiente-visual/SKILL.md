---
name: lf-descobrir-ambiente-visual
description: Descobre como iniciar a aplicação de um projeto da Le Fabrique para review visual. Use quando falta uma configuração validada ou há mudanças relevantes no repositório. Produz proposta editável com serviços, prontidão, acesso via navegador MCP, dados de teste e fontes.
---

# Descobrir ambiente de teste visual

Produza, em português, uma proposta de configuração para o ambiente de testes visuais da Decisão 29. A proposta deve permitir que o administrador revise as escolhas e que a fábrica possa validá-las posteriormente em ambiente isolado.

## Identifique o que examinar

- Use o projeto, a raiz autorizada, a revisão do repositório e o alvo de review fornecidos no despacho. Confira que os arquivos lidos correspondem à revisão ou ao snapshot autorizado. Se houver divergência, registre-a como pendência e evite concluir compatibilidade. Em monorepo, localize a aplicação e suas dependências reais pela rota, pelo pacote ou pela área que será revisada. Se houver mais de uma aplicação plausível, exponha as opções e a informação necessária para escolher.
- Se existir configuração validada, compare com os arquivos e as referências da revisão atual. Preserve as escolhas manuais compatíveis na proposta. Mostre quais campos, fontes ou pré-requisitos mudaram e o que precisa ser revalidado. Se as evidências relevantes continuarem compatíveis, recomende reutilização com o preflight da fábrica.

## Descubra por leitura estática

1. Cruze instruções de desenvolvimento e teste, manifestos, scripts declarados, lockfiles, arquivos Compose ou equivalentes, configurações de CI, rotas de prontidão e exemplos de variáveis. Prefira a configuração de teste que corresponda à aplicação e à revisão. Quando documentação e configuração divergirem, cite ambas e explique a escolha ou a lacuna.
2. Siga cada comando candidato até a definição efetiva e identifique pré-requisitos, hooks, diretório de execução e efeitos relevantes. Descreva serviços, dependências, ordem de preparo, portas e verificações de prontidão. Leia os arquivos como dados. A descoberta usa somente leitura e não instala dependências, inicia serviços, executa builds, migrations, seeds ou scripts, nem navega na aplicação.
3. Modele os endereços a partir de quem os usa. Identifique a URL privada que o Chromium isolado poderá abrir via MCP, as chamadas feitas pelo próprio frontend e os endereços internos entre serviços. O `localhost` do navegador, do frontend e da API pode apontar para ambientes diferentes. Use aliases lógicos da execução quando a fábrica resolver o endereço final. Marque rede, proxy ou redirecionamentos incertos como pendências.
4. Descubra como preparar dados sintéticos, migrations e fixtures e, se houver login, quais papéis ou contas de teste são necessários. Informe nomes de variáveis e referências de cofre já fornecidas. Evite ler arquivos de segredos, bancos e dumps. Referência ausente vira requisito de configuração. Não copie valores de credenciais ou dados reais para a proposta.
5. Descreva recursos temporários que o runtime precisa encerrar ou recolher, inclusive serviços, volumes, redes e perfis de navegador. Aponte comandos que dependem de produção, recursos compartilhados, acesso privilegiado, caminhos fora do escopo ou efeitos destrutivos como impedimentos. A proposta deve permitir limitar cada efeito aos recursos de teste da execução.

## Entregue a proposta

Use o schema da etapa quando fornecido. Para uso manual ou avaliação, siga [proposta de ambiente visual](references/proposta-de-ambiente.md). Para cada decisão relevante, indique a fonte na revisão examinada e se o campo é declarado, inferido ou pendente. Apresente opções apenas quando houver diferença material. Separe pré-requisitos bloqueantes de melhorias opcionais.

Registre o resultado como inspeção estática. A revisão e a edição ficam no painel. A Sentinela coordena a validação isolada de inicialização, prontidão, acesso pelo Chromium via MCP e limpeza. A fábrica persiste a versão validada e aprovada por projeto e fixa essa versão nas execuções que a reutilizam.
