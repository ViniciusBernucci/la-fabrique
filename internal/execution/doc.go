// Package execution provisiona e supervisiona o ambiente em que o código dos
// projetos roda: sandbox efêmero por feature (T11-01) e, nos tickets seguintes,
// ferramentas com limites (T11-02) e supervisão de processos (T11-03).
//
// Regra central: o código do projeto roda DENTRO do sandbox; quem tem acesso ao
// runtime de containers é só o supervisor (este pacote), nunca o código do
// projeto, e nenhum segredo é passado por variável de ambiente ou argumento.
package execution
