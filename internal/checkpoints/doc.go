// Package checkpoints captura e restaura o estado de uma execução: o conteúdo do
// workspace (arquivos, links e modos, com hash de cada um e do conjunto), a
// informação de Git quando há (SHA, ramo, patch e não rastreados), o estado
// serializado dos nós, as referências externas e um manifesto.
//
// Garantias:
//   - só captura com o sandbox PARADO (sem processo vivo), para não gravar um
//     estado no meio de uma escrita;
//   - a gravação é atômica (um checkpoint e todos os seus objetos numa única
//     transação): interromper no meio deixa o checkpoint anterior intacto;
//   - toda leitura reconfere os hashes (objeto, manifesto e estado) e a
//     restauração reconfere a árvore no container novo;
//   - os recursos só são liberados depois que o checkpoint está durável.
package checkpoints
