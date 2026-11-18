// Package projectsetup trata o script de setup do repositório (equivalente ao orca.yaml; ADR-0062,
// fase 3). O script é um arquivo versionado (.lafabrique/setup.sh) e roda SÓ dentro do sandbox,
// nunca no host da API. Rodar exige aprovação do usuário vinculada ao hash do conteúdo, por projeto
// e ref: conteúdo alterado muda o hash e pede nova aprovação. O que roda é exatamente o que foi
// hasheado (lido uma vez do commit e enviado por stdin), sem janela entre a checagem e o uso.
package projectsetup

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/ViniciusBernucci/le-fabrique/internal/execution"
	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
	"github.com/ViniciusBernucci/le-fabrique/internal/workspace"
	"github.com/ViniciusBernucci/le-fabrique/internal/worktrees"
)

// ScriptPath é o arquivo de setup, relativo à raiz do repositório.
const ScriptPath = ".lafabrique/setup.sh"

// MaxScript é o tamanho máximo do script (para o usuário conseguir revisar).
const MaxScript = 64 << 10

// Erros de domínio.
var (
	ErrNoScript    = errors.New("o repositório não tem " + ScriptPath + " nesta referência")
	ErrNotApproved = errors.New("o script de setup não está aprovado para este conteúdo: revise e aprove de novo")
	ErrChanged     = errors.New("o script mudou desde que você o revisou: revise o conteúdo atual")
	ErrBadScript   = errors.New("o script de setup precisa ser texto (UTF-8) sem bytes nulos")
	ErrNoSandbox   = errors.New("o sandbox (Docker) não está disponível neste servidor")
	ErrInvalid     = persistence.ErrInvalidInput
)

// Sandboxer cria sandboxes; *execution.Docker satisfaz.
type Sandboxer interface {
	Provision(ctx context.Context, spec execution.Spec) (*execution.Sandbox, error)
	Available(ctx context.Context) bool
}

// Service lê, aprova e executa o setup dos projetos.
type Service struct {
	DB      worktrees.DB
	Root    workspace.Root
	Git     workspace.Git
	Sandbox Sandboxer
}

// State é o que o usuário vê para decidir: o script atual e se está aprovado.
type State struct {
	Ref        string     `json:"ref"`
	Commit     string     `json:"commit"`
	Found      bool       `json:"found"`
	SHA256     string     `json:"sha256"`
	Content    string     `json:"content"`
	Approved   bool       `json:"approved"`
	ApprovalID string     `json:"approval_id,omitempty"`
	ApprovedAt *time.Time `json:"approved_at,omitempty"`
	LastRun    *Run       `json:"last_run,omitempty"`
}

// Run é o resultado de uma execução do setup.
type Run struct {
	ID         string    `json:"id"`
	Ref        string    `json:"ref"`
	SHA256     string    `json:"sha256"`
	Status     string    `json:"status"`
	ExitCode   int       `json:"exit_code"`
	Output     string    `json:"output_tail"`
	FinishedAt time.Time `json:"finished_at"`
}

type script struct {
	commit, content, sum string
	found                bool
}

func (s Service) repo(ctx context.Context, projectID string) (string, error) {
	m, err := worktrees.Provider{DB: s.DB, Root: s.Root}.For(ctx, projectID)
	if err != nil {
		return "", err
	}
	return m.Repo, nil
}

func (s Service) read(ctx context.Context, repo, ref string) (script, error) {
	commit, err := s.Git.ResolveCommit(ctx, repo, ref)
	if err != nil {
		return script{}, fmt.Errorf("%w: %v", ErrInvalid, err)
	}
	content, found, err := s.Git.ReadBlob(ctx, repo, commit, ScriptPath, MaxScript)
	if err != nil {
		return script{}, fmt.Errorf("%w: %v", ErrInvalid, err)
	}
	if !found {
		return script{commit: commit}, nil
	}
	if !utf8.ValidString(content) || strings.ContainsRune(content, 0) {
		return script{}, ErrBadScript
	}
	sum := sha256.Sum256([]byte(content))
	return script{commit: commit, content: content, sum: hex.EncodeToString(sum[:]), found: true}, nil
}

// Inspect devolve o script atual de ref e se ele está aprovado.
func (s Service) Inspect(ctx context.Context, projectID, ref string) (State, error) {
	repo, err := s.repo(ctx, projectID)
	if err != nil {
		return State{}, err
	}
	sc, err := s.read(ctx, repo, ref)
	if err != nil {
		return State{}, err
	}
	st := State{Ref: ref, Commit: sc.commit, Found: sc.found, SHA256: sc.sum, Content: sc.content}
	if sc.found {
		var at time.Time
		if err := s.DB.QueryRow(ctx, `SELECT id::text, approved_at FROM project_setup_approvals
			WHERE project_id::text = $1 AND ref = $2 AND script_sha256 = $3 AND revoked_at IS NULL`, projectID, ref, sc.sum).Scan(&st.ApprovalID, &at); err == nil {
			st.Approved, st.ApprovedAt = true, &at
		}
	}
	var r Run
	if err := s.DB.QueryRow(ctx, `SELECT id::text, ref, script_sha256, status, exit_code, output_tail, finished_at FROM project_setup_runs
		WHERE project_id::text = $1 AND ref = $2 ORDER BY started_at DESC LIMIT 1`, projectID, ref).
		Scan(&r.ID, &r.Ref, &r.SHA256, &r.Status, &r.ExitCode, &r.Output, &r.FinishedAt); err == nil {
		st.LastRun = &r
	}
	return st, nil
}

func (s Service) audit(ctx context.Context, projectID, adminID, action string, detail map[string]any) {
	raw, _ := json.Marshal(detail)
	var admin any
	if adminID != "" {
		admin = adminID
	}
	_, _ = s.DB.Exec(ctx, `INSERT INTO project_changes (project_id, admin_id, action, detail) VALUES ($1,$2,$3,$4::jsonb)`, projectID, admin, action, string(raw))
}

// Approve aprova o script de ref. sha256 é o hash que o usuário revisou: se o conteúdo atual for
// outro, recusa (ErrChanged) em vez de aprovar o que ele não viu.
func (s Service) Approve(ctx context.Context, projectID, adminID, ref, sha256sum string) (State, error) {
	repo, err := s.repo(ctx, projectID)
	if err != nil {
		return State{}, err
	}
	sc, err := s.read(ctx, repo, ref)
	if err != nil {
		return State{}, err
	}
	if !sc.found {
		return State{}, ErrNoScript
	}
	if sc.sum != sha256sum {
		return State{}, ErrChanged
	}
	var admin any
	if adminID != "" {
		admin = adminID
	}
	if _, err := s.DB.Exec(ctx, `INSERT INTO project_setup_approvals (project_id, ref, script_sha256, approved_by) VALUES ($1,$2,$3,$4)
		ON CONFLICT DO NOTHING`, projectID, ref, sc.sum, admin); err != nil {
		return State{}, persistence.MapError(err)
	}
	s.audit(ctx, projectID, adminID, "setup_approved", map[string]any{"ref": ref, "sha256": sc.sum})
	return s.Inspect(ctx, projectID, ref)
}

// Revoke revoga uma aprovação (o script volta a exigir aprovação).
func (s Service) Revoke(ctx context.Context, projectID, adminID, approvalID string) error {
	var ref, sum string
	err := s.DB.QueryRow(ctx, `UPDATE project_setup_approvals SET revoked_at = now()
		WHERE id::text = $1 AND project_id::text = $2 AND revoked_at IS NULL RETURNING ref, script_sha256`, approvalID, projectID).Scan(&ref, &sum)
	if err != nil {
		return persistence.MapError(err)
	}
	s.audit(ctx, projectID, adminID, "setup_revoked", map[string]any{"ref": ref, "sha256": sum})
	return nil
}

// RunSpec descreve uma execução do setup.
type RunSpec struct {
	ProjectID string
	AdminID   string
	Ref       string // ref do script (a aprovação vale para ela)
	Tree      string // o que semear no sandbox (branch/SHA); vazio = Ref
	Profile   string // perfil do sandbox (padrão "base")
	Timeout   time.Duration
}

// Run executa o setup aprovado no sandbox: sem rede, sem segredos, só a árvore versionada de Tree.
// Sem aprovação para o conteúdo atual, nada roda.
func (s Service) Run(ctx context.Context, rs RunSpec) (Run, error) {
	if s.Sandbox == nil || !s.Sandbox.Available(ctx) {
		return Run{}, ErrNoSandbox
	}
	repo, err := s.repo(ctx, rs.ProjectID)
	if err != nil {
		return Run{}, err
	}
	sc, err := s.read(ctx, repo, rs.Ref)
	if err != nil {
		return Run{}, err
	}
	if !sc.found {
		return Run{}, ErrNoScript
	}
	var n int
	if err := s.DB.QueryRow(ctx, `SELECT count(*) FROM project_setup_approvals WHERE project_id::text = $1 AND ref = $2 AND script_sha256 = $3 AND revoked_at IS NULL`,
		rs.ProjectID, rs.Ref, sc.sum).Scan(&n); err != nil {
		return Run{}, persistence.MapError(err)
	}
	if n == 0 {
		return Run{}, ErrNotApproved
	}
	tree := rs.Tree
	if tree == "" {
		tree = rs.Ref
	}
	commit, err := s.Git.ResolveCommit(ctx, repo, tree)
	if err != nil {
		return Run{}, fmt.Errorf("%w: %v", ErrInvalid, err)
	}
	tarball, err := s.Git.Archive(ctx, repo, commit)
	if err != nil {
		return Run{}, err
	}
	name := rs.Profile
	if name == "" {
		name = "base"
	}
	prof, ok := execution.Profiles[name]
	if !ok {
		return Run{}, fmt.Errorf("%w: perfil de sandbox desconhecido", ErrInvalid)
	}
	prof.Network = execution.NetNone // setup não tem internet
	if rs.Timeout <= 0 || rs.Timeout > 15*time.Minute {
		rs.Timeout = 10 * time.Minute
	}
	started := time.Now()
	sbx, err := s.Sandbox.Provision(ctx, execution.Spec{ProjectID: sandboxID(rs.ProjectID), FeatureID: "setup", Profile: prof})
	if err != nil {
		return Run{}, err
	}
	defer sbx.Destroy(context.WithoutCancel(ctx))
	if err := sbx.Seed(ctx, bytes.NewReader(tarball)); err != nil {
		return Run{}, err
	}
	res, err := sbx.Exec(ctx, execution.ExecSpec{Cmd: []string{"sh", "-s"}, Stdin: strings.NewReader(sc.content), Timeout: rs.Timeout})
	if err != nil {
		return Run{}, err
	}
	out := Run{Ref: rs.Ref, SHA256: sc.sum, ExitCode: res.ExitCode, Status: "ok", Output: tail(res.Stdout+res.Stderr, 4096)}
	switch {
	case res.TimedOut:
		out.Status = "timeout"
	case res.ExitCode != 0:
		out.Status = "failed"
	}
	if err := s.DB.QueryRow(ctx, `INSERT INTO project_setup_runs (project_id, ref, script_sha256, status, exit_code, output_tail, started_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7) RETURNING id::text, finished_at`,
		rs.ProjectID, rs.Ref, sc.sum, out.Status, out.ExitCode, out.Output, started).Scan(&out.ID, &out.FinishedAt); err != nil {
		return out, persistence.MapError(err)
	}
	s.audit(ctx, rs.ProjectID, rs.AdminID, "setup_ran", map[string]any{"ref": rs.Ref, "sha256": sc.sum, "status": out.Status})
	return out, nil
}

// sandboxID adapta o UUID do projeto ao padrão de identificadores do sandbox.
func sandboxID(projectID string) string { return strings.ReplaceAll(projectID, "-", "") }

func tail(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return strings.ToValidUTF8(s[len(s)-n:], "")
}
