// Package integrations reúne o que fala com sistemas externos:
//   - Tester: teste barato de credenciais do cofre (T04-03);
//   - Tracker: interface simétrica dos adaptadores Jira e ClickUp (T06-01/02);
//   - o adaptador GitHub completo chega no E15.
//
// Regras comuns: o segredo só existe dentro do adaptador, nenhuma mensagem de
// erro carrega corpo remoto, URL ou credencial, redirecionamentos não são
// seguidos e toda entrada que vira caminho ou consulta é validada antes de sair.
package integrations
