package integration

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/ViniciusBernucci/le-fabrique/internal/authorizations"
	"github.com/ViniciusBernucci/le-fabrique/internal/decisions"
)

type seqItem struct {
	At     time.Time      `json:"at"`
	Source string         `json:"source"`
	Type   string         `json:"type"`
	Detail map[string]any `json:"detail"`
}

func sequenceOf(t *testing.T, c *client, exec string) []seqItem {
	t.Helper()
	var out struct{ Items []seqItem }
	c.expect(c.get("/api/executions/"+exec+"/sequence"), 200).JSON(&out)
	return out.Items
}

func countType(items []seqItem, source, typ string) int {
	n := 0
	for _, i := range items {
		if i.Source == source && i.Type == typ {
			n++
		}
	}
	return n
}

func approvalRequest(project, execution, node string) decisions.NewRequest {
	r := newRequest(project, execution, node, decisions.KindSensitiveOperation)
	r.Title = "Publicar a branch da feature"
	r.Options = []decisions.Option{{ID: "approve", Label: "Aprovar", Description: "autoriza o push"}, {ID: "refuse", Label: "Recusar", Description: "não publica"}}
	return r
}

// T16-02 — "Aprovar, recusar ou revogar pelo painel produz evento e estado verificáveis sem notificação externa."
func TestApprovingRefusingAndRevokingFromThePanelLeaveVerifiableEventsAndState(t *testing.T) {
	e, c, _ := newProjectEnv(t)
	ctx := context.Background()
	p := mustProject(t, e.db.Pool, "central")
	var admin string
	_ = e.db.Pool.QueryRow(ctx, `SELECT id::text FROM admins LIMIT 1`).Scan(&admin)

	// Aprovar.
	exA := waitingExecution(t, e, p.ID, "C-1")
	reqA, _, _ := decisions.Create(ctx, e.db.Pool, nil, approvalRequest(p.ID, exA.ID, "git"))
	var got decisions.Request
	c.expect(c.post("/api/decisions/"+reqA.ID+"/resolve", map[string]any{"option": "approve", "note": "ok"}), 200).JSON(&got)
	if got.State != "resolved" || got.ResolutionOption != "approve" {
		t.Fatalf("%+v", got)
	}
	if state, _, ev := execRow(t, e, exA.ID); state != "queued" || ev != 2 {
		t.Fatalf("aprovar retoma só essa execução: %s %d", state, ev)
	}

	// Recusar: pela alternativa de recusa que o pedido declara.
	exR := waitingExecution(t, e, p.ID, "C-2")
	reqR, _, _ := decisions.Create(ctx, e.db.Pool, nil, approvalRequest(p.ID, exR.ID, "git"))
	c.expect(c.post("/api/decisions/"+reqR.ID+"/refuse", map[string]any{"note": "fora do escopo"}), 200).JSON(&got)
	if got.State != "resolved" || got.ResolutionOption != "refuse" || got.ResolutionNote != "fora do escopo" {
		t.Fatalf("%+v", got)
	}
	if r := c.post("/api/decisions/"+reqR.ID+"/refuse", map[string]any{}); r.StatusCode != 200 || r.Header.Get("X-Decision-Replayed") != "true" {
		t.Fatalf("repetir a recusa é idempotente: %d %s", r.StatusCode, r.Body)
	}
	// Pedido sem alternativa de recusa: a recusa não existe.
	exN := waitingExecution(t, e, p.ID, "C-3")
	reqN, _, _ := decisions.Create(ctx, e.db.Pool, nil, newRequest(p.ID, exN.ID, "dev", decisions.KindConflict))
	if r := c.post("/api/decisions/"+reqN.ID+"/refuse", map[string]any{}); r.StatusCode != 422 || r.ErrCode() != "no_refusal_option" {
		t.Fatalf("%d %s", r.StatusCode, r.Body)
	}
	if state, _, _ := execRow(t, e, exN.ID); state != "waiting" {
		t.Fatalf("a execução não muda: %s", state)
	}

	// Autorização derivada da aprovação: lista, validade e revogação.
	store := &authorizations.Store{DB: e.db.Pool}
	au, err := store.Create(ctx, authorizations.NewAuthorization{ProjectID: p.ID, ExecutionID: exA.ID, NodeID: "git", DecisionID: reqA.ID, Action: pushAction(), MaxUses: 2, TTL: time.Hour, GrantedBy: admin})
	if err != nil {
		t.Fatal(err)
	}
	var list struct {
		Items []struct {
			ID               string `json:"id"`
			Status           string `json:"status"`
			Remaining        int    `json:"remaining"`
			ExpiresInSeconds int64  `json:"expires_in_seconds"`
			RevokeReason     string `json:"revoke_reason"`
		}
	}
	c.expect(c.get("/api/authorizations?status=active"), 200).JSON(&list)
	if len(list.Items) != 1 || list.Items[0].ID != au.ID || list.Items[0].Remaining != 2 || list.Items[0].ExpiresInSeconds <= 0 {
		t.Fatalf("%+v", list)
	}
	if r := c.get("/api/authorizations?status=talvez"); r.StatusCode != 400 {
		t.Fatalf("%d", r.StatusCode)
	}
	// Sem reautenticação recente a revogação é recusada e nada muda.
	if _, err := e.db.Pool.Exec(ctx, `UPDATE admin_sessions SET reauth_at = NULL`); err != nil {
		t.Fatal(err)
	}
	if r := c.post("/api/authorizations/"+au.ID+"/revoke", map[string]any{"reason": "x"}); r.StatusCode != 403 || r.ErrCode() != "reauth_required" {
		t.Fatalf("%d %s", r.StatusCode, r.Body)
	}
	c.expect(c.get("/api/authorizations?status=active"), 200).JSON(&list)
	if len(list.Items) != 1 {
		t.Fatal("a recusa por falta de reautenticação não pode revogar")
	}
	c.expect(c.post("/api/auth/reauth", map[string]string{"password": testPassword, "code": e.code(c.secret)}), 200)
	var rev struct {
		Authorization struct {
			Status       string `json:"status"`
			RevokeReason string `json:"revoke_reason"`
		} `json:"authorization"`
		AlreadyRevoked bool `json:"already_revoked"`
		HaltFailed     bool `json:"halt_failed"`
	}
	c.expect(c.post("/api/authorizations/"+au.ID+"/revoke", map[string]any{"reason": "escopo mudou"}), 200).JSON(&rev)
	if rev.Authorization.Status != "revoked" || rev.Authorization.RevokeReason != "escopo mudou" || rev.AlreadyRevoked || rev.HaltFailed {
		t.Fatalf("%+v", rev)
	}
	c.expect(c.post("/api/authorizations/"+au.ID+"/revoke", map[string]any{"reason": "de novo"}), 200).JSON(&rev)
	if !rev.AlreadyRevoked {
		t.Fatalf("%+v", rev)
	}
	c.expect(c.get("/api/authorizations?status=revoked"), 200).JSON(&list)
	if len(list.Items) != 1 || list.Items[0].Remaining != 0 {
		t.Fatalf("%+v", list)
	}
	if r := c.post("/api/authorizations/00000000-0000-4000-8000-000000000000/revoke", map[string]any{}); r.StatusCode != 404 {
		t.Fatalf("%d", r.StatusCode)
	}

	// Vencida e esgotada aparecem com a situação certa.
	au2, _ := store.Create(ctx, authorizations.NewAuthorization{ProjectID: p.ID, ExecutionID: exA.ID, NodeID: "git2", Action: pushAction(), MaxUses: 1, TTL: time.Hour, GrantedBy: admin})
	_, _ = e.db.Pool.Exec(ctx, `UPDATE authorizations SET expires_at = now() - interval '1 minute' WHERE id = $1`, au2.ID)
	au3, _ := store.Create(ctx, authorizations.NewAuthorization{ProjectID: p.ID, ExecutionID: exA.ID, NodeID: "git3", Action: pushAction(), MaxUses: 1, TTL: time.Hour, GrantedBy: admin})
	_, _ = e.db.Pool.Exec(ctx, `UPDATE authorizations SET uses = 1 WHERE id = $1`, au3.ID)
	for status, id := range map[string]string{"expired": au2.ID, "exhausted": au3.ID} {
		c.expect(c.get("/api/authorizations?status="+status), 200).JSON(&list)
		if len(list.Items) != 1 || list.Items[0].ID != id || list.Items[0].Remaining != 0 {
			t.Fatalf("%s: %+v", status, list)
		}
	}

	// Sequência verificável: evento de cada ação, em ordem, e auditoria do administrador.
	_, _ = e.db.Pool.Exec(ctx, `INSERT INTO resume_events (project_id, execution_id, node_key, action, detail) VALUES ($1, $2, 'git', 'resumed', '{}'::jsonb)`, p.ID, exA.ID)
	items := sequenceOf(t, c, exA.ID)
	if countType(items, "decision", "decision_requested") != 1 || countType(items, "decision", "decision_resolution") != 1 ||
		countType(items, "execution", "decision_resolved") != 1 || countType(items, "authorization", "revoked") != 1 || countType(items, "resume", "resumed") != 1 {
		t.Fatalf("%+v", items)
	}
	for i := 1; i < len(items); i++ {
		if items[i].At.Before(items[i-1].At) {
			t.Fatalf("fora de ordem: %+v", items)
		}
	}
	if countType(sequenceOf(t, c, exR.ID), "decision", "decision_resolution") != 1 {
		t.Fatal("a recusa também aparece na sequência")
	}
	var audits int
	_ = e.db.Pool.QueryRow(ctx, `SELECT count(*) FROM auth_events WHERE type IN ('decision_resolved', 'authorization_revoked')`).Scan(&audits)
	if audits != 3 {
		t.Fatalf("aprovar, recusar e revogar (uma vez) ficam auditados: %d", audits)
	}
	// Sem notificação externa: nenhum efeito saiu para fora.
	var outbox int
	_ = e.db.Pool.QueryRow(ctx, `SELECT count(*) FROM outbox WHERE project_id = $1`, p.ID).Scan(&outbox)
	if outbox != 0 {
		t.Fatalf("nenhuma notificação externa: %d", outbox)
	}
	if r := c.get("/api/executions/00000000-0000-4000-8000-000000000000/sequence"); r.StatusCode != 404 {
		t.Fatalf("%d", r.StatusCode)
	}
	if strings.Contains(string(c.get("/api/authorizations").Body), "secret") {
		t.Fatal("nada sensível")
	}
}

func TestCentralRoutesRequireSessionAndCSRF(t *testing.T) {
	e, c, _ := newProjectEnv(t)
	anon := e.newClient()
	zero := "00000000-0000-4000-8000-000000000000"
	for _, r := range []apiResp{anon.get("/api/authorizations"), anon.get("/api/executions/" + zero + "/sequence"),
		anon.post("/api/authorizations/"+zero+"/revoke", map[string]any{}), anon.post("/api/decisions/"+zero+"/refuse", map[string]any{})} {
		if r.StatusCode != 401 {
			t.Errorf("sem sessão: %d", r.StatusCode)
		}
	}
	if r := c.do("POST", "/api/authorizations/"+zero+"/revoke", map[string]any{}, map[string]string{"X-CSRF-Token": ""}); r.StatusCode != 403 {
		t.Errorf("sem CSRF: %d", r.StatusCode)
	}
	if r := c.do("POST", "/api/decisions/"+zero+"/refuse", map[string]any{}, map[string]string{"X-CSRF-Token": ""}); r.StatusCode != 403 {
		t.Errorf("sem CSRF: %d", r.StatusCode)
	}
}
