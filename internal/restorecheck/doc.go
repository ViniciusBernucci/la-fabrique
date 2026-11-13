// Package restorecheck confere um banco RESTAURADO de um backup, só com leitura:
// migrations completas, administrador e segredo TOTP decifráveis com a chave,
// sessões coerentes, conexões do cofre decifráveis, checkpoints com todos os
// hashes (manifesto, estado e objetos), arquivos de skills e pacotes de
// contexto com hash igual ao gravado.
//
// O relatório nunca contém segredo: só contagens, identificadores e o motivo
// de cada problema. Um backup só vale quando esta conferência passa.
package restorecheck
