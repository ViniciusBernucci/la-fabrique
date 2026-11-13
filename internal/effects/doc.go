// Package effects é o ledger de efeitos externos (T13-04): registra a intenção
// ANTES de enviar, identifica a operação por uma chave de negócio e, quando a
// resposta se perde (timeout, queda), reconcilia com uma prova verificável antes
// de repetir. Resultado da prova: confirmado (não duplica), ausente (repetir é
// seguro) ou inconclusivo (bloqueia a repetição automática e abre uma decisão
// humana com evidência).
package effects
