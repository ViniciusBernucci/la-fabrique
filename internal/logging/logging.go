// Package logging cria o logger estruturado (JSON) usado por todos os binários.
package logging

import (
	"io"
	"log/slog"
	"strings"
	"sync/atomic"
)

// textRedactor (opcional) redige valores de segredos conhecidos em qualquer
// texto logado, inclusive mensagem e erros. Definido pela API ao subir.
var textRedactor atomic.Pointer[func(string) string]

// SetRedactor instala a função que redige segredos conhecidos nos logs.
func SetRedactor(fn func(string) string) { textRedactor.Store(&fn) }

func redactText(s string) string {
	if fn := textRedactor.Load(); fn != nil {
		return (*fn)(s)
	}
	return s
}

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
			a.Value = a.Value.Resolve()
			switch a.Value.Kind() {
			case slog.KindString:
				return slog.String(a.Key, redactText(a.Value.String()))
			case slog.KindAny:
				if err, ok := a.Value.Any().(error); ok {
					return slog.String(a.Key, redactText(err.Error()))
				}
			}
			return a
		},
	})
	return slog.New(h).With("service", service)
}
