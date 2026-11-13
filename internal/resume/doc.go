// Package resume retoma uma execução em um nó específico, só no conjunto afetado.
//
// Antes de retomar, valida (nesta ordem): o nó existe no grafo CONGELADO da
// execução e já foi percorrido; o último checkpoint está íntegro e é da
// execução; a versão publicada ainda é executável (permissões, contas, modelos);
// não há efeito externo incerto nos nós afetados (efeito confirmado é reaproveitado:
// o ledger não o repete); e os insumos (refs, requisito, configuração) não mudaram
// de forma conflitante desde o baseline.
//
// Mudança COMPATÍVEL (prioridade, rótulos, avanço de ref comprovado como
// fast-forward) é incorporada ao baseline e registrada. Mudança CONFLITANTE
// (texto do requisito, ref divergente, configuração congelada diferente) abre uma
// decisão e nada é alterado: estado, checkpoint e edições humanas ficam como estavam.
//
// A reexecução refaz SÓ o nó escolhido. Se a saída é igual, os dependentes
// continuam válidos e são reaproveitados; se mudou, só os descendentes (pelo
// grafo congelado) são invalidados e voltam a rodar. Nós fora desse conjunto
// ficam idênticos, e a execução segue presa à versão publicada original.
package resume
