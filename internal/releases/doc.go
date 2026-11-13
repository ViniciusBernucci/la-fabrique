// Package releases publica e congela workflows. Publicar valida o grafo e todas as
// referências (agentes, contas, modelos, skills, ferramentas, subfluxos
// publicados), expande os subfluxos fixando a versão de cada um, resolve a
// configuração efetiva de cada nó e grava uma versão IMUTÁVEL com um snapshot
// sanitizado e o hash do conteúdo. Rascunho não executa: uma execução só nasce
// de uma versão publicada, depois de uma revalidação em tempo de execução que
// respeita revogação (conta revogada, modelo desativado), e fica presa àquela
// versão: publicar outra depois não muda a execução existente.
package releases
