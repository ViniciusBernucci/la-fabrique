// Package consolidate integra, de forma serializada por feature, os commits das tentativas na
// branch única de entrega (AJ06-03). A atualização da ref é condicionada à cabeça esperada (nada
// obsoleto é gravado), conflito preserva os dois lados e bloqueia a publicação, e repetir após queda
// não duplica commits (a equivalência é provada pelo trailer do cherry-pick no próprio histórico).
package consolidate

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
)

// Erros.
var (
	ErrBusy     = errors.New("já há uma consolidação em andamento para esta feature")
	ErrStale    = errors.New("a branch de entrega mudou durante a consolidação: nada foi atualizado")
	ErrConflict = errors.New("conflito na consolidação: o trabalho foi preservado e a publicação está bloqueada")
)

// DB é o que o consolidador precisa do PostgreSQL.
type DB interface {
	persistence.DBTX
	persistence.TxStarter
}

// Attempt é uma tentativa a incorporar (a ordem do pedido segue as dependências).
type Attempt struct {
	Ref     string // identificação (ID da tentativa)
	Branch  string // branch temporária com os commits
	BaseSHA string // SHA de partida da tentativa
}

// Request pede a consolidação.
type Request struct {
	ProjectID, FeatureID string
	DeliveryBranch       string // branch única de entrega da feature
	StartSHA             string // usada só quando a branch de entrega ainda não existe
	Attempts             []Attempt
}

// Equivalence liga o commit original ao integrado.
type Equivalence struct {
	Original   string `json:"original"`
	Integrated string `json:"integrated"`
	Changed    bool   `json:"changed"`
	Attempt    string `json:"attempt"`
}

// Result é o desfecho.
type Result struct {
	RunID       string        `json:"run_id"`
	ExpectedSHA string        `json:"expected_head"`
	Head        string        `json:"head"`
	Commits     []Equivalence `json:"commits"`
	Revalidate  []string      `json:"revalidate"` // tentativas cuja evidência de revisão perdeu validade
	Conflict    *Conflict     `json:"conflict,omitempty"`
}

// Conflict descreve o conflito encontrado.
type Conflict struct {
	Attempt string   `json:"attempt"`
	Commit  string   `json:"commit"`
	Files   []string `json:"files"`
}

// Consolidator integra tentativas.
type Consolidator struct {
	DB    DB
	Repo  string // clone principal
	Root  string // diretório temporário das worktrees de consolidação
	Lease time.Duration
	Now   func() time.Time
	// OnBeforeUpdate é chamado logo antes de gravar a ref (ponto de corrida; usado em testes).
	OnBeforeUpdate func()
}

func (c *Consolidator) now() time.Time {
	if c.Now != nil {
		return c.Now().UTC()
	}
	return time.Now().UTC()
}

func (c *Consolidator) lease() time.Duration {
	if c.Lease > 0 {
		return c.Lease
	}
	return 10 * time.Minute
}

var refPat = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._/-]{0,120}$`)
var shaPat = regexp.MustCompile(`^[0-9a-f]{7,64}$`)

func run(ctx context.Context, dir string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	cmd.Env = []string{"PATH=" + os.Getenv("PATH"), "HOME=" + os.TempDir(), "GIT_TERMINAL_PROMPT=0", "GIT_CONFIG_NOSYSTEM=1",
		"GIT_AUTHOR_NAME=la-fabrique", "GIT_AUTHOR_EMAIL=fabrique@localhost", "GIT_COMMITTER_NAME=la-fabrique", "GIT_COMMITTER_EMAIL=fabrique@localhost"}
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	if err := cmd.Run(); err != nil {
		m := strings.TrimSpace(errb.String())
		if len(m) > 300 {
			m = m[:300]
		}
		return strings.TrimSpace(out.String()), fmt.Errorf("git %s: %v: %s", args[0], err, m)
	}
	return strings.TrimSpace(out.String()), nil
}

func (r Request) validate() error {
	if r.ProjectID == "" || r.FeatureID == "" || !refPat.MatchString(r.DeliveryBranch) || strings.Contains(r.DeliveryBranch, "..") || strings.HasPrefix(r.DeliveryBranch, "lf-tmp/") {
		return fmt.Errorf("%w: projeto, feature e branch de entrega (nunca uma temporária)", persistence.ErrInvalidInput)
	}
	for _, a := range r.Attempts {
		if a.Ref == "" || !refPat.MatchString(a.Branch) || !shaPat.MatchString(a.BaseSHA) {
			return fmt.Errorf("%w: tentativa %q inválida", persistence.ErrInvalidInput, a.Ref)
		}
	}
	return nil
}

// Run consolida. Em conflito devolve ErrConflict com o resultado descrevendo o ponto exato; em corrida
// com outra atualização da branch devolve ErrStale; com outra consolidação da feature, ErrBusy.
func (c *Consolidator) Run(ctx context.Context, req Request) (Result, error) {
	var res Result
	if err := req.validate(); err != nil {
		return res, err
	}
	// Serialização por feature: uma execução ativa por vez (com lease para queda).
	_, _ = c.DB.Exec(ctx, `UPDATE consolidation_runs SET state='failed', detail='lease expirado', finished_at=$2 WHERE feature_id::text=$1 AND state='running' AND lease_until < $2`, req.FeatureID, c.now())
	err := c.DB.QueryRow(ctx, `INSERT INTO consolidation_runs (project_id, feature_id, delivery_branch, lease_until) VALUES ($1,$2,$3,$4) RETURNING id::text`,
		req.ProjectID, req.FeatureID, req.DeliveryBranch, c.now().Add(c.lease())).Scan(&res.RunID)
	if err != nil {
		if strings.Contains(err.Error(), "consolidation_one_running") {
			return res, ErrBusy
		}
		return res, persistence.MapError(err)
	}
	finish := func(state, detail, newHead string) {
		_, _ = c.DB.Exec(context.WithoutCancel(ctx), `UPDATE consolidation_runs SET state=$2, detail=$3, new_head=$4, finished_at=$5 WHERE id::text=$1`, res.RunID, state, detail, newHead, c.now())
	}
	out, err := c.integrate(ctx, req, &res)
	switch {
	case errors.Is(err, ErrConflict):
		finish("conflict", "conflito preservado", "")
	case errors.Is(err, ErrStale):
		finish("stale", err.Error(), "")
	case err != nil:
		finish("failed", err.Error(), "")
	default:
		finish("done", "", out.Head)
	}
	return out, err
}

func (c *Consolidator) integrate(ctx context.Context, req Request, res *Result) (Result, error) {
	branchRef := "refs/heads/" + req.DeliveryBranch
	expected, err := run(ctx, c.Repo, "rev-parse", "--verify", "-q", branchRef)
	if err != nil {
		if !shaPat.MatchString(req.StartSHA) {
			return *res, fmt.Errorf("%w: a branch de entrega não existe e não há SHA de partida", persistence.ErrInvalidInput)
		}
		if expected, err = run(ctx, c.Repo, "rev-parse", req.StartSHA); err != nil {
			return *res, err
		}
		// Cria a branch de entrega condicionada a não existir (corrida segura).
		if _, err := run(ctx, c.Repo, "update-ref", branchRef, expected, strings.Repeat("0", 40)); err != nil {
			return *res, ErrStale
		}
	}
	res.ExpectedSHA = expected
	_, _ = c.DB.Exec(ctx, `UPDATE consolidation_runs SET expected_head=$2 WHERE id::text=$1`, res.RunID, expected)

	tmp := filepath.Join(c.Root, "consolidate-"+res.RunID)
	if err := os.MkdirAll(c.Root, 0o750); err != nil {
		return *res, err
	}
	if _, err := run(ctx, c.Repo, "worktree", "add", "--detach", tmp, expected); err != nil {
		return *res, err
	}
	defer func() {
		_, _ = run(context.WithoutCancel(ctx), c.Repo, "worktree", "remove", "--force", tmp)
	}()

	for _, a := range req.Attempts {
		tip, err := run(ctx, c.Repo, "rev-parse", "--verify", "refs/heads/"+a.Branch)
		if err != nil {
			return *res, fmt.Errorf("a branch da tentativa %s não existe: %w", a.Ref, err)
		}
		list, err := run(ctx, c.Repo, "rev-list", "--reverse", "--no-merges", a.BaseSHA+".."+tip)
		if err != nil {
			return *res, err
		}
		if list == "" {
			continue
		}
		for _, orig := range strings.Split(list, "\n") {
			// Já integrado (repetição após queda): o histórico da entrega prova pelo trailer do cherry-pick.
			if sha, ok := c.findIntegrated(ctx, tmp, orig); ok {
				res.Commits = append(res.Commits, Equivalence{Original: orig, Integrated: sha, Changed: sha != orig, Attempt: a.Ref})
				continue
			}
			head, _ := run(ctx, tmp, "rev-parse", "HEAD")
			parent, _ := run(ctx, tmp, "rev-parse", orig+"^")
			if parent == head {
				// Fast-forward: a SHA não muda e a evidência de revisão segue válida.
				if _, err := run(ctx, tmp, "cherry-pick", "--ff", orig); err != nil {
					return *res, err
				}
			} else if _, err := run(ctx, tmp, "cherry-pick", "-x", orig); err != nil {
				files, _ := run(ctx, tmp, "diff", "--name-only", "--diff-filter=U")
				_, _ = run(ctx, tmp, "cherry-pick", "--abort")
				cf := &Conflict{Attempt: a.Ref, Commit: orig, Files: splitLines(files)}
				res.Conflict = cf
				fj, _ := json.Marshal(cf.Files)
				_, _ = c.DB.Exec(ctx, `INSERT INTO consolidation_conflicts (run_id, feature_id, attempt_ref, commit_sha, files) VALUES ($1,$2,$3,$4,$5)`, res.RunID, req.FeatureID, a.Ref, orig, string(fj))
				return *res, ErrConflict
			}
			now, _ := run(ctx, tmp, "rev-parse", "HEAD")
			res.Commits = append(res.Commits, Equivalence{Original: orig, Integrated: now, Changed: now != orig, Attempt: a.Ref})
		}
	}
	newHead, err := run(ctx, tmp, "rev-parse", "HEAD")
	if err != nil {
		return *res, err
	}
	if c.OnBeforeUpdate != nil {
		c.OnBeforeUpdate()
	}
	if newHead != expected {
		// Atualização condicionada à cabeça esperada: se a branch mudou, nada obsoleto é gravado.
		if _, err := run(ctx, c.Repo, "update-ref", "-m", "consolidação "+res.RunID, branchRef, newHead, expected); err != nil {
			return *res, ErrStale
		}
	}
	res.Head = newHead
	seen := map[string]bool{}
	for _, e := range res.Commits {
		if _, err := c.DB.Exec(ctx, `INSERT INTO consolidated_commits (feature_id, original_sha, integrated_sha, changed, attempt_ref, run_id) VALUES ($1,$2,$3,$4,$5,$6) ON CONFLICT (feature_id, original_sha) DO NOTHING`,
			req.FeatureID, e.Original, e.Integrated, e.Changed, e.Attempt, res.RunID); err != nil {
			return *res, persistence.MapError(err)
		}
		if e.Changed && !seen[e.Attempt] {
			seen[e.Attempt] = true
			res.Revalidate = append(res.Revalidate, e.Attempt)
		}
	}
	if res.Revalidate == nil {
		res.Revalidate = []string{}
	}
	rj, _ := json.Marshal(res.Revalidate)
	if _, err := c.DB.Exec(ctx, `INSERT INTO delivery_candidates (feature_id, head_sha, base_head, run_id, revalidate) VALUES ($1,$2,$3,$4,$5) ON CONFLICT (feature_id, head_sha) DO NOTHING`,
		req.FeatureID, newHead, expected, res.RunID, string(rj)); err != nil {
		return *res, persistence.MapError(err)
	}
	return *res, nil
}

// findIntegrated procura no histórico da worktree o commit que integrou `orig` (ou o próprio).
func (c *Consolidator) findIntegrated(ctx context.Context, dir, orig string) (string, bool) {
	if _, err := run(ctx, dir, "merge-base", "--is-ancestor", orig, "HEAD"); err == nil {
		return orig, true
	}
	out, err := run(ctx, dir, "log", "--format=%H", "--fixed-strings", "--grep=(cherry picked from commit "+orig+")", "HEAD")
	if err == nil && out != "" {
		return strings.Split(out, "\n")[0], true
	}
	return "", false
}

func splitLines(s string) []string {
	var out []string
	for _, l := range strings.Split(s, "\n") {
		if l = strings.TrimSpace(l); l != "" {
			out = append(out, l)
		}
	}
	if out == nil {
		out = []string{}
	}
	return out
}

// Candidate devolve a última cabeça candidata registrada da feature.
func (c *Consolidator) Candidate(ctx context.Context, featureID string) (head string, revalidate []string, err error) {
	var rj []byte
	err = c.DB.QueryRow(ctx, `SELECT head_sha, revalidate FROM delivery_candidates WHERE feature_id::text = $1 ORDER BY created_at DESC LIMIT 1`, featureID).Scan(&head, &rj)
	if err != nil {
		return "", nil, persistence.MapError(err)
	}
	_ = json.Unmarshal(rj, &revalidate)
	return head, revalidate, nil
}

// Descends informa se `head` contém `ancestor` na história (fast-forward possível).
func (c *Consolidator) Descends(ctx context.Context, ancestor, head string) (bool, error) {
	if !shaPat.MatchString(ancestor) || !shaPat.MatchString(head) {
		return false, persistence.ErrInvalidInput
	}
	if _, err := run(ctx, c.Repo, "merge-base", "--is-ancestor", ancestor, head); err != nil {
		if strings.Contains(err.Error(), "exit status 1") {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// Range lista os commits de `from` (exclusivo) até `to` (inclusivo), do mais antigo ao mais novo.
func (c *Consolidator) Range(ctx context.Context, from, to string) ([]string, error) {
	if !shaPat.MatchString(from) || !shaPat.MatchString(to) {
		return nil, persistence.ErrInvalidInput
	}
	out, err := run(ctx, c.Repo, "rev-list", "--reverse", from+".."+to)
	if err != nil {
		return nil, err
	}
	return splitLines(out), nil
}
