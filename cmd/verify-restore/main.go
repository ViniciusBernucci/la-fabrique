// Comando verify-restore: confere um banco restaurado (somente leitura).
// Uso: DATABASE_URL=<banco restaurado> VAULT_KEY=<chave> verify-restore
// Sai com 0 só se todas as conferências passarem.
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/ViniciusBernucci/le-fabrique/internal/checkpoints"
	"github.com/ViniciusBernucci/le-fabrique/internal/config"
	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
	"github.com/ViniciusBernucci/le-fabrique/internal/restorecheck"
	"github.com/ViniciusBernucci/le-fabrique/internal/security"
)

func main() { os.Exit(run()) }

func run() int {
	cfg, err := config.LoadAPI(os.LookupEnv)
	if err != nil {
		fmt.Fprintln(os.Stderr, "verify-restore: "+err.Error())
		return 2
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	pool, err := persistence.Connect(ctx, cfg.DatabaseURL, 10*time.Second)
	if err != nil {
		fmt.Fprintln(os.Stderr, "verify-restore: conectar ao banco restaurado: "+err.Error())
		return 1
	}
	defer pool.Close()
	sealer, err := security.NewSealer(cfg.VaultKey, cfg.VaultKeyID)
	if err != nil {
		fmt.Fprintln(os.Stderr, "verify-restore: chave do cofre inválida")
		return 2
	}
	red := security.NewRedactor()
	v := &restorecheck.Verifier{DB: pool, Sealer: sealer, Vault: security.NewVault(pool, sealer, red), Checkpoints: &checkpoints.Store{DB: pool}}
	rep := v.Run(ctx)
	for _, c := range rep.Checks {
		mark := "OK  "
		if !c.OK {
			mark = "FALHA"
		}
		fmt.Printf("%-5s %-28s %d item(ns)", mark, c.Name, c.Count)
		if c.Info != "" {
			fmt.Printf(" — %s", c.Info)
		}
		fmt.Println()
		for _, p := range c.Problems {
			fmt.Println("        - " + p)
		}
	}
	if !rep.OK() {
		fmt.Println("RESULTADO: restauração NÃO confiável")
		return 1
	}
	fmt.Println("RESULTADO: restauração íntegra")
	return 0
}
