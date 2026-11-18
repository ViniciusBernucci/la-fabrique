// Comando egressproxy: proxy de saída dos terminais dos agentes (ADR-0069). Só deixa passar os destinos de
// EGRESS_ALLOW (lista separada por vírgulas; ".dominio.com" inclui subdomínios).
package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/ViniciusBernucci/le-fabrique/internal/egress"
)

func main() {
	allow := egress.ParseList(os.Getenv("EGRESS_ALLOW"))
	if len(allow) == 0 {
		fmt.Fprintln(os.Stderr, "egressproxy: defina EGRESS_ALLOW (destinos permitidos)")
		os.Exit(2)
	}
	addr := os.Getenv("EGRESS_ADDR")
	if addr == "" {
		addr = ":3128"
	}
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil)).With("service", "egressproxy")
	p := &egress.Proxy{Allow: allow, Ports: egress.ParseList(os.Getenv("EGRESS_PORTS")), Log: log}
	log.Info("proxy de saída iniciado", "addr", addr, "regras", p.Describe())
	srv := &http.Server{Addr: addr, Handler: p, ReadHeaderTimeout: 10 * time.Second}
	if err := srv.ListenAndServe(); err != nil {
		log.Error("proxy encerrou", "error", err.Error())
		os.Exit(1)
	}
}
