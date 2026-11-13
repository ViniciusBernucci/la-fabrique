// Package e2e orienta os testes de uma mudança e registra a evidência dos testes E2E,
// presa à revisão testada.
//
// Orientação: para cada arquivo de código alterado, o diff precisa trazer um teste criado ou
// atualizado (Go: um *_test.go no mesmo diretório; TS/TSX/JS: o .test/.spec do arquivo; Python:
// test_<nome>.py). A falta é listada; em risco ALTO ela bloqueia, a menos que haja justificativa.
//
// E2E por impacto: risco baixo não exige; médio exige quando a mudança toca a UI; alto exige
// sempre. A jornada roda num sandbox ISOLADO (sem rede, só do projeto — recusa um ambiente não
// isolado), pelo mesmo executor dos checks (T14-01), e a evidência guarda jornada, fixtures,
// navegador e as capturas (arquivos com hash). Evidência de outra revisão não vale.
//
// Caso em que o teste não se aplica (ou não há jornada testável) exige JUSTIFICATIVA e uma
// EVIDÊNCIA ALTERNATIVA que exista para a mesma revisão (um check aprovado, uma captura ou uma
// nota de revisão); sem as duas, o gate continua bloqueado.
package e2e
