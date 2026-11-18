/**
 * Passo a passo para conectar o Jira por token de API (formulário "Nova conexão por token").
 * Só orientação: nenhum valor passa por aqui.
 */
export function JiraTutorial() {
  return (
    <details className="tutorial" open aria-label="Como conectar o Jira">
      <summary>Como conectar o Jira (passo a passo)</summary>
      <ol>
        <li>
          <strong>Descubra o endereço do seu site Jira.</strong> Abra o Jira no navegador: o
          endereço tem a forma <code>https://&lt;site&gt;.atlassian.net</code>. Copie só essa parte,
          sem o que vem depois (<code>/jira/…</code>). Ela vai no campo <em>Destino</em>.
        </li>
        <li>
          <strong>Crie um token de API.</strong> Entre em{" "}
          <a
            href="https://id.atlassian.com/manage-profile/security/api-tokens"
            target="_blank"
            rel="noreferrer noopener"
          >
            id.atlassian.com/manage-profile/security/api-tokens
          </a>{" "}
          com a mesma conta que usa no Jira e clique em <em>Criar token de API</em> (
          <em>Create API token</em>). Dê um nome (ex.: <code>la-fabrique</code>), escolha a validade
          e confirme.
        </li>
        <li>
          <strong>Copie o token agora.</strong> A Atlassian mostra o valor uma única vez; se fechar
          a janela, será preciso criar outro. Não cole o token em chats, tickets ou arquivos.
        </li>
        <li>
          <strong>Preencha o formulário abaixo.</strong> <em>Tipo</em>: Jira · <em>Nome</em>: um
          nome para reconhecer a conta (ex.: <code>Jira da empresa</code>) · <em>Destino</em>: o
          endereço do passo 1 · <em>E-mail da Atlassian</em>: o e-mail da conta que criou o token ·{" "}
          <em>Segredo</em>: o token do passo 3.
        </li>
        <li>
          <strong>Guarde e teste.</strong> Clique em <em>Guardar no cofre</em> (pode pedir sua senha
          e o código TOTP) e, na lista, em <em>Testar</em>. O esperado é{" "}
          <q>Credencial aceita pelo Jira</q>.
        </li>
      </ol>
      <p className="muted">
        <strong>Se o teste falhar:</strong> “401” = e-mail ou token errados, expirados ou revogados
        (crie outro e use <em>Rotacionar</em>); “403” = a conta não tem permissão no site; “Destino
        Jira deve ser https://&lt;site&gt;.atlassian.net” = confira o passo 1.
      </p>
      <p className="muted">
        O token tem as mesmas permissões da conta que o criou, então use uma conta com acesso aos
        projetos que a fábrica vai ler e atualizar. Se o painel <em>Conectar com Jira</em> acima
        estiver disponível (OAuth configurado pelo administrador), ele dispensa o token: basta
        autorizar no navegador.
      </p>
    </details>
  );
}
