package checks

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/ViniciusBernucci/le-fabrique/internal/execution"
	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
	"github.com/ViniciusBernucci/le-fabrique/internal/security"
)

// Papéis e estados da execução.
const (
	RoleBaseline  = "baseline"
	RoleCandidate = "candidate"

	StatusPass      = "pass"
	StatusFail      = "fail"
	StatusError     = "error" // o check não pôde rodar (infraestrutura)
	StatusTimeout   = "timeout"
	StatusNotViable = "not_viable" // o baseline não pôde ser preparado
)

const (
	maxExcerpt = 16 << 10
	maxCommand = 500
)

// Spec é um check a executar.
type Spec struct {
	Name    string
	Command string
	Timeout time.Duration
}

// Outcome é o que um Execer devolve.
type Outcome struct {
	ExitCode int
	Output   string
	TimedOut bool
}

// Execer roda um comando no ambiente da revisão (sandbox) — injetado.
type Execer interface {
	Run(ctx context.Context, command string, timeout time.Duration) (Outcome, error)
}

// SandboxExecer roda os checks no sandbox (sh -c, em /workspace).
type SandboxExecer struct{ Sandbox *execution.Sandbox }

// Run implementa Execer.
func (s SandboxExecer) Run(ctx context.Context, command string, timeout time.Duration) (Outcome, error) {
	r, err := s.Sandbox.Exec(ctx, execution.ExecSpec{Cmd: []string{"sh", "-c", command}, Timeout: timeout})
	if err != nil {
		return Outcome{}, err
	}
	return Outcome{ExitCode: r.ExitCode, Output: r.Stdout + r.Stderr, TimedOut: r.TimedOut}, nil
}

// FieldError é um erro de entrada.
type FieldError struct{ Field, Message string }

func (e *FieldError) Error() string { return e.Field + ": " + e.Message }
func (e *FieldError) Unwrap() error { return persistence.ErrInvalidInput }

var (
	ansi     = regexp.MustCompile(`\x1b\[[0-9;]*[A-Za-z]`)
	digits   = regexp.MustCompile(`[0-9]+(\.[0-9]+)?`)
	hexAddr  = regexp.MustCompile(`0x[0-9a-fA-F]+`)
	spaces   = regexp.MustCompile(`\s+`)
	errLine  = regexp.MustCompile(`(?i)(error|fail|panic|exception|assert|denied|not found|cannot|undefined)`)
	tmpPaths = regexp.MustCompile(`(/tmp|/var/folders)/[^\s:]+`)
)

// normalizeLine tira o que muda entre execuções sem mudar a falha.
func normalizeLine(l string) string {
	l = tmpPaths.ReplaceAllString(l, "<tmp>")
	l = hexAddr.ReplaceAllString(l, "<addr>")
	l = digits.ReplaceAllString(l, "#")
	return strings.ToLower(spaces.ReplaceAllString(strings.TrimSpace(l), " "))
}

// Signature identifica uma falha de forma estável entre execuções: código de saída,
// as linhas de erro normalizadas (até 20) e as 3 últimas linhas da saída (onde costuma
// estar a causa, mesmo sem palavra-chave). Cores, tempos, números, endereços e
// caminhos temporários não entram.
func Signature(exitCode int, output string) string {
	output = ansi.ReplaceAllString(output, "")
	var errs, all []string
	for _, l := range strings.Split(output, "\n") {
		n := normalizeLine(l)
		if n == "" {
			continue
		}
		all = append(all, n)
		if errLine.MatchString(l) && len(errs) < 20 {
			errs = append(errs, n)
		}
	}
	tail := all
	if len(tail) > 3 {
		tail = tail[len(tail)-3:]
	}
	h := sha256.Sum256([]byte(fmt.Sprintf("%d\n%s\n--\n%s", exitCode, strings.Join(errs, "\n"), strings.Join(tail, "\n"))))
	return hex.EncodeToString(h[:])
}

// Run é uma execução gravada.
type Run struct {
	ID             string    `json:"id"`
	Role           string    `json:"role"`
	Revision       string    `json:"revision"`
	CheckName      string    `json:"check_name"`
	Command        string    `json:"command"`
	Status         string    `json:"status"`
	ExitCode       *int      `json:"exit_code"`
	Signature      string    `json:"signature"`
	OutputExcerpt  string    `json:"output_excerpt"`
	OutputHash     string    `json:"output_hash"`
	Truncated      bool      `json:"truncated"`
	Reason         string    `json:"reason"`
	DurationMS     int       `json:"duration_ms"`
	Classification string    `json:"classification,omitempty"`
	BaselineRunID  string    `json:"baseline_run_id,omitempty"`
	Resolution     string    `json:"resolution,omitempty"`
	StartedAt      time.Time `json:"started_at"`
	FinishedAt     time.Time `json:"finished_at"`
}

// DB é o que o pacote precisa do PostgreSQL.
type DB interface {
	persistence.DBTX
	persistence.TxStarter
}

// Store executa e grava checks.
type Store struct {
	DB       DB
	Redactor *security.Redactor
	Now      func() time.Time
}

func (s *Store) now() time.Time {
	if s.Now != nil {
		return s.Now().UTC()
	}
	return time.Now().UTC()
}

const runCols = `id, role, revision, check_name, command, status, exit_code, signature, output_excerpt, output_hash, truncated, reason, duration_ms,
	COALESCE(classification, ''), COALESCE(baseline_run_id::text, ''), resolution, started_at, finished_at`

func scanRun(row interface{ Scan(...any) error }) (Run, error) {
	var r Run
	err := row.Scan(&r.ID, &r.Role, &r.Revision, &r.CheckName, &r.Command, &r.Status, &r.ExitCode, &r.Signature, &r.OutputExcerpt, &r.OutputHash,
		&r.Truncated, &r.Reason, &r.DurationMS, &r.Classification, &r.BaselineRunID, &r.Resolution, &r.StartedAt, &r.FinishedAt)
	return r, persistence.MapError(err)
}

func (s *Store) clip(output string) (excerpt, hash string, truncated bool) {
	output = s.Redactor.Redact(output)
	h := sha256.Sum256([]byte(output))
	hash = hex.EncodeToString(h[:])
	if len(output) > maxExcerpt {
		// Fica o fim: é onde costumam estar o resumo e o erro.
		cut := len(output) - maxExcerpt
		for cut < len(output) && output[cut]&0xC0 == 0x80 {
			cut++
		}
		return output[cut:], hash, true
	}
	return output, hash, false
}

// RunSet executa os checks em sequência na revisão informada e grava cada resultado.
// Falha de infraestrutura vira `error` (nunca "falha do código").
func (s *Store) RunSet(ctx context.Context, projectID, featureID, role, revision string, ex Execer, specs []Spec) ([]Run, error) {
	if role != RoleBaseline && role != RoleCandidate {
		return nil, &FieldError{"role", "use baseline ou candidate"}
	}
	if strings.TrimSpace(revision) == "" || len(revision) > 100 {
		return nil, &FieldError{"revision", "informe a revisão testada (ex.: SHA)"}
	}
	if len(specs) == 0 {
		return nil, &FieldError{"checks", "informe ao menos um check"}
	}
	seen := map[string]bool{}
	for _, sp := range specs {
		if strings.TrimSpace(sp.Name) == "" || len(sp.Name) > 60 || strings.TrimSpace(sp.Command) == "" || len(sp.Command) > maxCommand || strings.ContainsAny(sp.Command, "\x00\r\n") {
			return nil, &FieldError{"checks", fmt.Sprintf("check inválido: %q", sp.Name)}
		}
		if seen[sp.Name] {
			return nil, &FieldError{"checks", fmt.Sprintf("check repetido: %q", sp.Name)}
		}
		seen[sp.Name] = true
		if s.Redactor.Redact(sp.Command) != sp.Command {
			return nil, &FieldError{"checks", fmt.Sprintf("o comando de %q contém um segredo", sp.Name)}
		}
	}
	out := make([]Run, 0, len(specs))
	for _, sp := range specs {
		started := s.now()
		o, err := ex.Run(ctx, sp.Command, sp.Timeout)
		finished := s.now()
		status, reason := StatusPass, ""
		var exit *int
		switch {
		case err != nil:
			status, reason = StatusError, "o check não pôde ser executado: "+s.Redactor.Redact(err.Error())
		case o.TimedOut:
			status, reason = StatusTimeout, "tempo limite excedido"
		case o.ExitCode != 0:
			status = StatusFail
			c := o.ExitCode
			exit = &c
		default:
			c := 0
			exit = &c
		}
		excerpt, hash, trunc := s.clip(o.Output)
		sig := ""
		if status == StatusFail {
			sig = Signature(o.ExitCode, o.Output)
		}
		r, err := scanRun(s.DB.QueryRow(ctx, `INSERT INTO check_runs (project_id, feature_id, role, revision, check_name, command, status, exit_code, signature,
			output_excerpt, output_hash, truncated, reason, duration_ms, started_at, finished_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16) RETURNING `+runCols,
			projectID, featureID, role, revision, sp.Name, sp.Command, status, exit, sig, excerpt, hash, trunc, reason,
			int(finished.Sub(started)/time.Millisecond), started, finished))
		if err != nil {
			return out, err
		}
		out = append(out, r)
	}
	return out, nil
}

// RecordNotViable registra que o baseline não pôde ser preparado (ex.: revisão
// anterior não compila por falta de dependência), com o motivo. Sem baseline, as
// falhas da candidata ficam em investigação.
func (s *Store) RecordNotViable(ctx context.Context, projectID, featureID, revision, reason string, specs []Spec) error {
	if strings.TrimSpace(reason) == "" || len(reason) > 500 {
		return &FieldError{"reason", "explique por que o baseline não é viável"}
	}
	now := s.now()
	for _, sp := range specs {
		if _, err := s.DB.Exec(ctx, `INSERT INTO check_runs (project_id, feature_id, role, revision, check_name, command, status, reason, started_at, finished_at)
			VALUES ($1, $2, 'baseline', $3, $4, $5, 'not_viable', $6, $7, $7)`, projectID, featureID, revision, sp.Name, sp.Command, s.Redactor.Redact(reason), now); err != nil {
			return persistence.MapError(err)
		}
	}
	return nil
}

// ErrNoRun: não há execução para o que foi pedido.
var ErrNoRun = errors.New("não há execução registrada para esta revisão")
