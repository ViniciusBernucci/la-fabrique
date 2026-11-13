package restorecheck

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"sort"
	"strconv"

	"github.com/ViniciusBernucci/le-fabrique/db/migrations"
	"github.com/ViniciusBernucci/le-fabrique/internal/checkpoints"
	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
	"github.com/ViniciusBernucci/le-fabrique/internal/security"
)

// Check é o resultado de uma conferência.
type Check struct {
	Name     string   `json:"name"`
	OK       bool     `json:"ok"`
	Count    int      `json:"count"`
	Problems []string `json:"problems"`
	Info     string   `json:"info,omitempty"`
}

// Report reúne as conferências.
type Report struct {
	Checks []Check `json:"checks"`
}

// OK informa se todas as conferências passaram.
func (r Report) OK() bool {
	for _, c := range r.Checks {
		if !c.OK {
			return false
		}
	}
	return len(r.Checks) > 0
}

// Verifier confere o banco restaurado.
type Verifier struct {
	DB          persistence.DBTX
	Sealer      *security.Sealer
	Vault       *security.Vault
	Checkpoints *checkpoints.Store
}

const maxProblems = 20

func hash(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }

type acc struct{ c Check }

func newAcc(name string) *acc { return &acc{c: Check{Name: name, Problems: []string{}}} }
func (a *acc) bad(format string, args ...any) {
	if len(a.c.Problems) < maxProblems {
		a.c.Problems = append(a.c.Problems, fmt.Sprintf(format, args...))
	} else if len(a.c.Problems) == maxProblems {
		a.c.Problems = append(a.c.Problems, "… mais problemas omitidos")
	}
	a.c.OK = false
}
func (a *acc) done() Check { a.c.OK = len(a.c.Problems) == 0; return a.c }

// Run executa todas as conferências (uma falha de leitura vira problema, não aborta as demais).
func (v *Verifier) Run(ctx context.Context) Report {
	return Report{Checks: []Check{
		v.migrationsCheck(ctx), v.adminCheck(ctx), v.sessionsCheck(ctx), v.connectionsCheck(ctx),
		v.checkpointsCheck(ctx), v.skillsCheck(ctx), v.contextCheck(ctx),
	}}
}

var migFile = regexp.MustCompile(`^(\d+)_.*\.sql$`)

// ExpectedMigration é a maior versão embutida no binário.
func ExpectedMigration() int64 {
	ents, _ := migrations.FS.ReadDir(".")
	var max int64
	for _, e := range ents {
		if m := migFile.FindStringSubmatch(e.Name()); m != nil {
			n, _ := strconv.ParseInt(m[1], 10, 64)
			if n > max {
				max = n
			}
		}
	}
	return max
}

func (v *Verifier) migrationsCheck(ctx context.Context) Check {
	a := newAcc("migrations")
	var applied int64
	var n int
	if err := v.DB.QueryRow(ctx, `SELECT COALESCE(max(version_id), 0), count(*) FROM goose_db_version WHERE is_applied AND version_id > 0`).Scan(&applied, &n); err != nil {
		a.bad("não foi possível ler goose_db_version: %v", err)
		return a.done()
	}
	a.c.Count = n
	if want := ExpectedMigration(); applied != want {
		a.bad("migration aplicada %d, esperada %d (backup de outra versão do código)", applied, want)
	}
	return a.done()
}

func (v *Verifier) adminCheck(ctx context.Context) Check {
	a := newAcc("administrador e TOTP")
	rows, err := v.DB.Query(ctx, `SELECT id, totp_secret, totp_confirmed_at IS NOT NULL FROM admins`)
	if err != nil {
		a.bad("ler administradores: %v", err)
		return a.done()
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		var sealed []byte
		var confirmed bool
		if err := rows.Scan(&id, &sealed, &confirmed); err != nil {
			a.bad("ler administrador: %v", err)
			continue
		}
		a.c.Count++
		if !confirmed {
			continue
		}
		if _, err := v.Sealer.Open(sealed, security.AdminTOTPAAD(id)); err != nil {
			a.bad("o segredo TOTP do administrador %s não abre com a chave informada", id)
		}
	}
	if a.c.Count == 0 {
		a.bad("nenhum administrador no banco restaurado")
	}
	return a.done()
}

func (v *Verifier) sessionsCheck(ctx context.Context) Check {
	a := newAcc("sessões")
	var total, active, orphan int
	err := v.DB.QueryRow(ctx, `SELECT count(*), count(*) FILTER (WHERE revoked_at IS NULL AND expires_at > now()),
		count(*) FILTER (WHERE admin_id NOT IN (SELECT id FROM admins)) FROM admin_sessions`).Scan(&total, &active, &orphan)
	if err != nil {
		a.bad("ler sessões: %v", err)
		return a.done()
	}
	a.c.Count = total
	a.c.Info = fmt.Sprintf("%d ativas no momento da conferência (sessões vencidas após a restauração são esperadas)", active)
	if orphan > 0 {
		a.bad("%d sessões apontam para administrador inexistente", orphan)
	}
	return a.done()
}

func (v *Verifier) connectionsCheck(ctx context.Context) Check {
	a := newAcc("conexões do cofre")
	rows, err := v.DB.Query(ctx, `SELECT id, name FROM connections WHERE state = 'active' ORDER BY name`)
	if err != nil {
		a.bad("ler conexões: %v", err)
		return a.done()
	}
	type conn struct{ id, name string }
	var cs []conn
	for rows.Next() {
		var c conn
		if err := rows.Scan(&c.id, &c.name); err == nil {
			cs = append(cs, c)
		}
	}
	rows.Close()
	for _, c := range cs {
		a.c.Count++
		if _, _, err := v.Vault.Resolve(ctx, c.id); err != nil {
			a.bad("a conexão %q (%s) não pôde ser decifrada: VAULT_KEY incorreta ou dado corrompido", c.name, c.id)
		}
	}
	return a.done()
}

func (v *Verifier) checkpointsCheck(ctx context.Context) Check {
	a := newAcc("checkpoints e artefatos")
	rows, err := v.DB.Query(ctx, `SELECT project_id, id FROM checkpoints ORDER BY created_at, id`)
	if err != nil {
		a.bad("ler checkpoints: %v", err)
		return a.done()
	}
	type ref struct{ project, id string }
	var refs []ref
	for rows.Next() {
		var r ref
		if err := rows.Scan(&r.project, &r.id); err == nil {
			refs = append(refs, r)
		}
	}
	rows.Close()
	objects := 0
	for _, r := range refs {
		a.c.Count++
		cp, err := v.Checkpoints.Get(ctx, r.project, r.id)
		if err != nil {
			a.bad("checkpoint %s: %v", r.id, err)
			continue
		}
		seen := map[string]bool{}
		for _, f := range cp.Manifest.Files {
			if f.Kind != checkpoints.KindFile || seen[f.SHA256] {
				continue
			}
			seen[f.SHA256] = true
			objects++
			if _, err := v.Checkpoints.Object(ctx, r.project, f.SHA256); err != nil {
				a.bad("checkpoint %s, arquivo %q: %v", r.id, f.Path, err)
			}
		}
	}
	a.c.Info = fmt.Sprintf("%d objetos conferidos por hash", objects)
	return a.done()
}

func (v *Verifier) skillsCheck(ctx context.Context) Check {
	a := newAcc("arquivos de skills")
	rows, err := v.DB.Query(ctx, `SELECT s.name, f.path, f.sha256, f.content FROM skill_files f JOIN skills s ON s.id = f.skill_id ORDER BY s.name, f.path`)
	if err != nil {
		a.bad("ler skills: %v", err)
		return a.done()
	}
	defer rows.Close()
	for rows.Next() {
		var name, path, sum string
		var content []byte
		if err := rows.Scan(&name, &path, &sum, &content); err != nil {
			a.bad("ler arquivo de skill: %v", err)
			continue
		}
		a.c.Count++
		if hash(content) != sum {
			a.bad("skill %q, arquivo %q: o hash não confere", name, path)
		}
	}
	return a.done()
}

func (v *Verifier) contextCheck(ctx context.Context) Check {
	a := newAcc("pacotes de contexto")
	rows, err := v.DB.Query(ctx, `SELECT id, content, content_hash FROM context_packs ORDER BY created_at, id`)
	if err != nil {
		a.bad("ler pacotes de contexto: %v", err)
		return a.done()
	}
	defer rows.Close()
	for rows.Next() {
		var id, content, sum string
		if err := rows.Scan(&id, &content, &sum); err != nil {
			a.bad("ler pacote: %v", err)
			continue
		}
		a.c.Count++
		if hash([]byte(content)) != sum {
			a.bad("pacote %s: o hash do conteúdo não confere", id)
		}
	}
	return a.done()
}

// Names lista os nomes das conferências (para a saída humana).
func (r Report) Names() []string {
	n := make([]string, 0, len(r.Checks))
	for _, c := range r.Checks {
		n = append(n, c.Name)
	}
	sort.Strings(n)
	return n
}
