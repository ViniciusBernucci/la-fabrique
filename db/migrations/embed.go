// Package migrations embute as migrations SQL versionadas da fábrica.
package migrations

import "embed"

// FS contém os arquivos NNNNN_nome.sql aplicados pelo cmd/migrate.
//
//go:embed *.sql
var FS embed.FS
