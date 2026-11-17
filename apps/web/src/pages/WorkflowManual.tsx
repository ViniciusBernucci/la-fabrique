import { Fields, ManualDialog, type Section } from "../components/ManualDialog";

const c = (text: string) => <code>{text}</code>;

const sections: Section[] = [
  {
    id: "visao",
    title: "O que é um workflow (comece por aqui)",
    body: (
      <>
        <p>
          Um <strong>workflow</strong> é um <strong>fluxograma</strong> que diz à fábrica, passo a
          passo, o que fazer para entregar uma tarefa: quem planeja, quem programa, quem revisa, o
          que testar, quando abrir a Pull Request e quando esperar uma pessoa. Cada caixinha do
          desenho é um <strong>nó</strong> (um passo) e cada seta é uma <strong>conexão</strong>{" "}
          (“terminou este passo, vá para aquele”).
        </p>
        <p>
          Você <strong>não precisa saber programar</strong> para montar um fluxo: basta saber
          descrever o processo com palavras (“primeiro planejar, depois programar, depois testar”) e
          traduzir isso em caixinhas ligadas por setas. Este manual explica cada tipo de caixinha,
          cada campo que ela pede e a pequena “linguagem de condições” usada para tomar decisões.
        </p>
        <h4>Dois tipos de workflow</h4>
        <Fields
          rows={[
            [
              "Fluxo de feature",
              "O fluxo principal de uma tarefa (uma funcionalidade ou correção). É o que um projeto usa para trabalhar do começo ao fim.",
            ],
            [
              "Subfluxo",
              "Um pedaço reutilizável de fluxo (por exemplo, “rodar os testes e corrigir”). Outros fluxos o chamam por um nó do tipo Subfluxo, em vez de redesenhar o mesmo trecho várias vezes.",
            ],
          ]}
        />
        <h4>Rascunho: o que ele faz e não faz</h4>
        <p>
          Tudo o que você desenha aqui é um <strong>rascunho</strong>. Um rascunho{" "}
          <strong>nunca executa</strong> nada: ele só guarda o desenho. Você pode errar, testar e
          refazer sem nenhum efeito sobre o projeto. Quando um fluxo é <em>publicado</em>, uma cópia
          congelada dele passa a valer para as execuções (alterar o rascunho depois não muda o que
          já foi publicado). Esta tela edita o rascunho; a publicação e a execução real ainda estão
          sendo ligadas ao painel.
        </p>
      </>
    ),
  },
  {
    id: "editor",
    title: "Como usar o editor, passo a passo",
    body: (
      <>
        <ol>
          <li>
            <strong>Crie</strong> um workflow na lista (Nome e Tipo) e clique em{" "}
            <em>Criar rascunho</em>. Ele já nasce com um nó <strong>Início</strong> ligado a um nó{" "}
            <strong>Fim</strong>.
          </li>
          <li>
            <strong>Adicione nós:</strong> na coluna da esquerda (“Adicionar nó”) clique no tipo
            desejado ou arraste-o para o canvas. O nó novo aparece à vista e já fica selecionado.
          </li>
          <li>
            <strong>Mova os nós:</strong> arraste pelo cartão. O canvas é infinito: arraste o fundo
            ou use a rolagem para passear; use <kbd>Ctrl</kbd> (ou <kbd>⌘</kbd>) + rolagem para dar
            zoom.
          </li>
          <li>
            <strong>Conecte:</strong> cada nó tem bolinhas nas bordas. As da <em>esquerda</em> são{" "}
            <strong>entradas</strong> (por onde o fluxo chega) e as da <em>direita</em> são{" "}
            <strong>saídas</strong> (por onde o fluxo sai). Clique na bolinha de uma saída, segure e
            arraste até a bolinha de entrada de outro nó. Se preferir sem arrastar, selecione o nó e
            use, no painel da direita, a lista <em>“Segue para”</em> de cada saída.
          </li>
          <li>
            <strong>Preencha os campos:</strong> clique no nó; o painel da direita mostra os campos
            dele. Os marcados com <strong>*</strong> são obrigatórios. Cada tipo de nó está
            explicado nas próximas seções.
          </li>
          <li>
            <strong>Confira a validação:</strong> a seção “Validação” abaixo do canvas lista tudo o
            que ainda está errado, com o nome do nó e o motivo; o botão <em>Localizar</em> leva até
            o nó. Nós com problema mostram um círculo vermelho com “!”. O fluxo só está pronto
            quando aparece “Nenhum problema encontrado”.
          </li>
          <li>
            <strong>Salve</strong> em <em>Salvar rascunho</em>. O selo no topo mostra “Alterações
            não salvas” enquanto houver mudança pendente.
          </li>
        </ol>
        <h4>Os botões do canvas</h4>
        <Fields
          rows={[
            [
              "Organizar",
              "Arruma os nós da esquerda para a direita, na ordem do fluxo, sem sobrepor. Útil quando o desenho ficou bagunçado.",
            ],
            ["Ajustar à tela", "Muda o zoom para mostrar o fluxo inteiro."],
            [
              "Remover seleção",
              "Apaga o nó ou a conexão selecionados (a tecla Delete também funciona). Apagar um nó apaga as conexões ligadas a ele.",
            ],
            [
              "Controles (canto inferior esquerdo)",
              "Aproximar, afastar e ajustar à tela. O mapinha no canto inferior direito mostra onde você está no desenho.",
            ],
          ]}
        />
        <p className="muted">
          Dica: ligar de novo uma saída que já tinha conexão <strong>substitui</strong> a antiga.
          Cada saída só pode levar a um lugar.
        </p>
      </>
    ),
  },
  {
    id: "conceitos",
    title: "Conceitos e regras básicas",
    body: (
      <>
        <Fields
          rows={[
            [
              "Nó",
              "Um passo do fluxo. Tem um tipo (agente, teste, condição…), um identificador único (por exemplo agent-1) e campos para preencher.",
            ],
            [
              "Porta",
              "Cada nó tem portas de entrada (esquerda) e de saída (direita). O nome da porta aparece ao lado: out (saiu bem), error (deu erro), true/false (resposta de uma pergunta), a/b (dois ramos), body/done (dentro do laço / terminou).",
            ],
            ["Conexão", "A seta que liga uma porta de saída à porta de entrada de outro nó."],
            [
              "Execução",
              "Uma “rodada” do fluxo: a fábrica entra pelo Início e segue as setas, um nó por vez, até chegar a um Fim.",
            ],
            [
              "Identificador (ID)",
              "O nome interno do nó (letras, números, _ e -). Aparece pequeno abaixo do título do cartão e é o que você usa nas condições.",
            ],
          ]}
        />
        <h4>Regras que o editor verifica por você</h4>
        <ul>
          <li>
            Exatamente <strong>um</strong> nó Início e <strong>pelo menos um</strong> nó Fim.
          </li>
          <li>
            Toda <strong>entrada</strong> precisa estar ligada a alguma saída, e toda{" "}
            <strong>saída</strong> também (a única exceção é a saída <code>error</code>, que é
            opcional).
          </li>
          <li>
            Cada <strong>saída</strong> leva a <strong>um único</strong> destino. Para seguir por
            dois caminhos use uma Condição (escolhe um) ou uma Ramificação (vai pelos dois).
          </li>
          <li>
            Cada <strong>entrada</strong> aceita <strong>uma única</strong> seta, exceto nos nós que
            existem para juntar caminhos: Convergência, Loop e Fim.
          </li>
          <li>Todo nó precisa ser alcançável a partir do Início (nada “solto” no desenho).</li>
          <li>
            Um fluxo <strong>não pode voltar para trás</strong> sozinho: se uma seta faz o fluxo
            retornar a um passo anterior (um ciclo), o caminho precisa passar por um nó{" "}
            <strong>Loop limitado</strong>, que impõe um número máximo de voltas.
          </li>
          <li>Um nó não pode se ligar a si mesmo, e não pode haver duas setas idênticas.</li>
          <li>Um fluxo tem no máximo 200 nós e 400 conexões; cada campo, até 4000 caracteres.</li>
        </ul>
      </>
    ),
  },
  {
    id: "setas",
    title: "Os tipos de setas, cores e bolinhas",
    body: (
      <>
        <p>
          O desenho “fala” por cores e traços. Toda seta tem uma <strong>ponta</strong> que mostra o
          sentido do fluxo (de onde sai para onde vai). O <strong>estilo da linha</strong> depende
          da saída de onde ela parte, para você enxergar o caminho bom e o caminho ruim sem ler
          nada.
        </p>
        <h4>As setas</h4>
        <Fields
          rows={[
            [
              "Cinza, linha contínua",
              <>
                O caminho <strong>normal</strong>: saídas {c("out")} (terminou bem), {c("a")} e{" "}
                {c("b")} (os dois ramos de uma Ramificação) e {c("body")} (o trecho que o Loop
                repete).
              </>,
            ],
            [
              "Verde-azulada, linha contínua",
              <>
                O caminho do <strong>“sim”</strong> ou de <strong>conclusão</strong>: saídas{" "}
                {c("true")} (a condição foi verdadeira), {c("approved")} (a pessoa aprovou) e{" "}
                {c("done")} (o Loop terminou).
              </>,
            ],
            [
              "Vermelha, tracejada (traços)",
              <>
                O caminho <strong>ruim</strong> ou do <strong>“não”</strong>: saídas {c("error")} (o
                passo falhou), {c("false")} (a condição foi falsa) e {c("rejected")} (a pessoa
                recusou). Se o desenho está cheio de tracejados vermelhos, é a rota de falhas.
              </>,
            ],
            [
              "Mais grossa e verde-azulada",
              <>
                Uma seta que você <strong>selecionou</strong> (clicou nela) ou sobre a qual passou o
                mouse. Com ela selecionada, “Remover seleção” (ou a tecla Delete) a apaga.
              </>,
            ],
            [
              "Linha verde-azulada que “segue o mouse”",
              "A seta sendo criada: você está arrastando de uma saída e ainda não soltou em uma entrada. Solte sobre a bolinha de entrada para fixá-la; soltar no vazio cancela.",
            ],
          ]}
        />
        <p className="muted">
          A cor e o traço vêm da <strong>saída de origem</strong>, não do destino. Mudar o destino
          de uma seta não muda o estilo dela; ligar a saída de erro a outro nó mantém a seta
          vermelha tracejada.
        </p>
        <h4>As bolinhas (portas)</h4>
        <Fields
          rows={[
            [
              "Bolinha à esquerda do cartão",
              "Uma entrada: é onde as setas chegam. Fica cinza; vira verde-azulada quando você aproxima uma seta que está sendo criada.",
            ],
            [
              "Bolinha à direita do cartão",
              "Uma saída: é de onde as setas partem. Arraste dela para criar a seta. O nome da saída aparece escrito ao lado (error, true, false…); a saída única “out” não precisa de nome.",
            ],
            [
              "Borda vermelha na bolinha",
              "Saídas error, false e rejected (o caminho ruim). A borda verde-azulada marca true, approved e done.",
            ],
          ]}
        />
        <h4>As bordas dos cartões</h4>
        <Fields
          rows={[
            [
              "Cor do ícone",
              "Verde-azulado: nós de fluxo (Início e Fim). Roxo: nós de execução (agente, skill, script, checks, git, documentação). Âmbar: nós de controle (condição, loop, espera, decisão humana, subfluxo, ramificação, convergência).",
            ],
            [
              "Contorno verde-azulado com halo",
              "O nó está selecionado; o painel da direita mostra os campos dele.",
            ],
            [
              "Contorno vermelho + círculo com “!”",
              "O nó tem pendências de validação (campo obrigatório vazio, saída solta…). O resumo em vermelho no cartão (“Escolha o agente”) diz o que falta. A lista completa está em “Validação”.",
            ],
          ]}
        />
      </>
    ),
  },
  {
    id: "inicio-fim",
    title: "Nós de começo e fim: Início e Fim",
    body: (
      <>
        <h4>Início</h4>
        <p>
          Onde toda execução começa. Tem <strong>só uma saída</strong> (<code>out</code>) e nenhum
          campo. Todo fluxo tem exatamente um.
        </p>
        <h4>Fim</h4>
        <p>
          Onde um caminho termina. Não tem saída. Pode haver vários Fins (um para o caminho que deu
          certo, outros para os que falharam).
        </p>
        <Fields
          rows={[
            [
              "Resultado *",
              <>
                <code>success</code> (sucesso): o caminho terminou bem. <code>failure</code>{" "}
                (falha): o caminho terminou mal. Use Fins de falha para onde as saídas de erro devem
                levar, e Fins de sucesso para o resultado desejado.
              </>,
            ],
          ]}
        />
        <p className="muted">
          Se um nó de trabalho falha e você não ligou a saída de erro dele a lugar nenhum, a
          execução inteira é marcada como falha na hora.
        </p>
      </>
    ),
  },
  {
    id: "trabalho",
    title: "Nós de execução (os que fazem trabalho)",
    when: "Todos têm uma entrada (in) e duas saídas: out (terminou bem) e error (falhou). A saída error é opcional.",
    body: (
      <>
        <p>
          São os nós que realmente <em>fazem</em> algo. Ao terminar bem, a execução segue por{" "}
          <code>out</code>; se falhar, segue por <code>error</code>. Ligue <code>error</code> a um
          Fim de falha, a um passo de correção ou a um aviso. Sem essa ligação, a falha derruba a
          execução.
        </p>

        <h4>Agente (IA)</h4>
        <p>
          Entrega um trabalho a um <strong>agente de inteligência artificial</strong> do catálogo
          (veja a tela <em>Agentes</em>): planejar, programar, revisar, escrever documentação… O
          agente já traz instruções, skills, conta e modelo configurados; o nó diz <em>qual</em>{" "}
          agente e <em>o que pedir a ele agora</em>.
        </p>
        <Fields
          rows={[
            ["Agente *", "Escolha na lista o agente do catálogo que fará o trabalho."],
            [
              "Instrução *",
              "O pedido desta etapa, em linguagem natural, como se falasse com uma pessoa. Ex.: “Implementar as subtarefas de backend e rodar os testes.”",
            ],
            [
              "Tempo máximo (min)",
              "Quanto o agente pode demorar (1 a 720 minutos). Vazio usa o tempo do cadastro do agente.",
            ],
            [
              "Modelo (sobrescreve o agente)",
              "Opcional. Troca o modelo de IA só neste passo (ex.: um modelo mais forte para planejar). Vazio usa o do agente.",
            ],
            [
              "Conta (conexão, sobrescreve)",
              "Opcional. ID de uma conexão de IA (tela Conexões) para usar só neste passo. Vazio usa a conta do agente.",
            ],
            [
              "Ferramentas (só reduz as do agente)",
              "Opcional. Lista, separada por vírgulas, das ferramentas permitidas neste passo. Só pode restringir: não dá ao agente mais poder do que o cadastro dele permite.",
            ],
            [
              "Instruções adicionais",
              "Opcional. Texto extra somado às instruções do agente apenas neste passo.",
            ],
          ]}
        />
        <p className="muted">
          Depois de salvar, o botão <em>Ver configuração efetiva</em> mostra, com a origem de cada
          valor (“do agente” ou “do nó”), exatamente o que o agente usará.
        </p>

        <h4>Skill</h4>
        <p>
          Executa uma <strong>skill</strong> (um procedimento pronto do catálogo <em>Skills</em>){" "}
          <strong>sozinha, sem agente de IA</strong>. Serve para tarefas repetitivas e previsíveis.
          As informações que a skill produz ficam disponíveis para as condições seguintes (veja
          “Variáveis”).
        </p>
        <Fields
          rows={[
            ["Skill *", "Escolha na lista a skill do catálogo."],
            ["Entrada", "Opcional. Um texto entregue à skill como dado de entrada."],
          ]}
        />

        <h4>Script</h4>
        <p>
          Roda um comando de terminal <strong>dentro de uma caixa isolada</strong> (sandbox), longe
          do seu computador e <strong>sem acesso à internet</strong>.
        </p>
        <Fields
          rows={[
            ["Comando *", "O comando a executar. Ex.: npm run build."],
            [
              "Perfil do sandbox *",
              <>
                A “caixa” onde o comando roda. {c("base")}: Linux mínimo (1 CPU, 256 MB), para
                comandos simples. {c("go")}: traz o Go (2 CPUs, 1 GB). {c("node")}: traz o Node (2
                CPUs, 1 GB). Escolha o que tem as ferramentas que o comando precisa.
              </>,
            ],
            ["Tempo máximo (min)", "Opcional (1 a 720). Se passar, o passo falha."],
          ]}
        />

        <h4>Testes e checks</h4>
        <p>
          Roda <strong>um ou mais comandos de verificação</strong> (formatação, análise, testes) e
          diz se passaram. Se algum falhar, o passo sai por <code>error</code>.
        </p>
        <Fields
          rows={[
            [
              "Comandos (um por linha) *",
              "Um comando por linha, executados em ordem. Ex.: uma linha “make lint” e outra “make test”.",
            ],
            ["Perfil do sandbox *", "Igual ao do Script: base, go ou node."],
          ]}
        />

        <h4>Git</h4>
        <p>
          Faz uma operação de versionamento do código do projeto. Operações que{" "}
          <strong>publicam fora da fábrica</strong> (enviar a branch, abrir a Pull Request) estão
          sujeitas às autorizações da <em>Central de Decisões</em>: a fábrica pode parar e pedir o
          seu “sim”.
        </p>
        <Fields
          rows={[
            [
              "Operação *",
              <>
                {c("branch")}: cria a branch de trabalho. {c("commit")}: grava as alterações num
                commit. {c("push")}: envia a branch ao GitHub. {c("draft_pr")}: abre uma Pull
                Request em rascunho (ainda não integrada).
              </>,
            ],
            [
              "Mensagem",
              "Opcional. Texto do commit (ou título da PR). Ex.: “Consolidar commits da feature”.",
            ],
          ]}
        />

        <h4>Documentação</h4>
        <p>Pede a atualização de um arquivo ou pasta de documentação.</p>
        <Fields
          rows={[
            ["Arquivo ou pasta *", "O que deve ser documentado. Ex.: docs/manual.md ou docs/api/."],
            ["Instrução *", "O que escrever ou atualizar, em linguagem natural."],
          ]}
        />
      </>
    ),
  },
  {
    id: "controle",
    title: "Nós de controle (os que decidem e organizam)",
    body: (
      <>
        <p>
          Estes nós não “trabalham”: eles <strong>decidem o caminho</strong>, repetem, esperam ou
          juntam ramos.
        </p>

        <h4>Condição (IF/ELSE)</h4>
        <p>
          Faz uma <strong>pergunta de sim ou não</strong> e escolhe o caminho: se a resposta for
          sim, sai por <code>true</code>; se for não, por <code>false</code>. As{" "}
          <strong>duas saídas precisam estar ligadas</strong>.
        </p>
        <Fields
          rows={[
            [
              "Expressão *",
              <>
                A pergunta, escrita na linguagem de condições (veja a seção “Expressões”). Ex.:{" "}
                {c("fabrica.correcao == true")} pergunta “esta execução é uma correção de PR?”.
              </>,
            ],
          ]}
        />

        <h4>Loop limitado</h4>
        <p>
          Faz o fluxo <strong>repetir um trecho</strong> até uma condição deixar de valer, com um{" "}
          <strong>limite máximo de voltas</strong> para nunca ficar preso para sempre. Tem duas
          saídas: <code>body</code> (o trecho a repetir) e <code>done</code> (terminou). O fim do
          trecho repetido <strong>volta por uma seta</strong> para a entrada do Loop (por isso a
          entrada dele aceita mais de uma seta).
        </p>
        <Fields
          rows={[
            [
              "Máximo de iterações *",
              "De 1 a 100. Nunca passa disso, mesmo que a condição continue verdadeira.",
            ],
            [
              "Continuar enquanto *",
              <>
                Uma expressão. Cada vez que o fluxo chega ao Loop, ela é avaliada: se for{" "}
                <strong>verdadeira</strong> e ainda não atingiu o máximo, vai por <code>body</code>{" "}
                (mais uma volta); se for falsa, vai por <code>done</code>. Se atingir o máximo com a
                condição ainda verdadeira, vai por <code>done</code> e marca{" "}
                {c("loop_<id>_exhausted")} como verdadeiro (veja “Variáveis”).
              </>,
            ],
          ]}
        />
        <p className="muted">
          Truque útil: com a condição {c("true")} o Loop repete exatamente o número de vezes do
          máximo, o que serve para “tentar até 3 vezes”.
        </p>

        <h4>Espera durável</h4>
        <p>
          Pausa o fluxo por um tempo e depois continua. É <strong>durável</strong>: a espera
          sobrevive a reinícios do sistema, porque o estado fica guardado. Saída única{" "}
          <code>out</code>.
        </p>
        <Fields
          rows={[
            [
              "Aguardar (min) *",
              "De 1 a 10080 minutos (7 dias). 1440 = 1 dia. Ex.: esperar um dia pela revisão humana de uma PR.",
            ],
          ]}
        />

        <h4>Decisão humana</h4>
        <p>
          Para o fluxo e <strong>pergunta a uma pessoa</strong>; o fluxo só continua quando alguém
          responde. Saídas: <code>approved</code> (aprovou) e <code>rejected</code> (recusou) —{" "}
          <strong>as duas precisam estar ligadas</strong>.
        </p>
        <Fields
          rows={[
            [
              "Pergunta *",
              "O que a pessoa deve decidir, escrito de forma que dê para responder sim ou não. Ex.: “Posso enviar a branch ao GitHub?”.",
            ],
          ]}
        />

        <h4>Subfluxo</h4>
        <p>
          Chama <strong>outro workflow</strong> (do tipo Subfluxo) como se fosse um único passo. Na
          prática o subfluxo é “encaixado” no ponto onde está o nó. Se o subfluxo termina em um Fim
          de sucesso, a execução segue por <code>out</code>; se termina em um Fim de falha, segue
          por <code>error</code> (quando ligada).
        </p>
        <Fields
          rows={[
            [
              "Subfluxo (ID) *",
              "O ID do workflow do tipo Subfluxo (aparece no endereço da página dele, depois de /workflows/). Um subfluxo não pode chamar a si mesmo, direta ou indiretamente, e os aninhados podem ter no máximo 5 níveis.",
            ],
          ]}
        />

        <h4>Ramificação</h4>
        <p>
          Faz o fluxo seguir <strong>pelos dois caminhos ao mesmo tempo</strong> (saídas{" "}
          <code>a</code> e <code>b</code>, as duas ligadas). Diferente da Condição, que escolhe um.
          Hoje os dois ramos são executados um depois do outro, na ordem a, b. Não tem campos.
        </p>

        <h4>Convergência</h4>
        <p>
          Faz o oposto da Ramificação: <strong>junta caminhos</strong> em um só. Sua entrada aceita
          várias setas.
        </p>
        <Fields
          rows={[
            [
              "Modo *",
              <>
                {c("all")}: espera <strong>todos</strong> os caminhos chegarem antes de continuar
                (use depois de uma Ramificação, para esperar os dois lados). {c("any")}: continua
                assim que <strong>o primeiro</strong> chegar (use quando caminhos diferentes levam
                ao mesmo próximo passo, como no início do fluxo padrão).
              </>,
            ],
          ]}
        />
      </>
    ),
  },
  {
    id: "expressoes",
    title: "Expressões: a linguagem das condições",
    body: (
      <>
        <p>
          Uma <strong>expressão</strong> é uma frase curta que a fábrica consegue responder com{" "}
          <strong>verdadeiro ou falso</strong>. Ela é usada em dois lugares: no campo{" "}
          <em>Expressão</em> do nó Condição e no campo <em>Continuar enquanto</em> do nó Loop.
        </p>

        <h4>Exemplo, palavra por palavra</h4>
        <p>
          {c("fabrica.correcao == true")} se lê: “o valor guardado em{" "}
          <strong>fabrica.correcao</strong> <strong>é igual a</strong> <strong>verdadeiro</strong>
          ?”.
        </p>
        <Fields
          rows={[
            [
              "fabrica.correcao",
              "Um nome de variável (uma “gaveta” com um valor dentro). O ponto separa o grupo do item: dentro do grupo “fabrica” está a gaveta “correcao”.",
            ],
            ["==", "O sinal de “é igual a”."],
            ["true", "O valor “verdadeiro” (literal)."],
          ]}
        />
        <p>
          No fluxo padrão, <code>fabrica.correcao</code> vale verdadeiro quando a execução é uma{" "}
          <strong>correção de uma Pull Request</strong> já existente e falso (ou ausente) numa
          funcionalidade nova. Por isso essa condição manda correções direto para o planejamento,
          pulando a checagem do PRD.
        </p>

        <h4>Os “ingredientes” permitidos</h4>
        <Fields
          rows={[
            [
              "Verdadeiro, falso, nulo",
              <>
                As palavras {c("true")}, {c("false")} e {c("null")} (nulo = “sem valor”).
              </>,
            ],
            [
              "Números",
              <>
                {c("3")}, {c("0")}, {c("-1")}, {c("2.5")} (com ponto, não vírgula).
              </>,
            ],
            [
              "Textos",
              <>
                Entre aspas simples ou duplas: {c("'aprovado'")} ou {c('"aprovado"')}. Maiúsculas e
                minúsculas importam: “Ok” é diferente de “ok”.
              </>,
            ],
            [
              "Variáveis",
              <>
                Nomes com letras, números e _, separados por ponto: {c("prd.concluido")},{" "}
                {c("agent_1.resumo")}. Não há colchetes nem posições de lista.
              </>,
            ],
          ]}
        />

        <h4>Os operadores</h4>
        <Fields
          rows={[
            [
              "==   !=",
              <>
                “é igual a” e “é diferente de”. Ex.: {c("last.status != 'erro'")}. Números são
                comparados como números (3 e 3.0 são iguais).
              </>,
            ],
            [
              "<   <=   >   >=",
              <>
                “menor”, “menor ou igual”, “maior”, “maior ou igual”. Só funcionam com{" "}
                <strong>números</strong>; com texto ou valor ausente a execução falha com “expressão
                inválida”. Ex.: {c("testes.falhas > 0")}.
              </>,
            ],
            [
              "&&",
              <>
                “<strong>e</strong>”: só é verdadeiro se os dois lados forem. Ex.:{" "}
                {c("prd.concluido == true && fabrica.correcao == false")}.
              </>,
            ],
            [
              "||",
              <>
                “<strong>ou</strong>”: verdadeiro se pelo menos um lado for. Ex.:{" "}
                {c("fabrica.correcao == true || prd.concluido == true")}.
              </>,
            ],
            [
              "!",
              <>
                “<strong>não</strong>”: inverte. Ex.: {c("!prd.concluido")} é verdadeiro quando o
                PRD <em>não</em> está concluído.
              </>,
            ],
          ]}
        />

        <h4>Regras de leitura</h4>
        <ul>
          <li>
            <strong>Ordem:</strong> primeiro as comparações (<code>==</code>, <code>&lt;</code>…),
            depois <code>&amp;&amp;</code>, por último <code>||</code>.
          </li>
          <li>
            <strong>Não há parênteses</strong>, nem contas (+ − × ÷), nem funções. Se a pergunta
            ficar complicada, quebre em <strong>duas Condições seguidas</strong>.
          </li>
          <li>Uma comparação por vez: {c("a < b < c")} não existe.</li>
          <li>O texto da expressão pode ter até 500 caracteres.</li>
        </ul>

        <h4>O que conta como “verdadeiro” sozinho</h4>
        <p>
          Quando a expressão é só um valor (por exemplo {c("prd.concluido")}, sem “== true”), vale a
          regra: são <strong>falsos</strong> {c("false")}, {c("null")}, o texto vazio {c("''")} e o
          número zero; <strong>todo o resto é verdadeiro</strong>. Para evitar dúvida, prefira
          escrever {c("== true")} explicitamente, como no fluxo padrão.
        </p>

        <h4>Variável que não existe</h4>
        <p>
          Se a variável não existir ainda, o valor é <code>null</code> (sem valor) — não é um erro.
          Assim, {c("prd.concluido == true")} dá <strong>falso</strong> enquanto ninguém tiver
          informado que o PRD terminou. Mas cuidado: {c("x < 3")} com {c("x")} ausente{" "}
          <strong>falha</strong> (comparação de ordem exige número).
        </p>

        <h4>Exemplos prontos</h4>
        <Fields
          rows={[
            [c("fabrica.correcao == true"), "É uma correção de PR?"],
            [c("prd.concluido == true"), "O PRD da tarefa já está concluído?"],
            [c("!fabrica.correcao"), "NÃO é correção (feature nova)."],
            [c("true"), "Sempre verdadeiro (no Loop: repete até o máximo)."],
            [c("last.status == 'ok'"), "O último nó de trabalho devolveu status “ok”?"],
            [
              c("loop_tentar_exhausted == true"),
              "O loop de ID “tentar” acabou por atingir o limite de voltas?",
            ],
          ]}
        />
        <p className="muted">
          O editor avisa na hora se a expressão não puder ser lida (“Validação”, código “expressão
          inválida”, com a posição aproximada do problema).
        </p>
      </>
    ),
  },
  {
    id: "variaveis",
    title: "De onde vêm as variáveis",
    body: (
      <>
        <p>
          As expressões leem <strong>variáveis</strong>: informações guardadas durante a execução.
          Elas vêm de três lugares.
        </p>
        <Fields
          rows={[
            [
              "Informadas ao iniciar a execução",
              <>
                Quem inicia a execução (a fábrica) entrega os dados de partida. No fluxo padrão:{" "}
                {c("fabrica.correcao")} (verdadeiro se é correção de PR) e {c("prd.concluido")}{" "}
                (verdadeiro se o PRD da tarefa já está concluído).
              </>,
            ],
            [
              "Resultados dos nós de trabalho",
              <>
                Quando um nó de trabalho devolve informações, elas ficam sob o{" "}
                <strong>ID do nó</strong>: o nó {c("skill-1")} que devolve “status” vira{" "}
                {c("skill_1.status")} (os símbolos do ID, como o traço, viram _). Além disso,{" "}
                {c("last")} guarda sempre o resultado do <strong>último</strong> nó de trabalho
                executado: {c("last.status")}.
              </>,
            ],
            [
              "Marcas do próprio fluxo",
              <>
                Todo Loop que bate no limite de voltas ainda com a condição verdadeira cria{" "}
                {c("loop_<id>_exhausted")} = verdadeiro (troque {c("<id>")} pelo ID do loop, com _
                no lugar de símbolos).
              </>,
            ],
          ]}
        />
        <p className="muted">
          Atenção: quais campos cada tipo de nó devolve depende de como ele foi construído. Hoje o
          caso mais claro é o nó <strong>Skill</strong>, que devolve as variáveis que a própria
          skill grava. Os executores reais de Agente, Script, Checks e Git ainda estão sendo ligados
          à execução ponta a ponta; por isso, evite basear decisões em campos que você não viu serem
          produzidos. Para decidir com segurança, use a <strong>saída de erro</strong> do nó (ela
          sempre existe) em vez de ler um campo.
        </p>
      </>
    ),
  },
  {
    id: "erro",
    title: "Tratando erros: a saída error",
    body: (
      <>
        <p>
          Todo nó de execução tem a saída <code>error</code>, além da <code>out</code>. É por ela
          que o fluxo sai quando o passo <strong>falha</strong> (comando que dá erro, teste que
          reprova, agente que não consegue). Ela é <strong>opcional</strong>, mas decide o destino
          da falha:
        </p>
        <ul>
          <li>
            <strong>Ligada</strong> a um Fim de falha: a execução termina como falha, de forma
            organizada.
          </li>
          <li>
            <strong>Ligada</strong> a um passo de correção (passando por um Loop limitado): o fluxo
            tenta consertar e repetir.
          </li>
          <li>
            <strong>Sem ligação</strong>: a execução inteira para como falha naquele ponto.
          </li>
        </ul>
        <p className="muted">
          No editor, saídas de erro (e a saída “false”) aparecem em <strong>vermelho</strong> e as
          setas que saem delas são tracejadas, para você enxergar o caminho ruim.
        </p>
      </>
    ),
  },
  {
    id: "padrao",
    title: "O fluxo padrão, nó por nó",
    body: (
      <>
        <p>
          Todo projeto novo recebe o <strong>Fluxo padrão</strong>, já montado e com os agentes
          preenchidos. Ler como ele funciona é o melhor exemplo de um fluxo real.
        </p>
        <ol>
          <li>
            <strong>inicio</strong> → <strong>entrada</strong> (Condição{" "}
            {c("fabrica.correcao == true")}): é correção de PR? Se <em>sim</em>, pula direto para o
            planejamento; se <em>não</em>, passa pelo portão do PRD.
          </li>
          <li>
            <strong>guard_prd</strong> (Condição {c("prd.concluido == true")}): o PRD (documento que
            descreve o que construir) está concluído? Se <em>sim</em>, segue; se <em>não</em>,
            termina em <strong>fim_pendente</strong> (falha: ainda falta o PRD).
          </li>
          <li>
            <strong>pronto_planejar</strong> (Convergência, modo <code>any</code>): junta os dois
            caminhos acima — chegando por qualquer um, segue.
          </li>
          <li>
            <strong>guard_planning</strong> (Agente <code>tech-lead</code>): planeja a feature (ou
            só o que mudou, numa correção).
          </li>
          <li>
            <strong>ramificar</strong> (Ramificação): divide o trabalho em duas trilhas.
            <ul>
              <li>
                Trilha <code>a</code> (frontend): <strong>guard_impl_front</strong> (Agente{" "}
                <code>implementador-frontend</code>) → <strong>guard_review_front</strong> (Agente{" "}
                <code>revisor-frontend</code>).
              </li>
              <li>
                Trilha <code>b</code> (backend): <strong>guard_impl_back</strong> (Agente{" "}
                <code>implementador-backend</code>) → <strong>guard_review_back</strong> (Agente{" "}
                <code>revisor-backend</code>).
              </li>
            </ul>
          </li>
          <li>
            <strong>juntar_trilhas</strong> (Convergência, modo <code>all</code>): espera{" "}
            <strong>as duas</strong> trilhas terminarem.
          </li>
          <li>
            <strong>guard_consolidation</strong> (Git <code>commit</code>): consolida o trabalho num
            commit.
          </li>
          <li>
            <strong>guard_integrated_checks</strong> (Testes e checks, perfil <code>base</code>,
            comandos <code>make lint</code> e <code>make test</code>): verifica o conjunto.
          </li>
          <li>
            <strong>guard_draft_pr</strong> (Git <code>draft_pr</code>): abre a Pull Request em
            rascunho.
          </li>
          <li>
            <strong>guard_merge_wait</strong> (Espera durável, 1440 min = 1 dia): aguarda a pessoa
            revisar e integrar a PR.
          </li>
          <li>
            <strong>fim_ok</strong> (Fim, sucesso). Qualquer passo de trabalho que falhe segue a
            saída <code>error</code> até <strong>fim_falha</strong> (Fim, falha).
          </li>
        </ol>
        <h4>As “guardas” (nós com nome guard_…)</h4>
        <p>
          Os nós cujo ID começa com <code>guard_</code> são <strong>obrigatórios</strong>: protegem
          a qualidade da entrega (PRD, planejamento, revisões, checks, PR, espera pelo merge). Você
          pode <strong>personalizar</strong> o fluxo — mudar instruções, agentes, comandos,
          acrescentar passos —, mas o editor <strong>não deixa</strong> remover uma guarda, trocar o
          tipo dela nem criar um caminho até o sucesso que a contorne. Se tentar, aparecem os avisos
          “guarda removida”, “guarda de tipo errado” ou “caminho que contorna a guarda”.
        </p>
      </>
    ),
  },
  {
    id: "receitas",
    title: "Receitas: situações comuns e como montar",
    body: (
      <>
        <h4>1. Rodar duas coisas ao mesmo tempo e esperar as duas</h4>
        <p>
          Início → <strong>Ramificação</strong>. Ligue <code>a</code> ao primeiro trabalho e{" "}
          <code>b</code> ao segundo. Ligue o <code>out</code> de cada trabalho à entrada de uma{" "}
          <strong>Convergência</strong> em modo <code>all</code>. Siga a partir dela.
        </p>

        <h4>2. Fazer algo só em certa situação</h4>
        <p>
          Coloque uma <strong>Condição</strong> com a pergunta (ex.: {c("fabrica.correcao == true")}
          ). Ligue <code>true</code> ao trecho “só nesse caso” e <code>false</code> ao outro. Se os
          dois caminhos se encontram depois, junte-os com uma Convergência em modo <code>any</code>.
        </p>

        <h4>3. Tentar de novo até 3 vezes quando falhar</h4>
        <ol>
          <li>
            Passo que pode falhar (ex.: <em>Testes e checks</em>). Sua saída <code>out</code> vai ao
            Fim de sucesso.
          </li>
          <li>
            A saída <code>error</code> dele vai à entrada de um <strong>Loop limitado</strong> com
            máximo 3 e condição {c("true")}.
          </li>
          <li>
            A saída <code>body</code> do Loop vai a um <strong>Agente</strong> que corrige o
            problema; o <code>out</code> do agente <strong>volta</strong> à entrada do passo que
            falha (assim o ciclo passa pelo Loop, que é permitido).
          </li>
          <li>
            A saída <code>done</code> do Loop vai a um Fim de <strong>falha</strong> (esgotou as
            tentativas).
          </li>
        </ol>

        <h4>4. Pedir aprovação de uma pessoa antes de um passo importante</h4>
        <p>
          Coloque uma <strong>Decisão humana</strong> antes do passo (ex.: antes do Git{" "}
          <code>push</code>). Ligue <code>approved</code> ao passo e <code>rejected</code> a um Fim
          de falha (ou a um passo de ajuste).
        </p>

        <h4>5. Esperar um tempo antes de seguir</h4>
        <p>
          Use a <strong>Espera durável</strong> (ex.: 1440 minutos para esperar um dia).
        </p>

        <h4>6. Reaproveitar um trecho em vários fluxos</h4>
        <p>
          Crie um workflow do tipo <strong>Subfluxo</strong> com o trecho (começa em Início, termina
          em Fim). Nos fluxos que precisam dele, use um nó <strong>Subfluxo</strong> e informe o ID.
        </p>

        <h4>7. Terminar de forma organizada quando algo dá errado</h4>
        <p>
          Ligue a saída <code>error</code> de cada nó de trabalho a um mesmo <strong>Fim</strong>{" "}
          com resultado <code>failure</code>. Um só Fim de falha pode receber várias setas.
        </p>
      </>
    ),
  },
  {
    id: "problemas",
    title: "Mensagens de validação: o que significam e como resolver",
    body: (
      <Fields
        rows={[
          [
            "O campo “X” é obrigatório",
            "Falta preencher um campo marcado com *. Clique no nó (botão Localizar) e preencha.",
          ],
          [
            "A saída “X” não está conectada",
            "Uma saída (exceto error) está solta. Ligue-a a outro nó. Em Condição, Decisão humana e Ramificação, as duas saídas precisam de destino.",
          ],
          [
            "A entrada “in” não está conectada",
            "Nada leva a esse nó. Ligue a saída de outro nó a ele (ou apague o nó se for sobra).",
          ],
          [
            "Este nó não é alcançado a partir do início",
            "Há um nó sem caminho vindo do Início. Ligue-o ao fluxo ou apague-o.",
          ],
          [
            "A saída “X” só pode ter uma conexão",
            "Uma saída tem duas setas. Mantenha uma; para dividir use Condição (escolhe) ou Ramificação (vai pelos dois).",
          ],
          [
            "Esta entrada aceita uma única conexão",
            "Duas setas chegam na mesma entrada. Junte-as antes com uma Convergência.",
          ],
          [
            "Há um ciclo que não passa por um nó de loop com limite",
            "O fluxo volta a um passo anterior sem passar por um Loop limitado. Coloque um Loop no caminho de volta.",
          ],
          [
            "Expressão inválida",
            "A condição não pôde ser lida. Confira aspas fechadas, operadores válidos (== != < <= > >= && || !), ausência de parênteses e de contas.",
          ],
          [
            "O fluxo precisa de exatamente um nó de início / ao menos um de fim",
            "Mantenha um Início só e pelo menos um Fim.",
          ],
          [
            "O campo “X” deve ser um de: …",
            "O campo aceita só as opções da lista (ex.: Resultado: success ou failure). Escolha na lista.",
          ],
          [
            "O campo “X” deve ser um inteiro entre N e M",
            "Informe um número inteiro dentro do intervalo indicado.",
          ],
          [
            "A conta deve ser identificada por um ID de conexão válido / O subfluxo deve ser identificado por um ID válido",
            "O campo espera um identificador (UUID) copiado da tela Conexões ou do endereço do workflow, não um nome.",
          ],
          [
            "Guarda obrigatória removida / de tipo errado / caminho que contorna a guarda",
            "Só no fluxo de um projeto: restaure o nó guard_… com o tipo original e garanta que o caminho até o sucesso passe por ele.",
          ],
          [
            "Conexão duplicada / Um nó não pode se ligar a si mesmo",
            "Apague a seta repetida ou a que sai e volta para o mesmo nó.",
          ],
        ]}
      />
    ),
  },
  {
    id: "glossario",
    title: "Glossário",
    body: (
      <Fields
        rows={[
          [
            "Agente",
            "Uma inteligência artificial configurada para um papel (planejar, programar, revisar…).",
          ],
          [
            "Skill",
            "Um procedimento pronto e reutilizável (um “passo a passo” que o agente ou o fluxo executa).",
          ],
          ["PRD", "Documento que descreve o que será construído e por quê, antes de programar."],
          [
            "Pull Request (PR)",
            "Pedido para integrar as alterações de uma branch ao código principal. Em rascunho (draft) ainda não pode ser integrada.",
          ],
          [
            "Branch",
            "Uma “linha” paralela do código onde o trabalho é feito sem mexer na principal.",
          ],
          ["Commit", "Um ponto gravado do histórico do código, com uma mensagem."],
          ["Merge", "Integrar uma PR ao código principal. Quem decide é uma pessoa."],
          [
            "Sandbox",
            "Caixa isolada onde comandos rodam sem acesso ao seu computador nem à internet.",
          ],
          [
            "Guarda",
            "Nó obrigatório do fluxo padrão que protege a qualidade e não pode ser removido.",
          ],
          ["Publicar", "Congelar uma cópia do rascunho para valer nas execuções."],
          ["Rascunho", "A versão editável do fluxo; nunca executa."],
          ["Expressão", "Pergunta curta de verdadeiro/falso escrita na linguagem de condições."],
          [
            "Variável",
            "Informação guardada durante a execução (ex.: prd.concluido) que as expressões leem.",
          ],
        ]}
      />
    ),
  },
];

export function WorkflowManualButton() {
  return (
    <ManualDialog
      buttonLabel="Manual de workflows"
      label="Manual operacional"
      title="Como montar um workflow"
      sections={sections}
    />
  );
}
