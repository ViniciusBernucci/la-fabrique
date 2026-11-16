import { type ReactNode, useEffect, useRef, useState } from "react";

type Section = { id: string; title: string; when?: string; body: ReactNode };

/** Pequena tabela "campo → o que significa", usada em todas as seções. */
function Fields({ rows }: { rows: [string, ReactNode][] }) {
  return (
    <dl className="manual-fields">
      {rows.map(([name, text]) => (
        <div key={name}>
          <dt>{name}</dt>
          <dd>{text}</dd>
        </div>
      ))}
    </dl>
  );
}

const sections: Section[] = [
  {
    id: "visao",
    title: "Visão geral: para que serve esta tela",
    body: (
      <>
        <p>
          Esta é a <strong>sala de controle de um projeto</strong>. Aqui você acompanha o trabalho
          da fábrica (tarefas, execuções dos agentes, Pull Requests), decide o que fazer quando algo
          precisa de uma pessoa e ajusta como o projeto se comporta.
        </p>
        <p>A tela é lida de cima para baixo, na ordem em que o trabalho acontece:</p>
        <ol>
          <li>
            <strong>Topo:</strong> estado do projeto e os botões principais.
          </li>
          <li>
            <strong>Execução:</strong> o que está rodando agora e os controles de pausa.
          </li>
          <li>
            <strong>Entregas:</strong> as Pull Requests que a fábrica abriu e o que falta para
            integrá-las.
          </li>
          <li>
            <strong>Memória técnica:</strong> o que a fábrica aprendeu sobre o projeto.
          </li>
          <li>
            <strong>Setup e Limpeza:</strong> preparo do ambiente e faxina das pastas de trabalho.
          </li>
          <li>
            <strong>Resumo, Configuração, Pendências e Histórico:</strong> a ficha do projeto.
          </li>
        </ol>
        <p className="muted">
          Nem tudo aparece o tempo todo: um projeto em <em>Rascunho</em> mostra só a ficha (Resumo,
          Configuração, Pendências e Histórico), porque ainda não começou a trabalhar. Cada seção
          abaixo diz quando ela aparece.
        </p>
      </>
    ),
  },
  {
    id: "topo",
    title: "O topo: estado e botões",
    body: (
      <>
        <p>
          O <strong>selo de estado</strong> ao lado do nome diz em que fase o projeto está:
        </p>
        <Fields
          rows={[
            [
              "Rascunho",
              "Cadastro ainda não concluído. A fábrica não faz nada neste projeto. Use Continuar cadastro.",
            ],
            [
              "Ativo",
              "Cadastro validado e ativado. A fábrica lê tarefas da origem e trabalha neste projeto.",
            ],
            [
              "Pausado",
              "Projeto suspenso por você. Nada novo começa. É o estado para editar repositório e origem.",
            ],
            ["Arquivado", "Projeto arquivado, fora de operação."],
          ]}
        />
        <p>Os botões mudam conforme o estado:</p>
        <Fields
          rows={[
            [
              "Continuar cadastro",
              "(Rascunho) Abre o assistente de cadastro na etapa em que você parou.",
            ],
            [
              "Suspender",
              "(Ativo) Pausa o projeto. Ele passa a Pausado e deixa de iniciar trabalho.",
            ],
            [
              "Editar configuração",
              "(Pausado) Reabre o assistente. Repositório e origem só podem mudar com o projeto pausado.",
            ],
            ["Retomar", "(Pausado) Volta o projeto para Ativo."],
            [
              "Apagar projeto",
              "(em qualquer estado, menos Ativo) Remove o projeto e sua configuração de forma permanente. Projetos com execuções registradas não podem ser apagados. Projeto Ativo precisa ser suspenso antes.",
            ],
          ]}
        />
        <p className="muted">
          Atenção ao vocabulário: <em>Suspender</em> (botão do topo) muda o estado do projeto para
          Pausado. Já <em>Pausar após a feature atual</em> e <em>Suspender agora</em>, da seção
          Execução, são controles de ritmo do trabalho e não mudam o estado do projeto.
        </p>
      </>
    ),
  },
  {
    id: "execucao",
    title: "Execução: o que está rodando agora",
    when: "Aparece em projetos que não são Rascunho. Atualiza sozinha a cada 10 segundos.",
    body: (
      <>
        <h4>Linhas de controle e botões</h4>
        <p>
          Se existir alguma pausa em vigor, uma linha avisa. <strong>Controle global</strong> vale
          para todos os projetos da fábrica; <strong>Controle do projeto</strong> vale só para este.
          Sem pausas, aparece “Controles operacionais: ativos”.
        </p>
        <Fields
          rows={[
            [
              "Motivo (opcional)",
              "Texto livre registrado junto com a ação (ex.: “deploy em andamento”). Ajuda quem olhar depois a entender por que parou.",
            ],
            [
              "Pausar após a feature atual",
              "Modo gentil: a feature em andamento termina normalmente e nenhuma feature nova começa. Use quando quiser parar sem perder trabalho.",
            ],
            [
              "Suspender agora",
              "Modo imediato: nada novo começa e o que estava rodando é parado com um checkpoint (um ponto de retomada). Se alguma tarefa não puder parar com segurança, a tela avisa quais continuam rodando e por quê.",
            ],
            ["Retomar projeto", "Aparece quando há pausa. Volta ao funcionamento normal."],
          ]}
        />
        <h4>Os quatro contadores</h4>
        <Fields
          rows={[
            ["Em execução", "Tarefas sendo trabalhadas por agentes neste momento."],
            ["Na fila", "Tarefas elegíveis esperando vaga para começar."],
            [
              "Aguardando",
              "Tarefas paradas esperando algo (uma decisão, uma liberação, uma dependência). A causa aparece como “Bloqueio” na lista.",
            ],
            ["Concluída", "Tarefas finalizadas."],
            [
              "Última sincronização",
              "Quando a fábrica leu a origem de tarefas (Jira ou ClickUp) pela última vez e se deu certo (“ok”) ou “falhou”. “Nunca” significa que ainda não leu.",
            ],
          ]}
        />
        <h4>Lista de tarefas</h4>
        <p>
          Cada linha mostra a chave da tarefa (ex.: <code>ABC-123</code>); <em>(correção de PR)</em>{" "}
          indica que é um ajuste pedido na revisão de uma Pull Request. “Bloqueio” diz por que não
          anda e “Tempo” mostra há quanto tempo está rodando. A etiqueta à direita é o estado: Na
          fila, Em execução, Aguardando, Concluída ou Cancelada.
        </p>
        <h4>Logs ao vivo, Monitor e Orientações</h4>
        <p>
          Quando existem execuções, escolha uma no seletor (“Tentativa N · estado · N evento(s)”).
          Três blocos aparecem:
        </p>
        <Fields
          rows={[
            [
              "Logs",
              "O diário técnico da execução, em tempo real. O texto de status mostra “Conectando…”, “Ao vivo” ou “Reconectando pelo cursor…” (a tela retoma de onde parou, sem perder linhas). A caixa Buscar pesquisa em todo o histórico; Voltar ao ao vivo retorna ao fluxo.",
            ],
            [
              "Monitor do agente",
              "Resumo vivo: se está em andamento ou concluída, tempo decorrido, quantas ferramentas usou, quantos erros, progresso (quando informado) e a lista de subagentes com o andamento de cada um. Abaixo, a saída do agente como num terminal.",
            ],
            [
              "Orientações ao agente",
              "Um canal para você mandar uma instrução a um agente em andamento. O estado de cada mensagem mostra só o que foi comprovado: na fila, enviada, confirmada pelo cliente, entrega incerta ou não entregue (a execução já terminou). Importante: orientação não altera contrato, permissões nem aprovações.",
            ],
          ]}
        />
      </>
    ),
  },
  {
    id: "entregas",
    title: "Entregas: Pull Requests e merge",
    when: "Aparece em projetos que não são Rascunho.",
    body: (
      <>
        <p>
          Uma <strong>entrega</strong> é o resultado de uma feature: a fábrica abre uma{" "}
          <strong>Draft PR</strong> (Pull Request em rascunho) no GitHub para você revisar. Esta
          seção acompanha cada uma. Vazia, mostra “Nenhuma entrega ainda”.
        </p>
        <h4>O que cada entrega mostra</h4>
        <Fields
          rows={[
            ["Título (chave · tentativa)", "A feature e qual tentativa gerou esta PR."],
            [
              "Selo de estado",
              "Bloqueada, Draft PR aberta, Integrada (merge feito), PR fechada sem merge ou Cancelada.",
            ],
            [
              "Link da PR e Merge",
              "Endereço da PR no GitHub e, depois do merge, os 12 primeiros caracteres do commit.",
            ],
            [
              "Origem: sem movimento / pendente / concluída / falhou",
              "Depois do merge, o que aconteceu com o item no Jira/ClickUp. “Falhou” pede nova consulta para reconciliar.",
            ],
            ["Bloqueio", "Motivo pelo qual a entrega não pode seguir."],
            [
              "Comentários",
              "Contagem dos comentários de revisão: na fila (vão virar correção), retido (esperando decisão sobre a PR), corrigido e substituído.",
            ],
            [
              "Última consulta",
              "Quando a fábrica olhou a PR no GitHub. A consulta é sob demanda: acontece quando você clica em Consultar PR.",
            ],
          ]}
        />
        <h4>Os botões</h4>
        <Fields
          rows={[
            [
              "Consultar PR",
              "Pergunta ao GitHub como a PR está. Resultados: segue aberta (nada muda); fechada sem merge (escolha uma ação); merge confirmado (a fila é liberada, o item de origem é movido se configurado e a memória é promovida); ou não foi possível confirmar (nada é alterado).",
            ],
            [
              "Detalhes",
              "Abre comentários da revisão, histórico de ações e o registro das consultas feitas à PR.",
            ],
            ["Histórico da feature", "Relatório em texto de tudo o que aconteceu na feature."],
            [
              "Tarefa que recebe a correção + Buscar comentários da revisão",
              "(Draft PR aberta) Informe a chave da tarefa que receberá os ajustes e a fábrica busca os comentários da revisão. Só valem comentários dos autores autorizados (veja Configuração da entrega); os demais são ignorados.",
            ],
          ]}
        />
        <h4>Quando a PR foi fechada sem merge</h4>
        <p>
          Em Detalhes aparece o aviso “A PR foi fechada sem merge. O que fazer?”, com as opções
          (como reabrir, cancelar ou abrir nova tentativa). Passe o mouse ou leia a descrição de
          cada uma. Você pode escrever uma <em>Observação</em> opcional. Essas ações pedem{" "}
          <strong>confirmação de identidade</strong> (senha e código), por serem decisões que mudam
          o rumo da entrega.
        </p>
      </>
    ),
  },
  {
    id: "config-entrega",
    title: "Configuração da entrega",
    when: "Aparece logo abaixo de Entregas.",
    body: (
      <>
        <Fields
          rows={[
            [
              "Autores autorizados a pedir correção",
              "Logins do GitHub, separados por vírgula. Só comentários de revisão feitos por essas pessoas viram correção. Vazio = nenhum comentário é aceito.",
            ],
            [
              "Estado “Aguardando merge” na origem",
              "O nome do estado, no seu Jira/ClickUp, que significa “PR aberta esperando integração”. Escreva exatamente como aparece no quadro.",
            ],
            [
              "Estado “Concluída” na origem",
              "O nome do estado final. Sem ele, o merge ainda libera a fila e promove a memória, mas o item da origem não é movido.",
            ],
          ]}
        />
        <p className="muted">Clique em Salvar configuração; “Configuração salva.” confirma.</p>
      </>
    ),
  },
  {
    id: "memoria",
    title: "Memória técnica",
    when: "Aparece em projetos que não são Rascunho.",
    body: (
      <>
        <p>
          A memória é o que a fábrica <strong>aprende e registra</strong> sobre o projeto (decisões,
          restrições, lições) para não repetir erros. Ela nasce <em>pendente</em>, presa à branch da
          feature, e só vira <em>oficial</em> depois do merge. Assim, trabalho que não foi integrado
          não contamina o conhecimento do projeto.
        </p>
        <Fields
          rows={[
            [
              "Abas Todas / Pendentes / Oficiais / Invalidadas",
              "Filtram os registros pelo estado.",
            ],
            ["Buscar na memória", "Pesquisa por texto nos registros."],
            ["Chave e conteúdo", "O nome do registro e o que ele afirma."],
            ["Tipo", "Fato, Decisão, Restrição ou Lição."],
            ["Confiança", "Baixa, média ou alta: o quanto a fábrica confia no registro."],
            [
              "Fonte e Evidências",
              "De onde veio e as provas que o sustentam (cada uma com sua referência).",
            ],
            [
              "Revisão",
              "Versão do código (12 primeiros caracteres do commit) em que o registro foi feito.",
            ],
            [
              "Branch / Promovida no merge",
              "Pendente: a branch de origem. Oficial: o merge em que foi promovida.",
            ],
            [
              "Invalidar",
              "Para um registro que ficou errado ou obsoleto. Exige um motivo (fica visível). Registro invalidado deixa de valer, mas continua listado.",
            ],
          ]}
        />
      </>
    ),
  },
  {
    id: "setup",
    title: "Setup do repositório",
    when: "Aparece quando o projeto tem uma pasta de repositório.",
    body: (
      <>
        <p>
          O <strong>setup</strong> é um script que prepara o ambiente antes de cada tentativa
          (instalar dependências, por exemplo). Ele fica no seu repositório, em{" "}
          <code>.lafabrique/setup.sh</code>, na branch base.
        </p>
        <Fields
          rows={[
            ["“Sem setup”", "O arquivo não existe. Para usar, versione-o na branch indicada."],
            [
              "Branch e hash",
              "A branch lida e a impressão digital (hash) do conteúdo. Se o arquivo mudar, o hash muda.",
            ],
            ["Conteúdo do script", "O texto exato que será executado. Leia antes de aprovar."],
            [
              "Aprovar este script",
              "Autoriza este conteúdo exato. Se alguém alterar o script depois, a aprovação perde a validade e é preciso aprovar de novo.",
            ],
            ["Revogar aprovação", "Cancela a autorização; o script deixa de rodar."],
            ["Última execução", "Resultado da última vez que rodou (status e código de saída)."],
          ]}
        />
        <p className="muted">
          Segurança: o script só roda dentro de um sandbox isolado, sem rede e sem segredos, e só
          depois da sua aprovação.
        </p>
      </>
    ),
  },
  {
    id: "limpeza",
    title: "Limpeza das worktrees",
    when: "Aparece quando o projeto tem uma pasta de repositório.",
    body: (
      <>
        <p>
          Cada tentativa de uma feature trabalha numa <strong>worktree</strong> (uma pasta de
          trabalho separada). Com o tempo elas se acumulam; a limpeza organiza isso.{" "}
          <strong>Vem desligada</strong> por padrão.
        </p>
        <Fields
          rows={[
            [
              "Limpeza ligada",
              "Liga ou desliga a política. Desligada, nada é arquivado nem removido.",
            ],
            [
              "Arquivar após (dias parada)",
              "Tentativa sem atividade por esse tempo é candidata a arquivar. Arquivar preserva tudo e retira só a pasta.",
            ],
            [
              "Remover arquivadas após (dias)",
              "Depois de arquivada por esse tempo, vira candidata a remover. Remover apaga o arquivo de trabalho já incorporado na branch base e é irreversível.",
            ],
            ["Salvar prazos", "Grava os dois números."],
            [
              "Simulação",
              "Lista o que seria feito, sem fazer nada: Arquivar, Remover ou Manter, com o motivo de cada branch.",
            ],
            [
              "Aplicar esta simulação (N)",
              "Executa exatamente a simulação que você está vendo, e só isso. Se algo mudar desde a simulação, o servidor recusa (“simule de novo”) e você revisa a nova lista. O resultado mostra quantas foram arquivadas, removidas e quantas falharam.",
            ],
          ]}
        />
        <p className="muted">Uma tentativa em execução nunca é tocada pela limpeza.</p>
      </>
    ),
  },
  {
    id: "resumo",
    title: "Resumo e Configuração (a ficha do projeto)",
    body: (
      <>
        <h4>Resumo</h4>
        <Fields
          rows={[
            [
              "Identificador",
              "Nome curto e fixo do projeto. Dá nome padrão à pasta e ao repositório.",
            ],
            [
              "Prioridade",
              "De 0 a 4. Maior = atendido primeiro quando vários projetos disputam a fábrica.",
            ],
            [
              "Versão",
              "Número que sobe a cada alteração. Protege contra duas pessoas editando ao mesmo tempo: quem tem uma versão velha é avisado em vez de sobrescrever.",
            ],
            ["Criado em / Atualizado em", "Datas do cadastro e da última alteração."],
          ]}
        />
        <h4>Configuração</h4>
        <Fields
          rows={[
            ["Repositório", "dono/nome no GitHub. “Não configurado” se ainda não foi escolhido."],
            [
              "Branch base",
              "A branch de onde o trabalho parte e para onde as PRs voltam (normalmente main).",
            ],
            [
              "Conexão GitHub",
              "Qual conta do GitHub a fábrica usa para clonar, enviar branches e abrir PRs.",
            ],
            [
              "Origem de tarefas",
              "Jira ou ClickUp e o projeto/lista de onde as tarefas são lidas.",
            ],
            [
              "Conexão da origem",
              "Qual conta do Jira/ClickUp a fábrica usa para ler (e escrever, se permitido) as tarefas.",
            ],
            [
              "Agentes, workflows e recursos",
              "O que está associado ao projeto. “Nenhum associado” significa que nada foi vinculado ainda.",
            ],
          ]}
        />
        <p className="muted">
          Se uma conexão aparecer como “Conexão indisponível”, ela foi revogada ou apagada: escolha
          outra em Editar configuração.
        </p>
      </>
    ),
  },
  {
    id: "pendencias",
    title: "Pendências e Histórico",
    body: (
      <>
        <Fields
          rows={[
            [
              "Pendências",
              "(Só enquanto não está Ativo.) O que ainda falta para ativar, cada item com a ação a tomar. “Nenhuma pendência de preenchimento” = pronto para ativar na etapa Revisão do cadastro.",
            ],
            [
              "Histórico",
              "O diário de tudo o que mudou na configuração: criação, dados atualizados, origem alterada ou verificada, repositório verificado, ativação, suspensão, retomada, setup aprovado ou revogado e executado, política de limpeza alterada e limpeza aplicada, cada um com data e hora.",
            ],
          ]}
        />
      </>
    ),
  },
  {
    id: "receitas",
    title: "Receitas: o que fazer quando…",
    body: (
      <dl className="manual-recipes">
        <dt>Quero que a fábrica pare, sem perder nada</dt>
        <dd>
          Execução → <em>Pausar após a feature atual</em>. Para voltar: <em>Retomar projeto</em>.
        </dd>
        <dt>Preciso parar tudo agora</dt>
        <dd>
          Execução → <em>Suspender agora</em> (informe o motivo). Confira se apareceu aviso de
          tarefas que não puderam parar.
        </dd>
        <dt>Preciso trocar o repositório ou a origem de tarefas</dt>
        <dd>
          Suspender (topo) → Editar configuração → altere → volte à Revisão, verifique repositório e
          origem → Retomar.
        </dd>
        <dt>Uma tarefa está “Aguardando” e não anda</dt>
        <dd>
          Veja o “Bloqueio” na lista de tarefas. Normalmente é uma decisão sua (Central de decisões)
          ou uma liberação pendente.
        </dd>
        <dt>A PR recebeu comentários de revisão</dt>
        <dd>
          Entregas → informe a tarefa que recebe a correção → <em>Buscar comentários da revisão</em>
          . Confira se o autor está em Autores autorizados.
        </dd>
        <dt>Fizeram o merge no GitHub</dt>
        <dd>
          Entregas → <em>Consultar PR</em>. Ao confirmar o merge, a fila é liberada, a memória é
          promovida e o item da origem é movido (se o estado “Concluída” estiver configurado).
        </dd>
        <dt>A PR foi fechada sem merge</dt>
        <dd>Entregas → Detalhes → escolha uma das opções (confirmando sua identidade).</dd>
        <dt>Um registro de memória está errado</dt>
        <dd>Memória técnica → Invalidar, com o motivo.</dd>
        <dt>Muitas pastas de trabalho acumuladas</dt>
        <dd>
          Limpeza → ligue, ajuste os prazos, revise a Simulação e use{" "}
          <em>Aplicar esta simulação</em>.
        </dd>
      </dl>
    ),
  },
  {
    id: "glossario",
    title: "Glossário rápido",
    body: (
      <Fields
        rows={[
          ["Feature", "Uma unidade de trabalho entregável, ligada a uma tarefa da origem."],
          [
            "Tentativa",
            "Cada rodada de execução de uma feature. Se falhar ou for reprovada, nasce a tentativa seguinte.",
          ],
          ["Execução", "A corrida de um agente numa tentativa, com logs, monitor e orientações."],
          ["Draft PR", "Pull Request em rascunho, aberta pela fábrica para você revisar."],
          ["Merge", "Integrar a PR na branch base. É o momento em que o trabalho vira oficial."],
          ["Origem", "O Jira ou ClickUp de onde as tarefas vêm."],
          [
            "Checkpoint",
            "Ponto de retomada salvo ao parar uma tarefa, para continuar depois sem recomeçar.",
          ],
          [
            "Worktree",
            "Pasta de trabalho separada de uma tentativa, para uma não interferir na outra.",
          ],
          ["Sandbox", "Ambiente isolado, sem rede e sem segredos, onde o setup roda."],
          [
            "Conexão",
            "Credencial guardada no cofre (Conexões). A tela só mostra o nome, nunca o segredo.",
          ],
        ]}
      />
    ),
  },
];

/** Botão do topo do painel + manual operacional em janela (índice e uma seção por parte da tela). */
export function ProjectManualButton() {
  const [open, setOpen] = useState(false);
  const closeRef = useRef<HTMLButtonElement>(null);
  const opener = useRef<HTMLButtonElement>(null);
  const bodyRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!open) return;
    closeRef.current?.focus();
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        setOpen(false);
        opener.current?.focus();
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [open]);

  function close() {
    setOpen(false);
    opener.current?.focus();
  }

  function jump(id: string) {
    bodyRef.current?.querySelector(`#manual-${id}`)?.scrollIntoView?.({ block: "start" });
  }

  return (
    <>
      <button
        ref={opener}
        type="button"
        className="secondary-action"
        aria-haspopup="dialog"
        onClick={() => setOpen(true)}
      >
        Como funciona este painel
      </button>
      {open && (
        <div className="dialog-backdrop">
          <div
            className="manual panel"
            role="dialog"
            aria-modal="true"
            aria-labelledby="manual-title"
          >
            <header className="manual-head">
              <div>
                <p className="page-label">Manual operacional</p>
                <h2 id="manual-title">Como funciona o painel do projeto</h2>
              </div>
              <button ref={closeRef} type="button" className="secondary-action" onClick={close}>
                Fechar
              </button>
            </header>
            <div className="manual-layout">
              <nav className="manual-toc" aria-label="Seções do manual">
                <ol>
                  {sections.map((s) => (
                    <li key={s.id}>
                      <button type="button" className="link-button" onClick={() => jump(s.id)}>
                        {s.title}
                      </button>
                    </li>
                  ))}
                </ol>
              </nav>
              <div className="manual-body" ref={bodyRef}>
                {sections.map((s) => (
                  <section key={s.id} id={`manual-${s.id}`} aria-labelledby={`manual-h-${s.id}`}>
                    <h3 id={`manual-h-${s.id}`}>{s.title}</h3>
                    {s.when && <p className="manual-when">{s.when}</p>}
                    {s.body}
                  </section>
                ))}
              </div>
            </div>
          </div>
        </div>
      )}
    </>
  );
}
