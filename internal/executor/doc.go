// Package executor é o contrato único dos adaptadores de execução (LF-AJ-07): CLI (preferencial,
// com sessão nativa quando existe) e API (eventos estruturados) recebem o MESMO formato de tarefa,
// com contrato, instruções e skill, e devolvem os MESMOS eventos. Cada adaptador declara o que
// suporta (sessão nativa, retomada, interação, subagentes); a fábrica nunca presume uma capacidade
// que não foi declarada.
package executor
