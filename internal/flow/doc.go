// Package flow é o catálogo de status e a tabela de transições da adequação (LF-AJ-02).
//
// Cada status tem um ID interno estável e um código canônico. O alias legado
// REPROVED_PR converge para REPROVADO_PR na leitura; o valor bruto observado fica
// registrado para auditoria. As guardas (Check) são regras puras e testáveis sem
// integração: dado o estado atual, o destino e as evidências, dizem se a transição
// é válida e, se não for, por quê. Somente a Sentinela efetiva transições.
package flow
