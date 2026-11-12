> Leitura: [← Anterior](00-README.md) · [Índice didático](../02-INDEX.md) · [Próximo →](02-ticket-execution.md)

# Navegar e configurar a fábrica

IMPLEMENTADO em a2cc5e0; evidência histórica por FAC-011/013–031. Verificação de software NOT_RUN nesta migração; aceite por ticket permanece no backlog real.

## Usuário e comportamento

Operador administrativo da fábrica. Home demonstrativa com SVG animado e menu persistente entre telas; sidebar inicia recolhida. Configurações em abas/modal com rascunho; agentes/skills por projeto, contas e modelos; login/verificação/PR têm fluxo explícito separado de salvar.

A navegação principal oferece Painel, Projetos, Tarefas, Agentes de IA, Núcleo IA, Pipeline, Revisões, Repositórios, Histórico, Escritório, Usuários e Configurações, nessa ordem. Os cinco itens acrescentados em FAC-034, Escritório e Usuários exibem prévia; não comprovam módulos implementados. [FAC-032](../09-entregas/2026/2026-10-07-FAC-032-menu-inicial-reduzido.md).

## Regras e critério

Critério: cancelar modal descarta campos, sem desfazer job já enviado; cadastro de skill/agente não executa código; PR depende de aprovação exata.

## Falhas e segurança

Auth/versão/digest/estado inválido interrompe a operação pertinente; informação do modelo não concede autoridade. Sem API de IA/extras/fallback/recarga. Gates de implantação e UID são independentes de comportamento codificado.

## Relações e evidências

[controle](../03-modulos/controle/00-README.md), [configuracao](../03-modulos/configuracao/00-README.md), [providers](../03-modulos/providers/00-README.md). [Contratos](../05-contratos/00-README.md), [ADRs](../06-decisoes/00-README.md), [entregas](../09-entregas/00-README.md), [aceites reais](../08-desenvolvimento/09-backlog.md).

## Limite

[AS-IS versus alvo](../02-arquitetura/01-as-is.md). Cenários de tenancy e broker dependem das propostas LF-MT; o catálogo FAC não prova seus controles.

## Painel Tarefas — demonstração FAC-033

IMPLEMENTADO como demonstração local: quadro com nove etapas e lista dos mesmos tickets, pesquisa por título/código/especificação, filtros por projeto/responsável/prioridade/fase/etapa, resumo recalculado para o filtro. Nova tarefa e edição permitem especificação, critérios por linha, prazo, dependências e responsável. Histórico registra criação, edições e mudanças de etapa. Fase do projeto e etapa do ticket são campos distintos.

O onboarding simulado cria projeto, depois cinco agentes habilitados e nove tickets atribuídos à equipe. A ação manual de atribuição automática usa o primeiro agente habilitado do projeto; a escolha por papel/skills pertence ao onboarding real futuro. Sem equipe habilitada, não há atribuição automática; rascunhos podem ficar sem responsável. Sessão administrativa ativa permite consultar projetos e agentes cadastrados por ID; nenhuma tarefa desta tela é enviada à fila real.

O aviso permanente identifica dados mockados. Simular onboarding recria exemplos mediante confirmação, preservando tarefas manuais. Remover tickets de exemplo limpa referências aos exemplos e preserva tarefas manuais/equipe. Exemplos editados continuam exemplos. Armazenamento local não disponível/corrompido é comunicado e não substituído silenciosamente. Multiusuário, sincronização entre abas, persistência PostgreSQL e geração/consumo pelo onboarding/worker real permanecem PLANEJADOS.

Fonte de dados e regras: [contrato da demonstração](../05-contratos/schemas/00-entidades.md#tarefas-demonstrativas-fac-033). [Entrega FAC-033](../09-entregas/2026/2026-10-07-FAC-033-painel-tarefas.md).

## Tema Centro de Comando — FAC-034

IMPLEMENTADO no diff da developer sobre de5dbfa: tema HUD comum, núcleo SVG acessível/determinístico, relógio e saudação locais; navegação/autorizações originais preservadas. Login com anel decorativo; mensagens mantêm conteúdo e role status, com marca visual OK/INFO/ATENÇÃO/ERRO oculta de leitores de tela. Inputs técnicos mono, foco visível, ações de pausa/cancelamento em rosa, tabelas/logs/estados vazios/abas/dialogs seguem os tokens. Os 17 estados de execução reais têm tom próprio; tarefas demonstrativas mantêm suas nove etapas separadas, sem criar enums do kit ilustrativo.

Critérios automatizados e limitações de browser na [entrega FAC-034](../09-entregas/2026/2026-10-07-FAC-034-centro-comando.md). Movimento reduzido por CSS; interações reais, fidelidade visual e revisão independente aguardam validação em ambiente que permita browser/porta local. Nenhuma API, regra de negócio ou credencial alterada.

## Nome dos agentes — FAC-035

Em Configurações → Equipes, abrir um agente e suas configurações permite editar Nome do agente. O cargo anterior continua apresentado separadamente; agentes personalizados permitem editar também o cargo. Salvar grava o nome no JSON versionado, cancelar descarta a edição. Nome pode ser removido, retornando à exibição do cargo, e admite até 100 caracteres após trim. [Funcionamento e evidências](../09-entregas/2026/2026-10-08-FAC-035-nomes-agentes.md).

## Integrações e escolha de IA/modelo — FAC-037

Adicionar conta e Cadastrar integração de IA abrem primeiro duas opções: Cadastrar chave de API e Cadastrar plano de assinatura (CLI). Cancelar descarta a escolha. API permite informar a chave num campo protegido; uma conta salva nunca revela o valor, e campo vazio mantém sua chave. Assinatura usa o fluxo oficial já existente.

Agentes personalizados e pré-configurados escolhem a integração habilitada no campo IA do agente e um modelo do catálogo dessa integração no campo Modelo da IA. Trocar a integração redefine o modelo. Sem cadastro habilitado, a tela orienta cadastrar uma integração; sem catálogo, orienta configurar seus modelos.

Cadastro API foi autorizado pelo responsável em 2026-10-09; não ativa execução ou fallback pago. O runtime atual continua CLI, e a tela informa que execução por API está indisponível. [Configuração](../03-modulos/configuracao/00-README.md#cadastro-api-ou-assinatura-cli--fac-037), [dados](../05-contratos/schemas/00-entidades.md#chaves-de-api-fac-037) e [entrega](../09-entregas/2026/2026-10-09-FAC-037-api-cli-agentes.md).

## Painel da conta do usuário — FAC-038

IMPLEMENTADO no diff feat/fac-038-painel-conta sobre ee4485b. Avatar Minha conta e menu Usuários abrem a camada de usuário independente do login por token da administração existente. Cadastro cria perfil e conta própria; sessões individuais permitem perfil/preferências, nome da conta, membros, convites por link privado, segurança, plano/uso e atividade. [Decisão](../06-decisoes/ADR-007-conta-usuario-e-sessoes.md), [contratos](../05-contratos/api/01-controle.md#api-da-conta-do-usuário--fac-038) e [entrega](../09-entregas/2026/2026-10-09-FAC-038-painel-conta-usuario.md).

Proprietário/admin gerenciam equipe com restrições para outros admins; proprietário permanece ativo. Convite dura 72 horas, é de uso único e exige login com email correspondente **verificado**. Compartilhamento privado continua disponível; SMTP configurado também entrega o convite. Suspensão/remoção revoga sessões e impede admissões novas. O lease solicita PAUSE, preservando writer/fence até confirmação de término.

Abas: Meu perfil, Minha conta, Pessoas e acesso, Segurança, Plano e uso, Atividade, Projetos e execuções, Integrações (gestores), Notificações e Meus dados. A conta reúne membros; cada pessoa mantém perfil, foto raster pequena, idioma/fuso e preferências próprios. Sessões duram sete dias; alterar senha exige a atual e o segundo fator quando ativo, encerra outras sessões e invalida links sensíveis antigos. Cancelar perfil/conta descarta rascunho. Avatar usa PNG/JPEG de até 256 pixels/24 mil caracteres persistidos; a web reduz a foto para 128 pixels.

### Projetos, acesso e execução — FAC-039

Projetos novos têm organizationId. Legados continuam null; nenhum dono é inferido. Gestores criam projetos com URL HTTPS github.com sem credenciais e commit inicial exato, configuram definição/checks/contexto/documentação e grants. Leitor consulta somente projetos concedidos. Operação exige OPERATOR na conta e no projeto; aceite exige APPROVER em ambos. Gestores têm acesso à própria conta. A tela permite trabalhos, fila READY, execuções, entrega por digest, diff/artefatos, pausa/cancelamento e retomada preservada.

Instalações CLI recebem identidade privada `ac-UUID`; cada integração precisa de grant no projeto. Agentes do projeto escolhem somente essas instalações/modelos. Desenvolvedor e revisor são obrigatórios; revisor é READ_ONLY. GitHub usa exatamente uma integração privada com token autorizado ao projeto; o canal do supervisor exige worker/attempt/fence/lease correntes e nunca inclui o token no job/contexto/artifact/UI. Projetos da conta não usam credencial GitHub global. Chaves API podem ser registradas, mas execução API/gastos permanecem desabilitados. Antigravity tem onboarding/verificação existentes, sem adapter de execução.

Autorização de job persiste no READY/retomada e é revalidada em claim/lease. Rotas HTTP, artifacts e stream conferem organização/grant; o stream revalida a cada poll. Desativar integração ou revogar grant impede novas attempts e solicita pausa conservadora. Pause/cancel/recovery não inventam parada física. Storage reserva 16 MiB por attempt antes da admissão para preservar artefatos; a gravação válida converte reserva em bytes reais. Resultado sem snapshot libera reserva na conclusão confirmada; evidência ainda não recebida mantém reserva. Limite de projeto/membro/armazenamento não aumenta o writer global.

### Integrações, uso e dados — FAC-040

Cadastro privado registra serviço, modalidade, modelos, assinatura e custo fixo **informado pelo usuário**. Segredos GitHub/API usam AES-256-GCM com chave dedicada e AAD por registro. Listas/exportações não revelam credenciais. CLI usa login/verificação oficiais existentes pelo worker e identidade da instalação; UI acompanha challenge/código/status. Token GitHub pode ser verificado por consulta ao usuário autenticado; não executa código nem cobra IA. Grants controlam projetos; desativação é versionada e auditada.

Uso agrega projetos/runs/artefatos e reservas reais, respeitando grants do leitor. Tokens vêm de runtimeObservations persistidas, com fonte/data e marca parcial; ausência permanece nullable, sem converter assinatura em preço API/reset inventado. Agregação de tokens é limitada a 10 mil attempts e sinaliza truncamento. Métricas globais de projetos não são expostas a quem não tem grant.

Notificações são geradas por mudanças persistidas de estado de ticket (incluindo espera/limite/login/entrega) e mudanças de acesso; preferências controlam avisos comuns e segurança tem aviso próprio. Leitura individual e grants atuais valem também para mensagens antigas. A aba mostra até 100 notificações dentro da retenção. O proprietário configura dados fiscais e retenção de notificações (7–3650 dias); salvar remove lidas anteriores ao prazo e preserva auditoria/evidências.

Exportação JSON paginada (200 registros por página) inclui perfil, projetos/definições, tickets, runs e, para gestores, membros/auditoria/integrações sanitizadas. Diff/artifacts podem ser baixados separadamente em Projetos e execuções. Não expõe hashes de senha/sessão, tokens/MFA/credenciais. Transferência exige senha/segundo fator, destinatário ativo com email verificado e aceite de link privado em até 24h. Encerramento arquiva a conta/revoga sessões; assinatura deve estar cancelada e nenhum writer pode ter término desconhecido. Evidência histórica permanece preservada.

### Segurança e planos — FAC-041

Verificação/mudança de email e recuperação usam links no fragmento, hashes e tokens de uso único/uma hora. Recuperação não revela existência do email e exige segundo fator se ativo. MFA TOTP SHA1/6 dígitos/30 segundos aceita janela de um passo, impede replay e oferece dez códigos de recuperação armazenados somente como hash. Ativar/desativar protege com senha e encerra outras sessões. Segredo TOTP e corpos SMTP ficam criptografados; entrega SMTP tem retries limitados, message-id estável e sem logging de conteúdo. Queda após aceitação SMTP pode causar reenvio; não alegar exactly-once.

Catálogo de planos é configuração versionada: nomes/preços/cotas reais precisam ser definidos pelo responsável. Sem catálogo, o painel informa ausência; limites operacionais (200 membros, 1000 projetos, 1 GiB) não são preços/benefícios comerciais. Moedas suportadas para novos preços/custos são EUR/BRL/USD/GBP em centavos. Plano gratuito configurado pode ser escolhido localmente. Plano pago verifica preço/moeda/ciclo na Stripe, usa intenção persistida/idempotency key e checkout externo. Resposta desconhecida mantém a mesma intenção; não criar segunda cobrança. Após 24h de outcome desconhecido, reconciliar operacionalmente antes de repetir.

Proprietário gerencia troca/cancelamento pelo portal e consulta faturas. Webhook usa raw body, assinatura com janela de cinco minutos, dedupe transacional e consulta o estado corrente da Stripe sob serialização; retorno do navegador nunca prova pagamento. Inadimplência/cancelamento bloqueiam novas execuções da conta sem apagar dados. SMTP/Stripe são adaptadores **não ativados** nesta entrega: nenhuma credencial real, email externo ou cobrança foi utilizado. Cadastro/rollout público, isolamento hostil LF-MT e aceite independente continuam gates separados. Administração global não faz parte deste painel.

[Entrega complementar FAC-039–041](../09-entregas/2026/2026-10-09-FAC-041-complemento-painel-conta.md) · [Operação](../07-operacao/01-deploy.md#configuração-de-email-segurança-e-planos--fac-039041).

FAC-041: tempo de execução soma apenas tentativas com término confirmado; tokens observados também são agrupados por provider. Resultados desconhecidos/ausentes e limite de 10.000 tentativas tornam a agregação parcial, sem estimar consumo ou custo. Exportação inclui tentativas e resultados sanitizados, além das demais seções da conta.

## Instruções em Markdown — FAC-036

Configurações → Equipes → modal de agente ou skill oferece descrição breve e área separada para arquivo .md. Adicionar/substituir prepara o rascunho; salvar registra o conteúdo e o nome; cancelar preserva o cadastro anterior. Baixar arquivo .md permite recuperar instruções registradas, inclusive conteúdo anterior ao upload. Arquivo inválido mostra erro e mantém o conteúdo anterior. [Regras e limites](../03-modulos/configuracao/00-README.md#arquivos-de-instruções--fac-036) · [Entrega](../09-entregas/2026/2026-10-09-FAC-036-instrucoes-markdown.md).
