import type { ReactNode } from "react";
import type { WizardStep } from "../lib/api";

const guides: Record<WizardStep, { title: string; body: ReactNode }> = {
  basics: {
    title: "Como preencher: Dados",
    body: (
      <>
        <ol>
          <li>
            <strong>Nome:</strong> como o projeto aparece nas listas e na Central. Pode ser o nome
            do produto (ex.: <code>App de pedidos</code>).
          </li>
          <li>
            <strong>Identificador:</strong> definido ao criar o projeto e não muda depois. Serve de
            nome padrão da pasta e do repositório.
          </li>
          <li>
            <strong>Prioridade na fila:</strong> de 0 (mais baixa) a 4 (mais alta). Quando houver
            vários projetos com trabalho pendente, os de prioridade maior são atendidos primeiro.
          </li>
          <li>
            <strong>Descrição:</strong> opcional, até 500 caracteres. Se a fábrica criar o
            repositório no GitHub, ela é usada como descrição dele.
          </li>
          <li>
            Clique em <em>Salvar e continuar</em>.
          </li>
        </ol>
        <p className="muted">
          Só o nome é obrigatório. Tudo é salvo ao avançar, e você pode fechar a página e voltar
          depois na etapa em que parou.
        </p>
      </>
    ),
  },
  repository: {
    title: "Como preencher: Repositório",
    body: (
      <>
        <p>
          Antes de começar, a conta do GitHub precisa estar conectada em <strong>Conexões</strong> e
          com <q>Teste OK</q>. Escolha um dos três caminhos:
        </p>
        <ol>
          <li>
            <strong>Clonar do GitHub</strong> (o repositório já existe lá): escolha a{" "}
            <em>Conexão GitHub</em>, clique no repositório na lista (ou digite dono e nome) e
            confirme a <em>Branch base</em>, normalmente <code>main</code>. Em{" "}
            <em>Clonar e continuar</em>, a fábrica baixa o código para a área de workspaces. Se a
            lista não carregar, preencha dono e nome à mão.
          </li>
          <li>
            <strong>Pasta existente</strong> (o código já está no seu computador): navegue pelas
            pastas e use uma marcada como <em>repositório git</em>. A pasta precisa ter o remoto{" "}
            <code>origin</code> apontando para o repositório no GitHub. Branch base vazia usa a
            branch atual da pasta.
          </li>
          <li>
            <strong>Criar novo</strong> (projeto do zero): deixe a pasta vazia para usar o
            identificador do projeto. Marque <em>Criar também o repositório no GitHub</em> se quiser
            que a fábrica o crie na sua conta (ou numa organização, informando o dono), escolhendo
            privado ou público.
          </li>
        </ol>
        <p className="muted">
          Um mesmo repositório ou pasta não pode estar em dois projetos ativos. A conta conectada
          precisa ter permissão de escrita, porque a fábrica abre PRs. A prova disso é feita na
          Revisão, em <em>Verificar repositório</em>.
        </p>
      </>
    ),
  },
  source: {
    title: "Como preencher: Origem de tarefas",
    body: (
      <>
        <p>
          A origem é de onde a fábrica lê as tarefas. Cada projeto usa uma única: Jira ou ClickUp. A
          conexão precisa existir em <strong>Conexões</strong> (no Jira, há um passo a passo lá).
        </p>
        <ol>
          <li>
            <strong>Plataforma e Conexão:</strong> escolha Jira ou ClickUp e depois a conexão.
            Trocar a plataforma limpa a conexão e o identificador.
          </li>
          <li>
            <strong>Projeto ou lista:</strong> no Jira, a <em>chave</em> do projeto, as letras antes
            do número das tarefas (em <code>ABC-123</code>, é <code>ABC</code>). No ClickUp, o{" "}
            <em>ID da lista</em>: abra a lista, clique em <em>Copiar link</em> e use o número do
            final do endereço.
          </li>
          <li>
            <strong>Marcador de elegibilidade:</strong> um rótulo (Jira, sem espaços) ou tag
            (ClickUp), como <code>lf-ready</code>. Só tarefas com esse marcador entram na fila. Isso
            evita que a fábrica pegue o que você ainda não liberou.
          </li>
          <li>
            <strong>Estados elegíveis:</strong> nomes de estado separados por vírgula, exatamente
            como aparecem no quadro (ex.: <code>To Do, Pronto para dev</code>). É preciso definir o
            marcador, ao menos um estado, ou ambos.
          </li>
          <li>
            <strong>Mapa de prioridade:</strong> traduz a prioridade da origem para a da fábrica, no
            formato <code>nome=número</code> de 0 a 4 (ex.:{" "}
            <code>highest=0, high=1, medium=2, low=3</code>). Sem mapa, as tarefas ficam{" "}
            <q>sem mapa</q>.
          </li>
          <li>
            <strong>Escopo na origem:</strong> <em>Somente leitura</em> só lê as tarefas;{" "}
            <em>Leitura e escrita</em> também deixa a fábrica criar subtarefas e mudar estados.
            Comece por leitura se estiver testando.
          </li>
          <li>
            Clique em <em>Salvar e verificar origem</em>. Ela consulta a plataforma de verdade e
            mostra os itens elegíveis agora, com a prioridade traduzida. Se a lista vier vazia,
            confira o marcador, os estados e se há tarefas liberadas. Quando estiver certo, use{" "}
            <em>Salvar e continuar</em>.
          </li>
        </ol>
        <p className="muted">
          A mesma lista não pode alimentar dois projetos ativos. A verificação da origem é exigida
          para ativar.
        </p>
      </>
    ),
  },
  review: {
    title: "Como concluir: Revisão",
    body: (
      <>
        <ol>
          <li>
            Leia a lista de itens: cada <strong>!</strong> é uma pendência que impede a ativação,
            com o que fazer. <em>Ir para a etapa</em> leva direto ao ponto.
          </li>
          <li>
            Clique em <em>Verificar repositório</em>. A fábrica confere a pasta, o remoto{" "}
            <code>origin</code>, a branch base, a conta GitHub e a permissão de push. Se a conta for
            recusada, reconecte em Conexões.
          </li>
          <li>
            Se faltar a verificação da origem, volte à etapa Origem e use{" "}
            <em>Salvar e verificar origem</em>. Mudar repositório ou origem depois da verificação
            exige verificar de novo.
          </li>
          <li>
            Sem pendências, clique em <em>Ativar projeto</em>. Depois de ativo, repositório e origem
            ficam travados; para editar, suspenda o projeto na tela de detalhe.
          </li>
        </ol>
      </>
    ),
  },
};

/** Passo a passo da etapa atual do assistente. Só orientação: nenhum valor passa por aqui. */
export function SetupTutorial({ step }: { step: WizardStep }) {
  const g = guides[step];
  return (
    <details className="tutorial" key={step}>
      <summary>{g.title}</summary>
      {g.body}
    </details>
  );
}
