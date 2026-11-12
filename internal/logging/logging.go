// Package logging cria o logger estruturado (JSON) usado por todos os binários.
package logging

import (
	"io"
	"log/slog"
	"strings"
)

// sensitiveKeys são atributos cujo valor é sempre redigido na origem.
var sensitiveKeys = []string{"password", "secret", "token", "authorization", "cookie", "database_url", "dsn"}

// New devolve um logger JSON com o nível e o serviço informados.
// Atributos com nomes sensíveis têm o valor substituído por [REDACTED].
func New(w io.Writer, level, service string) *slog.Logger {
	var lvl slog.Level
	switch strings.ToLower(level) {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}
	h := slog.NewJSONHandler(w, &slog.HandlerOptions{
		Level: lvl,
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			key := strings.ToLower(a.Key)
			for _, s := range sensitiveKeys {
				if strings.Contains(key, s) {
					return slog.String(a.Key, "[REDACTED]")
				}
			}
			return a
		},
	})
	return slog.New(h).With("service", service)
}
