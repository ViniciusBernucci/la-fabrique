package integration

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/ViniciusBernucci/le-fabrique/internal/contextpack"
	"github.com/ViniciusBernucci/le-fabrique/internal/restorecheck"
	"github.com/ViniciusBernucci/le-fabrique/internal/security"
)

func rcVerifier(t *testing.T, c cxEnv, key []byte) *restorecheck.Verifier {
	t.Helper()
	sealer, err := security.NewSealer(key, c.e.cfg.VaultKeyID)
	if err != nil {
		t.Fatal(err)
	}
	pool := c.e.db.Pool
	return &restorecheck.Verifier{DB: pool, Sealer: sealer, Vault: security.NewVault(pool, sealer, security.NewRedactor()), Checkpoints: c.store}
}

func rcProblems(r restorecheck.Report) map[string][]string {
	m := map[string][]string{}
	for _, c := range r.Checks {
		if !c.OK {
			m[c.Name] = c.Problems
		}
	}
	return m
}

// T02-04 — "conferir sessão/configuração, capacidade de desencriptar conexões, banco e hashes de artefatos."
func TestRestoredDatabaseIsVerifiedForKeysConnectionsAndArtifactHashes(t *testing.T) {
	c := newCxEnv(t, "rc")
	ctx := context.Background()
	pool := c.e.db.Pool
	connWith(t, c.cl, "api", "rc-gateway", "openrouter")
	cp := c.checkpoint(t, "ponto", "package main // artefato")
	p := c.pack(t)
	if _, err := c.renew(t, cp.ID, p, contextpack.MilestonePlanning, "m", "", false); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		WITH s AS (INSERT INTO skills (name, executor, source_kind, content_hash, manifest) VALUES ('rc-skill', 'ai', 'manual', repeat('a', 64), '{}'::jsonb) RETURNING id)
		INSERT INTO skill_files (skill_id, path, sha256, size, content)
		SELECT id, 'SKILL.md', encode(sha256('conteudo'::bytea), 'hex'), 8, 'conteudo'::bytea FROM s`); err != nil {
		t.Fatal(err)
	}

	v := rcVerifier(t, c, c.e.cfg.VaultKey)
	rep := v.Run(ctx)
	if !rep.OK() {
		t.Fatalf("banco íntegro deveria passar: %+v", rcProblems(rep))
	}
	counts := map[string]int{}
	for _, ck := range rep.Checks {
		counts[ck.Name] = ck.Count
	}
	if counts["conexões do cofre"] != 1 || counts["checkpoints e artefatos"] != 1 || counts["arquivos de skills"] != 1 || counts["pacotes de contexto"] != 1 || counts["administrador e TOTP"] < 1 {
		t.Fatalf("contagens: %v", counts)
	}
	if blob, _ := json.Marshal(rep); strings.Contains(string(blob), "marcador-rc-gateway") {
		t.Fatal("o relatório não pode conter segredo")
	}

	// Chave errada: o cofre e o TOTP não abrem, sem vazar nada.
	wrong := append([]byte{}, c.e.cfg.VaultKey...)
	wrong[0] ^= 0xFF
	bad := rcProblems(rcVerifier(t, c, wrong).Run(ctx))
	if len(bad["conexões do cofre"]) != 1 || len(bad["administrador e TOTP"]) == 0 || len(bad) != 2 {
		t.Fatalf("chave errada: %v", bad)
	}
	if blob, _ := json.Marshal(bad); strings.Contains(string(blob), "marcador-") {
		t.Fatal("problema não pode conter segredo")
	}

	// Adulterações: cada uma é apontada na sua conferência.
	tamper := map[string]string{
		"checkpoints e artefatos": `UPDATE checkpoint_objects SET data = convert_to(repeat('x', size), 'UTF8')`,
		"arquivos de skills":      `UPDATE skill_files SET content = 'adulterad'::bytea, size = 9`,
		"pacotes de contexto":     `UPDATE context_packs SET content = content || 'x'`,
		"migrations":              `DELETE FROM goose_db_version WHERE version_id = (SELECT max(version_id) FROM goose_db_version)`,
	}
	for name, stmt := range tamper {
		if _, err := pool.Exec(ctx, stmt); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
	}
	got := rcProblems(v.Run(ctx))
	for name := range tamper {
		if len(got[name]) == 0 {
			t.Errorf("%s: adulteração não detectada (%v)", name, got)
		}
	}
	if len(got) != len(tamper) {
		t.Fatalf("só as conferências adulteradas devem falhar: %v", got)
	}
}

func TestRestoredDatabaseWithoutAdministratorFails(t *testing.T) {
	c := newCxEnv(t, "rc-empty")
	ctx := context.Background()
	if _, err := c.e.db.Pool.Exec(ctx, `DELETE FROM admins`); err != nil {
		t.Skipf("não foi possível esvaziar admins (FK): %v", err)
	}
	if got := rcProblems(rcVerifier(t, c, c.e.cfg.VaultKey).Run(ctx)); len(got["administrador e TOTP"]) == 0 {
		t.Fatalf("%v", got)
	}
}
