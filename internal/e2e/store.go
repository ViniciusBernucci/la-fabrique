package e2e

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/ViniciusBernucci/le-fabrique/internal/checks"
	"github.com/ViniciusBernucci/le-fabrique/internal/execution"
	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
	"github.com/ViniciusBernucci/le-fabrique/internal/risk"
	"github.com/ViniciusBernucci/le-fabrique/internal/security"
)

// Limites das capturas.
const (
	MaxCaptures     = 20
	MaxCaptureBytes = 2 << 20
)

// Erros.
var (
	ErrNotIsolated = errors.New("o E2E só roda em ambiente isolado (sandbox do projeto, sem rede)")
	ErrNoAlt       = errors.New("a justificativa exige uma evidência alternativa existente para esta revisão")
)

var journeyPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,59}$`)

// FieldError é um erro de entrada.
type FieldError struct{ Field, Message string }

func (e *FieldError) Error() string { return e.Field + ": " + e.Message }
func (e *FieldError) Unwrap() error { return persistence.ErrInvalidInput }

// Isolated é um executor que declara se roda isolado.
type Isolated interface {
	checks.Execer
	Isolated() bool
	Browser() string
}

// Reader lê as capturas (caminhos relativos ao workspace) do ambiente do teste.
type Reader interface {
	Read(ctx context.Context, path string) ([]byte, error)
}

// SandboxEnv liga o E2E ao sandbox: executa e lê as capturas dentro dele. É isolado quando o
// perfil não tem rede.
type SandboxEnv struct {
	Sandbox     *execution.Sandbox
	BrowserName string
}

// Run implementa checks.Execer.
func (s SandboxEnv) Run(ctx context.Context, command string, timeout time.Duration) (checks.Outcome, error) {
	return checks.SandboxExecer{Sandbox: s.Sandbox}.Run(ctx, command, timeout)
}

// Isolated: sem rede.
func (s SandboxEnv) Isolated() bool { return s.Sandbox.Profile.Network == execution.NetNone }

// Browser devolve o navegador declarado.
func (s SandboxEnv) Browser() string { return s.BrowserName }

// Read lê uma captura do workspace (limitada ao tamanho máximo).
func (s SandboxEnv) Read(ctx context.Context, p string) ([]byte, error) {
	r, err := s.Sandbox.Exec(ctx, execution.ExecSpec{Cmd: []string{"sh", "-c", `head -c ` + fmt.Sprint(MaxCaptureBytes+1) + ` -- "$1"`, "read", p}})
	if err != nil {
		return nil, err
	}
	if r.ExitCode != 0 {
		return nil, fmt.Errorf("captura %q não encontrada", p)
	}
	return []byte(r.Stdout), nil
}

// DB é o que o pacote precisa do PostgreSQL.
type DB interface {
	persistence.DBTX
	persistence.TxStarter
}

// Store grava e confere evidências.
type Store struct {
	DB       DB
	Checks   *checks.Store
	Redactor *security.Redactor
	Now      func() time.Time
}

// Request pede a execução de uma jornada.
type Request struct {
	ProjectID, FeatureID, TaskKey, Revision string
	Journey                                 string
	Command                                 string
	Timeout                                 time.Duration
	Fixtures                                []string // dados de teste usados (nomes, sem valores sensíveis)
	Captures                                []string // caminhos relativos no workspace
}

// Evidence é a evidência de uma jornada.
type Evidence struct {
	ID         string    `json:"id"`
	Journey    string    `json:"journey"`
	Revision   string    `json:"revision"`
	Status     string    `json:"status"`
	Browser    string    `json:"browser"`
	Isolated   bool      `json:"isolated"`
	Fixtures   []string  `json:"fixtures"`
	CheckRunID string    `json:"check_run_id"`
	Captures   []Capture `json:"captures"`
}

// Capture é um arquivo capturado (com hash).
type Capture struct {
	ID     string `json:"id"`
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Size   int    `json:"size"`
}

func validCapturePath(p string) bool {
	return p != "" && len(p) <= 300 && !strings.HasPrefix(p, "/") && !strings.ContainsAny(p, "\x00\n\r\\") && !strings.Contains(p, "..")
}

// Run executa a jornada num ambiente ISOLADO, registra a execução (T14-01) e guarda a
// evidência: jornada, fixtures, navegador e capturas com hash, presas à revisão.
func (s *Store) Run(ctx context.Context, req Request, env Isolated, rd Reader) (Evidence, error) {
	if !journeyPattern.MatchString(req.Journey) {
		return Evidence{}, &FieldError{"journey", "nome inválido (minúsculas, números, - e _)"}
	}
	if len(req.Captures) > MaxCaptures {
		return Evidence{}, &FieldError{"captures", fmt.Sprintf("no máximo %d capturas", MaxCaptures)}
	}
	for _, p := range req.Captures {
		if !validCapturePath(p) {
			return Evidence{}, &FieldError{"captures", fmt.Sprintf("caminho inválido: %q", p)}
		}
	}
	if !env.Isolated() {
		return Evidence{}, ErrNotIsolated
	}
	for _, f := range req.Fixtures {
		if strings.TrimSpace(f) == "" || len(f) > 200 || s.Redactor.Redact(f) != f {
			return Evidence{}, &FieldError{"fixtures", "fixture inválida ou com segredo"}
		}
	}
	runs, err := s.Checks.RunSet(ctx, req.ProjectID, req.FeatureID, checks.RoleCandidate, req.Revision, env,
		[]checks.Spec{{Name: "e2e:" + req.Journey, Command: req.Command, Timeout: req.Timeout}})
	if err != nil {
		return Evidence{}, err
	}
	run := runs[0]
	status := map[string]string{checks.StatusPass: "pass", checks.StatusFail: "fail"}[run.Status]
	if status == "" {
		status = "error"
	}
	type blob struct {
		path string
		data []byte
	}
	var blobs []blob
	for _, p := range req.Captures {
		data, err := rd.Read(ctx, p)
		if err != nil {
			continue // captura ausente não derruba a evidência; fica de fora
		}
		if len(data) > MaxCaptureBytes {
			return Evidence{}, &FieldError{"captures", fmt.Sprintf("a captura %q passa de %d bytes", p, MaxCaptureBytes)}
		}
		blobs = append(blobs, blob{p, data})
	}
	fx, _ := json.Marshal(nonNil(req.Fixtures))
	ev := Evidence{Journey: req.Journey, Revision: req.Revision, Status: status, Browser: env.Browser(), Isolated: true, Fixtures: nonNil(req.Fixtures), CheckRunID: run.ID, Captures: []Capture{}}
	err = persistence.WithTx(ctx, s.DB, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx, `INSERT INTO e2e_evidence (project_id, feature_id, task_key, revision, journey, check_run_id, status, browser, isolated, fixtures)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, true, $9::jsonb) RETURNING id`,
			req.ProjectID, req.FeatureID, req.TaskKey, req.Revision, req.Journey, run.ID, status, ev.Browser, string(fx)).Scan(&ev.ID); err != nil {
			return persistence.MapError(err)
		}
		for _, b := range blobs {
			h := sha256.Sum256(b.data)
			sum := hex.EncodeToString(h[:])
			var id string
			if err := tx.QueryRow(ctx, `INSERT INTO e2e_captures (evidence_id, path, sha256, size, content) VALUES ($1, $2, $3, $4, $5) RETURNING id`,
				ev.ID, b.path, sum, len(b.data), b.data).Scan(&id); err != nil {
				return persistence.MapError(err)
			}
			ev.Captures = append(ev.Captures, Capture{ID: id, Path: b.path, SHA256: sum, Size: len(b.data)})
		}
		return nil
	})
	return ev, err
}

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

// Waiver é a justificativa de inaplicabilidade.
type Waiver struct {
	ID       string `json:"id"`
	Subject  string `json:"subject"`
	Revision string `json:"revision"`
	Reason   string `json:"reason"`
	AltKind  string `json:"alt_kind"`
	AltRef   string `json:"alt_ref"`
}

// WaiverInput pede a justificativa. Subject: "tests" ou "e2e:<jornada>".
type WaiverInput struct {
	ProjectID, FeatureID, TaskKey, Revision, Subject, Reason, AltKind, AltRef string
	AdminID                                                                   string
}

var subjectPattern = regexp.MustCompile(`^(tests|e2e:[a-z0-9][a-z0-9_-]{0,59})$`)

// Waive registra por que o teste não se aplica, com uma evidência alternativa que PRECISA
// existir para a mesma revisão: um check aprovado (`check_run`), uma captura (`capture`) ou
// uma nota de revisão (`review`, texto de 15+ caracteres).
func (s *Store) Waive(ctx context.Context, in WaiverInput) (Waiver, error) {
	if !subjectPattern.MatchString(in.Subject) {
		return Waiver{}, &FieldError{"subject", "use tests ou e2e:<jornada>"}
	}
	if n := len(strings.TrimSpace(in.Reason)); n < 20 || n > 1000 {
		return Waiver{}, &FieldError{"reason", "justifique (20 a 1000 caracteres)"}
	}
	switch in.AltKind {
	case "check_run":
		var st string
		err := s.DB.QueryRow(ctx, `SELECT status FROM check_runs WHERE id::text = $1 AND feature_id = $2 AND revision = $3`, in.AltRef, in.FeatureID, in.Revision).Scan(&st)
		if err != nil || st != checks.StatusPass {
			return Waiver{}, fmt.Errorf("%w: o check precisa existir, ser da mesma revisão e ter passado", ErrNoAlt)
		}
	case "capture":
		var one int
		err := s.DB.QueryRow(ctx, `SELECT 1 FROM e2e_captures c JOIN e2e_evidence e ON e.id = c.evidence_id WHERE c.id::text = $1 AND e.feature_id = $2 AND e.revision = $3`,
			in.AltRef, in.FeatureID, in.Revision).Scan(&one)
		if err != nil {
			return Waiver{}, fmt.Errorf("%w: a captura precisa existir e ser da mesma revisão", ErrNoAlt)
		}
	case "review":
		if len(strings.TrimSpace(in.AltRef)) < 15 {
			return Waiver{}, fmt.Errorf("%w: descreva a revisão (15+ caracteres)", ErrNoAlt)
		}
	default:
		return Waiver{}, &FieldError{"alt_kind", "use check_run, capture ou review"}
	}
	var admin any
	if in.AdminID != "" {
		admin = in.AdminID
	}
	w := Waiver{Subject: in.Subject, Revision: in.Revision, Reason: strings.TrimSpace(in.Reason), AltKind: in.AltKind, AltRef: in.AltRef}
	err := s.DB.QueryRow(ctx, `INSERT INTO e2e_waivers (project_id, feature_id, task_key, revision, subject, reason, alt_kind, alt_ref, admin_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING id`, in.ProjectID, in.FeatureID, in.TaskKey, in.Revision, in.Subject, w.Reason, in.AltKind, in.AltRef, admin).Scan(&w.ID)
	return w, persistence.MapError(err)
}

// JourneyStatus é a situação de uma jornada na revisão.
type JourneyStatus struct {
	Journey    string `json:"journey"`
	Status     string `json:"status"` // passed | failed | missing | waived
	EvidenceID string `json:"evidence_id,omitempty"`
	WaiverID   string `json:"waiver_id,omitempty"`
}

// Result é o veredito de testes e E2E de uma revisão.
type Result struct {
	Blocked     bool            `json:"blocked"`
	Reasons     []string        `json:"reasons"`
	E2E         Decision        `json:"e2e_decision"`
	Journeys    []JourneyStatus `json:"journeys"`
	TestGaps    []TestGap       `json:"test_gaps"`
	TestsWaived bool            `json:"tests_waived"`
}

// Gate confere, para a revisão, a orientação de testes e o E2E exigido pelo nível de risco e
// pelo impacto. `journeys` são as jornadas de usuário configuradas no projeto. E2E exigido sem
// jornada configurada é um caso inaplicável: só passa com justificativa e evidência alternativa.
func (s *Store) Gate(ctx context.Context, projectID, featureID, taskKey, revision, level string, files []risk.File, cfg Config, journeys []string) (Result, error) {
	res := Result{Reasons: []string{}, Journeys: []JourneyStatus{}}
	res.E2E = Decide(level, files, cfg)
	res.TestGaps = TestGaps(files)
	waiver := func(subject string) (string, error) {
		var id string
		err := s.DB.QueryRow(ctx, `SELECT id::text FROM e2e_waivers WHERE feature_id = $1 AND task_key = $2 AND revision = $3 AND subject = $4 ORDER BY created_at DESC LIMIT 1`,
			featureID, taskKey, revision, subject).Scan(&id)
		if errors.Is(persistence.MapError(err), persistence.ErrNotFound) {
			return "", nil
		}
		return id, persistence.MapError(err)
	}
	// Orientação de testes: em risco alto a falta bloqueia, salvo justificativa.
	if len(res.TestGaps) > 0 {
		w, err := waiver("tests")
		if err != nil {
			return res, err
		}
		res.TestsWaived = w != ""
		if level == risk.High && w == "" {
			res.Blocked = true
			res.Reasons = append(res.Reasons, fmt.Sprintf("%d arquivo(s) de código alterado(s) sem teste criado ou atualizado (ex.: %s)", len(res.TestGaps), res.TestGaps[0].File))
		}
	}
	if !res.E2E.Required {
		return res, nil
	}
	if len(journeys) == 0 {
		w, err := waiver("e2e:sem-jornada")
		if err != nil {
			return res, err
		}
		st := JourneyStatus{Journey: "sem-jornada", Status: "missing"}
		if w != "" {
			st.Status, st.WaiverID = "waived", w
		} else {
			res.Blocked = true
			res.Reasons = append(res.Reasons, "E2E exigido ("+res.E2E.Reason+"), mas o projeto não tem jornada de usuário testável: registre a justificativa e uma evidência alternativa")
		}
		res.Journeys = append(res.Journeys, st)
		return res, nil
	}
	for _, j := range journeys {
		st := JourneyStatus{Journey: j, Status: "missing"}
		var id, status string
		err := s.DB.QueryRow(ctx, `SELECT id::text, status FROM e2e_evidence WHERE feature_id = $1 AND task_key = $2 AND revision = $3 AND journey = $4 ORDER BY created_at DESC, id DESC LIMIT 1`,
			featureID, taskKey, revision, j).Scan(&id, &status)
		switch {
		case err == nil && status == "pass":
			st.Status, st.EvidenceID = "passed", id
		case err == nil:
			st.Status, st.EvidenceID = "failed", id
			w, werr := waiver("e2e:" + j)
			if werr != nil {
				return res, werr
			}
			if w != "" {
				st.Status, st.WaiverID = "waived", w
			}
		case !errors.Is(persistence.MapError(err), persistence.ErrNotFound):
			return res, persistence.MapError(err)
		default:
			w, werr := waiver("e2e:" + j)
			if werr != nil {
				return res, werr
			}
			if w != "" {
				st.Status, st.WaiverID = "waived", w
			}
		}
		switch st.Status {
		case "failed":
			res.Blocked = true
			res.Reasons = append(res.Reasons, fmt.Sprintf("a jornada %q falhou nesta revisão", j))
		case "missing":
			res.Blocked = true
			res.Reasons = append(res.Reasons, fmt.Sprintf("a jornada %q não tem E2E nesta revisão (nem justificativa com evidência alternativa)", j))
		}
		res.Journeys = append(res.Journeys, st)
	}
	return res, nil
}

// Captures devolve o conteúdo de uma captura (reconferindo o hash).
func (s *Store) Capture(ctx context.Context, id string) ([]byte, Capture, error) {
	var c Capture
	var data []byte
	if err := s.DB.QueryRow(ctx, `SELECT id::text, path, sha256, size, content FROM e2e_captures WHERE id::text = $1`, id).Scan(&c.ID, &c.Path, &c.SHA256, &c.Size, &data); err != nil {
		return nil, c, persistence.MapError(err)
	}
	h := sha256.Sum256(data)
	if hex.EncodeToString(h[:]) != c.SHA256 {
		return nil, c, errors.New("captura corrompida: o hash não confere")
	}
	return data, c, nil
}
