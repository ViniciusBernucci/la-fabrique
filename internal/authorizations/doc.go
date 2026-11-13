// Package authorizations controla o que uma aprovação humana realmente autoriza.
//
// Uma autorização vale para UMA ação canônica — ferramenta, operação, recurso,
// destino e hash dos parâmetros — com prazo e número de usos. Qualquer
// divergência (argumento, destino, recurso, ferramenta), prazo vencido, usos
// esgotados ou revogação impede a operação, e a recusa fica na trilha. O consumo
// é atômico (UPDATE condicional): na corrida pelo último uso, um só vence.
// Revogar impede novos usos e, havendo uso em andamento, dispara a parada
// segura (cancelar, confirmar quiescência e gravar checkpoint) por um Halter.
package authorizations
