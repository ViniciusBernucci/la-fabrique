package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
	"github.com/ViniciusBernucci/le-fabrique/internal/policy"
	"github.com/ViniciusBernucci/le-fabrique/internal/security"
)

// Erros com resultado definido.
var (
	ErrNotGranted    = errors.New("ferramenta MCP não concedida a este agente")
	ErrSchemaChanged = errors.New("o contrato da ferramenta mudou desde a aprovação: aprove de novo")
	ErrStaleApproval = errors.New("a aprovação não corresponde ao esquema atual da ferramenta")
	ErrToolMissing   = errors.New("ferramenta inexistente ou removida do servidor")
	ErrInvalidArgs   = errors.New("argumentos fora do esquema da ferramenta")
)

// FieldError é um campo inválido.
type FieldError struct{ Field, Message string }

func (e *FieldError) Error() string { return e.Field + ": " + e.Message }
func (e *FieldError) Unwrap() error { return persistence.ErrInvalidInput }

// DB é o que o serviço precisa do banco.
type DB interface {
	persistence.DBTX
	persistence.TxStarter
}

// Server é um servidor MCP cadastrado.
type Server struct {
	ID            string     `json:"id"`
	Name          string     `json:"name"`
	URL           string     `json:"url"`
	ConnectionID  *string    `json:"connection_id"`
	Protocol      string     `json:"protocol"`
	ServerName    string     `json:"server_name"`
	ServerVersion string     `json:"server_version"`
	DiscoveredAt  *time.Time `json:"discovered_at"`
	CreatedAt     time.Time  `json:"created_at"`
}

// Tool é uma ferramenta descoberta.
type Tool struct {
	ServerID    string         `json:"server_id"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"input_schema"`
	SchemaHash  string         `json:"schema_hash"`
	Status      string         `json:"status"`
}

// Grant é a concessão de uma ferramenta a um agente, presa ao hash aprovado.
type Grant struct {
	AgentID    string `json:"agent_id"`
	ServerID   string `json:"server_id"`
	Tool       string `json:"tool"`
	SchemaHash string `json:"schema_hash"`
	Current    bool   `json:"current"` // o hash aprovado ainda é o do esquema atual
}

// Service reúne o catálogo MCP.
type Service struct {
	DB           DB
	Vault        *security.Vault
	Redactor     *security.Redactor
	AllowPrivate bool // só testes: aceita servidores em http/loopback
}

var serverName = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,63}$`)

const serverCols = `id, name, url, connection_id, protocol, server_name, server_version, discovered_at, created_at`

func scanServer(row interface{ Scan(...any) error }) (Server, error) {
	var s Server
	err := row.Scan(&s.ID, &s.Name, &s.URL, &s.ConnectionID, &s.Protocol, &s.ServerName, &s.ServerVersion, &s.DiscoveredAt, &s.CreatedAt)
	return s, persistence.MapError(err)
}

// RegisterServer cadastra um servidor. A URL precisa ser https e externa; a
// credencial, se houver, é só a referência de uma conexão `mcp` do cofre.
func (s *Service) RegisterServer(ctx context.Context, name, rawURL string, connectionID *string) (Server, error) {
	name = strings.TrimSpace(name)
	if !serverName.MatchString(name) {
		return Server{}, &FieldError{"name", "O nome deve usar minúsculas, números, hífen e sublinhado (até 64)."}
	}
	if _, err := NewClient(rawURL, security.Secret{}, s.AllowPrivate); err != nil {
		return Server{}, &FieldError{"url", "Use uma URL https de um servidor externo (sem credenciais, sem endereço interno)."}
	}
	if connectionID != nil {
		c := strings.TrimSpace(*connectionID)
		if c == "" {
			connectionID = nil
		} else {
			var kind, state string
			err := s.DB.QueryRow(ctx, `SELECT kind, state FROM connections WHERE id = $1`, c).Scan(&kind, &state)
			if err != nil || kind != "mcp" || state != "active" {
				return Server{}, &FieldError{"connection_id", "Informe uma conexão do tipo mcp, ativa."}
			}
			connectionID = &c
		}
	}
	return scanServer(s.DB.QueryRow(ctx, `INSERT INTO mcp_servers (name, url, connection_id) VALUES ($1, $2, $3::uuid) RETURNING `+serverCols,
		name, strings.TrimSpace(rawURL), connectionID))
}

// Servers lista os servidores.
func (s *Service) Servers(ctx context.Context) ([]Server, error) {
	rows, err := s.DB.Query(ctx, `SELECT `+serverCols+` FROM mcp_servers ORDER BY name`)
	if err != nil {
		return nil, persistence.MapError(err)
	}
	defer rows.Close()
	out := []Server{}
	for rows.Next() {
		sv, err := scanServer(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, sv)
	}
	return out, persistence.MapError(rows.Err())
}

// Server lê um servidor.
func (s *Service) Server(ctx context.Context, id string) (Server, error) {
	return scanServer(s.DB.QueryRow(ctx, `SELECT `+serverCols+` FROM mcp_servers WHERE id = $1`, id))
}

func (s *Service) client(ctx context.Context, sv Server) (*Client, error) {
	var token security.Secret
	if sv.ConnectionID != nil {
		if s.Vault == nil {
			return nil, errors.New("cofre ausente")
		}
		sec, _, err := s.Vault.Resolve(ctx, *sv.ConnectionID)
		if err != nil {
			return nil, err
		}
		token = sec
	}
	return NewClient(sv.URL, token, s.AllowPrivate)
}

// Tools lista as ferramentas descobertas do servidor.
func (s *Service) Tools(ctx context.Context, serverID string) ([]Tool, error) {
	rows, err := s.DB.Query(ctx, `SELECT server_id, name, description, input_schema, schema_hash, status FROM mcp_tools WHERE server_id = $1 ORDER BY name`, serverID)
	if err != nil {
		return nil, persistence.MapError(err)
	}
	defer rows.Close()
	out := []Tool{}
	for rows.Next() {
		var t Tool
		var raw []byte
		if err := rows.Scan(&t.ServerID, &t.Name, &t.Description, &raw, &t.SchemaHash, &t.Status); err != nil {
			return nil, persistence.MapError(err)
		}
		_ = json.Unmarshal(raw, &t.InputSchema)
		out = append(out, t)
	}
	return out, persistence.MapError(rows.Err())
}

func (s *Service) tool(ctx context.Context, serverID, name string) (Tool, error) {
	var t Tool
	var raw []byte
	err := s.DB.QueryRow(ctx, `SELECT server_id, name, description, input_schema, schema_hash, status FROM mcp_tools WHERE server_id = $1 AND name = $2`, serverID, name).
		Scan(&t.ServerID, &t.Name, &t.Description, &raw, &t.SchemaHash, &t.Status)
	if err != nil {
		if errors.Is(persistence.MapError(err), persistence.ErrNotFound) {
			return Tool{}, ErrToolMissing
		}
		return Tool{}, persistence.MapError(err)
	}
	_ = json.Unmarshal(raw, &t.InputSchema)
	return t, nil
}

// upsert grava o resultado de uma listagem: ferramenta nova entra ativa e SEM
// concessão; esquema mudado atualiza o hash (a concessão antiga deixa de valer);
// ausente da listagem fica `removed`.
func (s *Service) upsert(ctx context.Context, tx pgx.Tx, serverID string, defs []ToolDef) error {
	seen := map[string]bool{}
	for _, d := range defs {
		seen[d.Name] = true
		schema, _ := json.Marshal(d.InputSchema)
		if _, err := tx.Exec(ctx, `
			INSERT INTO mcp_tools (server_id, name, description, input_schema, schema_hash)
			VALUES ($1, $2, $3, $4::jsonb, $5)
			ON CONFLICT (server_id, name) DO UPDATE SET description = EXCLUDED.description, input_schema = EXCLUDED.input_schema,
				schema_hash = EXCLUDED.schema_hash, status = 'active', updated_at = now()`,
			serverID, d.Name, d.Description, string(schema), d.SchemaHash()); err != nil {
			return persistence.MapError(err)
		}
	}
	rows, err := tx.Query(ctx, `SELECT name FROM mcp_tools WHERE server_id = $1 AND status = 'active'`, serverID)
	if err != nil {
		return persistence.MapError(err)
	}
	var gone []string
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			rows.Close()
			return persistence.MapError(err)
		}
		if !seen[n] {
			gone = append(gone, n)
		}
	}
	rows.Close()
	for _, n := range gone {
		if _, err := tx.Exec(ctx, `UPDATE mcp_tools SET status = 'removed', updated_at = now() WHERE server_id = $1 AND name = $2`, serverID, n); err != nil {
			return persistence.MapError(err)
		}
	}
	return nil
}

// Discover negocia o protocolo, lista as ferramentas e atualiza o catálogo.
// Nada ganha acesso por ser descoberto.
func (s *Service) Discover(ctx context.Context, serverID string) ([]Tool, error) {
	sv, err := s.Server(ctx, serverID)
	if err != nil {
		return nil, err
	}
	c, err := s.client(ctx, sv)
	if err != nil {
		return nil, err
	}
	info, err := c.Initialize(ctx)
	if err != nil {
		return nil, err
	}
	defs, err := c.ListTools(ctx)
	if err != nil {
		return nil, err
	}
	err = persistence.WithTx(ctx, s.DB, func(tx pgx.Tx) error {
		if err := s.upsert(ctx, tx, serverID, defs); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `UPDATE mcp_servers SET protocol = $2, server_name = $3, server_version = $4, discovered_at = now() WHERE id = $1`,
			serverID, info.Protocol, info.Name, info.Version)
		return persistence.MapError(err)
	})
	if err != nil {
		return nil, err
	}
	return s.Tools(ctx, serverID)
}

// Grant concede a ferramenta ao agente fixando a revisão do esquema aprovada.
// approvedHash precisa ser o hash ATUAL: quem aprovou viu o contrato vigente.
func (s *Service) Grant(ctx context.Context, agentID, serverID, tool, approvedHash string) error {
	t, err := s.tool(ctx, serverID, tool)
	if err != nil {
		return err
	}
	if t.Status != "active" {
		return ErrToolMissing
	}
	if t.SchemaHash != approvedHash {
		return ErrStaleApproval
	}
	_, err = s.DB.Exec(ctx, `
		INSERT INTO mcp_grants (agent_id, server_id, tool_name, schema_hash) VALUES ($1, $2, $3, $4)
		ON CONFLICT (agent_id, server_id, tool_name) DO UPDATE SET schema_hash = EXCLUDED.schema_hash, granted_at = now()`,
		agentID, serverID, tool, approvedHash)
	return persistence.MapError(err)
}

// Revoke retira a concessão (idempotente).
func (s *Service) Revoke(ctx context.Context, agentID, serverID, tool string) error {
	_, err := s.DB.Exec(ctx, `DELETE FROM mcp_grants WHERE agent_id = $1 AND server_id = $2 AND tool_name = $3`, agentID, serverID, tool)
	return persistence.MapError(err)
}

// Grants lista as concessões de um agente, marcando se ainda valem.
func (s *Service) Grants(ctx context.Context, agentID string) ([]Grant, error) {
	rows, err := s.DB.Query(ctx, `
		SELECT g.agent_id, g.server_id, g.tool_name, g.schema_hash, (t.schema_hash = g.schema_hash AND t.status = 'active')
		FROM mcp_grants g JOIN mcp_tools t ON t.server_id = g.server_id AND t.name = g.tool_name
		WHERE g.agent_id = $1 ORDER BY g.server_id, g.tool_name`, agentID)
	if err != nil {
		return nil, persistence.MapError(err)
	}
	defer rows.Close()
	out := []Grant{}
	for rows.Next() {
		var g Grant
		if err := rows.Scan(&g.AgentID, &g.ServerID, &g.Tool, &g.SchemaHash, &g.Current); err != nil {
			return nil, persistence.MapError(err)
		}
		out = append(out, g)
	}
	return out, persistence.MapError(rows.Err())
}

// AgentTools devolve SÓ o que o agente pode usar agora: concedido e com o esquema
// ainda igual ao aprovado. Descrição e esquema das demais não chegam ao modelo.
func (s *Service) AgentTools(ctx context.Context, agentID string) ([]Tool, error) {
	rows, err := s.DB.Query(ctx, `
		SELECT t.server_id, t.name, t.description, t.input_schema, t.schema_hash, t.status
		FROM mcp_grants g JOIN mcp_tools t ON t.server_id = g.server_id AND t.name = g.tool_name
		WHERE g.agent_id = $1 AND t.status = 'active' AND t.schema_hash = g.schema_hash ORDER BY t.server_id, t.name`, agentID)
	if err != nil {
		return nil, persistence.MapError(err)
	}
	defer rows.Close()
	out := []Tool{}
	for rows.Next() {
		var t Tool
		var raw []byte
		if err := rows.Scan(&t.ServerID, &t.Name, &t.Description, &raw, &t.SchemaHash, &t.Status); err != nil {
			return nil, persistence.MapError(err)
		}
		_ = json.Unmarshal(raw, &t.InputSchema)
		out = append(out, t)
	}
	return out, persistence.MapError(rows.Err())
}

// CallRequest descreve uma chamada de ferramenta MCP por um agente.
type CallRequest struct {
	AgentID  string
	ServerID string
	Tool     string
	Args     map[string]any
}

func (s *Service) audit(ctx context.Context, r CallRequest, outcome, reason string, dur time.Duration, size int) {
	_, hash := policy.Canonical(r.Args)
	raw, _ := json.Marshal(policy.RedactParamsWith(r.Args, s.Redactor))
	var agent, server any
	if r.AgentID != "" {
		agent = r.AgentID
	}
	if r.ServerID != "" {
		server = r.ServerID
	}
	if len(reason) > 300 {
		reason = reason[:300]
	}
	_, _ = s.DB.Exec(context.WithoutCancel(ctx), `
		INSERT INTO mcp_calls (agent_id, server_id, tool_name, args_hash, args_redacted, outcome, reason, duration_ms, result_bytes)
		VALUES ($1::uuid, $2::uuid, $3, $4, $5::jsonb, $6, $7, $8, $9)`,
		agent, server, r.Tool, hash, string(raw), outcome, reason, int(dur.Milliseconds()), size)
}

// Call executa a chamada depois de revalidar tudo: concessão do agente, hash do
// esquema aprovado, argumentos contra o esquema, o contrato ATUAL no servidor e a
// política. Qualquer falha acontece antes de `tools/call` e fica na trilha.
func (s *Service) Call(ctx context.Context, r CallRequest) (CallResult, error) {
	start := time.Now()
	deny := func(outcome string, err error) (CallResult, error) {
		s.audit(ctx, r, outcome, err.Error(), time.Since(start), 0)
		return CallResult{}, err
	}
	sv, err := s.Server(ctx, r.ServerID)
	if err != nil {
		return deny("denied", err)
	}
	t, err := s.tool(ctx, r.ServerID, r.Tool)
	if err != nil || t.Status != "active" {
		return deny("denied", ErrToolMissing)
	}
	var granted string
	gerr := s.DB.QueryRow(ctx, `SELECT schema_hash FROM mcp_grants WHERE agent_id = $1 AND server_id = $2 AND tool_name = $3`, r.AgentID, r.ServerID, r.Tool).Scan(&granted)
	if gerr != nil {
		return deny("denied", ErrNotGranted)
	}
	if granted != t.SchemaHash {
		return deny("changed", ErrSchemaChanged)
	}
	if err := ValidateArgs(t.InputSchema, r.Args); err != nil {
		return deny("invalid", fmt.Errorf("%w: %v", ErrInvalidArgs, err))
	}

	c, err := s.client(ctx, sv)
	if err != nil {
		return deny("failed", err)
	}
	cctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	if _, err := c.Initialize(cctx); err != nil {
		return deny("failed", err)
	}
	// Revalida o contrato vigente no servidor antes de cada chamada.
	live, err := c.ListTools(cctx)
	if err != nil {
		return deny("failed", err)
	}
	var cur *ToolDef
	for i := range live {
		if live[i].Name == r.Tool {
			cur = &live[i]
		}
	}
	if cur == nil || cur.SchemaHash() != granted {
		// Registra o que o servidor anuncia agora: a concessão antiga deixa de valer.
		_ = persistence.WithTx(context.WithoutCancel(ctx), s.DB, func(tx pgx.Tx) error { return s.upsert(context.WithoutCancel(ctx), tx, r.ServerID, live) })
		return deny("changed", ErrSchemaChanged)
	}

	// Política: só esta ferramenta, concedida; o que muda estado por nome ainda exige aprovação.
	_, argsHash := policy.Canonical(r.Args)
	act := policy.Action{Tool: policy.ToolMCP, Operation: "tools/call", Resource: sv.Name, Params: map[string]any{"tool": r.Tool, "args_hash": argsHash}}
	if !s.AllowPrivate {
		act.Target = sv.URL
	}
	guard := &policy.Guard{Policy: policy.DefaultPolicy(policy.Options{AllowedMCPTools: []string{r.Tool}}), Rec: policy.Store{DB: s.DB, Redactor: s.Redactor}}
	var res CallResult
	_, err = guard.Do(cctx, act, func() error {
		var cerr error
		res, cerr = c.CallTool(cctx, r.Tool, r.Args)
		return cerr
	})
	if err != nil {
		outcome := "failed"
		if errors.Is(err, policy.ErrForbidden) || errors.Is(err, policy.ErrApprovalRequired) {
			outcome = "blocked"
		}
		return deny(outcome, err)
	}
	outcome := "ok"
	if res.IsError {
		outcome = "tool_error"
	}
	s.audit(ctx, r, outcome, "", time.Since(start), len(res.Text))
	return res, nil
}

// CallRow é uma linha da trilha de chamadas.
type CallRow struct {
	ID         int64          `json:"id"`
	AgentID    *string        `json:"agent_id"`
	ServerID   *string        `json:"server_id"`
	Tool       string         `json:"tool"`
	ArgsHash   string         `json:"args_hash"`
	Args       map[string]any `json:"args"`
	Outcome    string         `json:"outcome"`
	Reason     string         `json:"reason"`
	DurationMs int            `json:"duration_ms"`
	CreatedAt  time.Time      `json:"created_at"`
}

// Calls devolve a trilha (mais recente primeiro).
func (s *Service) Calls(ctx context.Context, limit int) ([]CallRow, error) {
	if limit < 1 || limit > 200 {
		limit = 50
	}
	rows, err := s.DB.Query(ctx, `SELECT id, agent_id, server_id, tool_name, args_hash, args_redacted, outcome, reason, duration_ms, created_at FROM mcp_calls ORDER BY id DESC LIMIT $1`, limit)
	if err != nil {
		return nil, persistence.MapError(err)
	}
	defer rows.Close()
	out := []CallRow{}
	for rows.Next() {
		var c CallRow
		var raw []byte
		if err := rows.Scan(&c.ID, &c.AgentID, &c.ServerID, &c.Tool, &c.ArgsHash, &raw, &c.Outcome, &c.Reason, &c.DurationMs, &c.CreatedAt); err != nil {
			return nil, persistence.MapError(err)
		}
		_ = json.Unmarshal(raw, &c.Args)
		out = append(out, c)
	}
	return out, persistence.MapError(rows.Err())
}
