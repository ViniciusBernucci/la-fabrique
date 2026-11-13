package worktrees

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
)

// Erros.
var (
	ErrInvalid      = fmt.Errorf("%w: tentativa inválida", persistence.ErrInvalidInput)
	ErrWriterActive = errors.New("há um processo escritor ativo nesta worktree: nada é removido")
)

// DB é o que o gerenciador precisa do PostgreSQL.
type DB interface {
	persistence.DBTX
	persistence.TxStarter
}

// Spec descreve a tentativa.
type Spec struct {
	ProjectID, FeatureID, IssueKey string
	Track                          string // FRONTEND | BACKEND
	Kind                           string // implementation | correction
	Attempt                        int
	BaseSHA                        string
	Paths                          []string // caminhos que a tarefa pode alterar (prefixo de diretório ou arquivo exato)
	Resources                      []string // recursos compartilhados reservados
}

// Attempt é a tentativa provisionada.
type Attempt struct {
	ID, ProjectID, FeatureID, IssueKey, Track, Kind, Branch, Path, BaseSHA, State, ArchiveRef string
	Attempt                                                                                   int
	Paths, Resources                                                                          []string
	WriterActive                                                                              bool
}

// Checkpoint é um ponto recuperável (inclui arquivos não versionados).
type Checkpoint struct {
	Seq   int
	Ref   string
	SHA   string
	Files int
}

// Manager gerencia as worktrees de um repositório.
type Manager struct {
	DB   DB
	Repo string // clone principal (nunca é alterado pela tentativa)
	Root string // diretório das worktrees
}

var safe = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`)
var shaPat = regexp.MustCompile(`^[0-9a-f]{7,64}$`)

func (s Spec) validate() error {
	if s.ProjectID == "" || s.FeatureID == "" || s.Attempt < 1 || !shaPat.MatchString(s.BaseSHA) {
		return fmt.Errorf("%w: projeto, feature, tentativa e SHA de partida são obrigatórios", ErrInvalid)
	}
	if s.Track != "FRONTEND" && s.Track != "BACKEND" {
		return fmt.Errorf("%w: trilha deve ser FRONTEND ou BACKEND", ErrInvalid)
	}
	if s.Kind != "implementation" && s.Kind != "correction" {
		return fmt.Errorf("%w: tipo deve ser implementation ou correction", ErrInvalid)
	}
	for _, p := range s.Paths {
		if !cleanRel(p) {
			return fmt.Errorf("%w: caminho %q fora do repositório", ErrInvalid, p)
		}
	}
	return nil
}

func cleanRel(p string) bool {
	if p == "" || strings.HasPrefix(p, "/") || strings.Contains(p, "\\") || strings.Contains(p, "\x00") {
		return false
	}
	c := path.Clean(p)
	return c != ".." && !strings.HasPrefix(c, "../") && c != "."
}

// Branch é o nome da branch temporária. Nunca coincide com a branch de entrega da feature.
func Branch(featureID, track, kind string, attempt int) string {
	k := "impl"
	if kind == "correction" {
		k = "fix"
	}
	return fmt.Sprintf("lf-tmp/%s/%s-%s-%d", short(featureID), strings.ToLower(track), k, attempt)
}

func short(id string) string {
	id = strings.ReplaceAll(id, "-", "")
	if len(id) > 12 {
		id = id[:12]
	}
	return id
}

const cols = `id::text, project_id::text, feature_id::text, issue_key, track, kind, attempt, branch, path, base_sha, paths, resources, state, writer_active, archive_ref`

func scan(r interface{ Scan(...any) error }) (Attempt, error) {
	var a Attempt
	var p, rs []byte
	if err := r.Scan(&a.ID, &a.ProjectID, &a.FeatureID, &a.IssueKey, &a.Track, &a.Kind, &a.Attempt, &a.Branch, &a.Path, &a.BaseSHA, &p, &rs, &a.State, &a.WriterActive, &a.ArchiveRef); err != nil {
		return a, err
	}
	_ = json.Unmarshal(p, &a.Paths)
	_ = json.Unmarshal(rs, &a.Resources)
	return a, nil
}

// Provision cria (ou recupera) a worktree da tentativa na SHA de partida. Repetir devolve a mesma
// tentativa; se o diretório sumiu (queda), é recriado a partir da branch preservada.
func (m *Manager) Provision(ctx context.Context, s Spec) (Attempt, error) {
	if err := s.validate(); err != nil {
		return Attempt{}, err
	}
	if _, err := git(ctx, m.Repo, nil, "", "cat-file", "-e", s.BaseSHA+"^{commit}"); err != nil {
		return Attempt{}, fmt.Errorf("%w: a SHA de partida não existe no repositório", ErrInvalid)
	}
	full, err := git(ctx, m.Repo, nil, "", "rev-parse", s.BaseSHA)
	if err != nil {
		return Attempt{}, err
	}
	branch := Branch(s.FeatureID, s.Track, s.Kind, s.Attempt)
	dir := filepath.Join(m.Root, short(s.ProjectID), short(s.FeatureID), fmt.Sprintf("%s-%s-%d", strings.ToLower(s.Track), s.Kind, s.Attempt))
	paths, _ := json.Marshal(nonNil(s.Paths))
	res, _ := json.Marshal(nonNil(s.Resources))
	row := m.DB.QueryRow(ctx, `INSERT INTO worktree_attempts (project_id, feature_id, issue_key, track, kind, attempt, branch, path, base_sha, paths, resources)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		ON CONFLICT (feature_id, track, kind, attempt) DO UPDATE SET issue_key = worktree_attempts.issue_key
		RETURNING `+cols, s.ProjectID, s.FeatureID, s.IssueKey, s.Track, s.Kind, s.Attempt, branch, dir, full, paths, res)
	a, err := scan(row)
	if err != nil {
		return Attempt{}, persistence.MapError(err)
	}
	if a.State != "active" {
		return a, fmt.Errorf("%w: a tentativa já foi encerrada", ErrInvalid)
	}
	if _, err := os.Stat(a.Path); err == nil {
		return a, nil
	}
	if err := os.MkdirAll(filepath.Dir(a.Path), 0o750); err != nil {
		return a, err
	}
	// Recria: a branch já existe se houve trabalho antes da queda; senão nasce da SHA de partida.
	if _, err := git(ctx, m.Repo, nil, "", "rev-parse", "--verify", "refs/heads/"+a.Branch); err == nil {
		_, _ = git(ctx, m.Repo, nil, "", "worktree", "prune")
		_, err = git(ctx, m.Repo, nil, "", "worktree", "add", a.Path, a.Branch)
		return a, err
	}
	_, err = git(ctx, m.Repo, nil, "", "worktree", "add", "-b", a.Branch, a.Path, a.BaseSHA)
	return a, err
}

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

// Get lê uma tentativa.
func (m *Manager) Get(ctx context.Context, id string) (Attempt, error) {
	a, err := scan(m.DB.QueryRow(ctx, `SELECT `+cols+` FROM worktree_attempts WHERE id::text = $1`, id))
	return a, persistence.MapError(err)
}

// SetWriter marca se há processo escritor ativo (a limpeza recusa enquanto houver).
func (m *Manager) SetWriter(ctx context.Context, id string, active bool) error {
	_, err := m.DB.Exec(ctx, `UPDATE worktree_attempts SET writer_active = $2 WHERE id::text = $1 AND state = 'active'`, id, active)
	return persistence.MapError(err)
}

// Checkpoint grava o estado atual da worktree (versionado e não versionado) numa ref própria, sem
// tocar no índice nem na branch da tentativa. Se nada mudou desde o último, devolve o mesmo.
func (m *Manager) Checkpoint(ctx context.Context, id string) (Checkpoint, error) {
	a, err := m.Get(ctx, id)
	if err != nil {
		return Checkpoint{}, err
	}
	if a.State != "active" {
		return Checkpoint{}, fmt.Errorf("%w: tentativa encerrada", ErrInvalid)
	}
	idx := filepath.Join(os.TempDir(), "lf-index-"+a.ID)
	defer os.Remove(idx)
	env := []string{"GIT_INDEX_FILE=" + idx}
	if _, err := git(ctx, a.Path, env, "", "read-tree", "HEAD"); err != nil {
		return Checkpoint{}, err
	}
	if _, err := git(ctx, a.Path, env, "", "add", "-A"); err != nil {
		return Checkpoint{}, err
	}
	tree, err := git(ctx, a.Path, env, "", "write-tree")
	if err != nil {
		return Checkpoint{}, err
	}
	head, err := git(ctx, a.Path, nil, "", "rev-parse", "HEAD")
	if err != nil {
		return Checkpoint{}, err
	}
	var seq int
	var lastSHA string
	_ = m.DB.QueryRow(ctx, `SELECT seq, sha FROM worktree_checkpoints WHERE attempt_id::text = $1 ORDER BY seq DESC LIMIT 1`, id).Scan(&seq, &lastSHA)
	if lastSHA != "" {
		if lastTree, e := git(ctx, a.Path, nil, "", "rev-parse", lastSHA+"^{tree}"); e == nil && lastTree == tree {
			var cp Checkpoint
			err := m.DB.QueryRow(ctx, `SELECT seq, ref, sha, files FROM worktree_checkpoints WHERE attempt_id::text = $1 AND seq = $2`, id, seq).Scan(&cp.Seq, &cp.Ref, &cp.SHA, &cp.Files)
			return cp, err
		}
	}
	commit, err := git(ctx, a.Path, nil, "", "commit-tree", tree, "-p", head, "-m", fmt.Sprintf("checkpoint %d da tentativa %s", seq+1, a.ID))
	if err != nil {
		return Checkpoint{}, err
	}
	ref := fmt.Sprintf("refs/lf/checkpoints/%s/%d", a.ID, seq+1)
	if _, err := git(ctx, a.Path, nil, "", "update-ref", ref, commit); err != nil {
		return Checkpoint{}, err
	}
	names, _ := git(ctx, a.Path, nil, "", "diff", "--name-only", head, commit)
	files := 0
	if names != "" {
		files = len(strings.Split(names, "\n"))
	}
	cp := Checkpoint{Seq: seq + 1, Ref: ref, SHA: commit, Files: files}
	if _, err := m.DB.Exec(ctx, `INSERT INTO worktree_checkpoints (attempt_id, seq, ref, sha, files) VALUES ($1,$2,$3,$4,$5)`, id, cp.Seq, ref, commit, files); err != nil {
		return Checkpoint{}, persistence.MapError(err)
	}
	return cp, nil
}

// Changed lista os arquivos alterados em relação à SHA de partida (versionados e não versionados).
func (m *Manager) Changed(ctx context.Context, a Attempt) ([]string, error) {
	set := map[string]bool{}
	for _, args := range [][]string{{"diff", "--name-only", a.BaseSHA}, {"ls-files", "--others", "--exclude-standard"}} {
		out, err := git(ctx, a.Path, nil, "", args...)
		if err != nil {
			return nil, err
		}
		for _, l := range strings.Split(out, "\n") {
			if l = strings.TrimSpace(l); l != "" {
				set[l] = true
			}
		}
	}
	out := make([]string, 0, len(set))
	for f := range set {
		out = append(out, f)
	}
	sort.Strings(out)
	return out, nil
}

// Violations devolve os arquivos alterados fora do conjunto de caminhos do plano.
func (m *Manager) Violations(ctx context.Context, id string) ([]string, error) {
	a, err := m.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	changed, err := m.Changed(ctx, a)
	if err != nil {
		return nil, err
	}
	var bad []string
	for _, f := range changed {
		if !within(f, a.Paths) {
			bad = append(bad, f)
		}
	}
	return bad, nil
}

func within(file string, allowed []string) bool {
	for _, p := range allowed {
		p = strings.TrimSuffix(path.Clean(p), "/")
		if file == p || strings.HasPrefix(file, p+"/") {
			return true
		}
	}
	return false
}

// Release remove a worktree DESTA tentativa depois de preservar tudo: um checkpoint do que está no
// diretório e uma ref de arquivo apontando para a ponta da branch (que então é apagada). Recusa se há
// escritor ativo. Não toca em nenhuma outra tentativa.
func (m *Manager) Release(ctx context.Context, id string) error {
	a, err := m.Get(ctx, id)
	if err != nil {
		return err
	}
	if a.State == "released" {
		return nil
	}
	if a.WriterActive {
		return ErrWriterActive
	}
	if _, err := os.Stat(a.Path); err == nil {
		if _, err := m.Checkpoint(ctx, id); err != nil {
			return fmt.Errorf("preservar antes de limpar: %w", err)
		}
		tip, err := git(ctx, a.Path, nil, "", "rev-parse", "HEAD")
		if err != nil {
			return err
		}
		archive := "refs/lf/archive/" + a.ID
		if _, err := git(ctx, m.Repo, nil, "", "update-ref", archive, tip); err != nil {
			return err
		}
		if _, err := git(ctx, m.Repo, nil, "", "worktree", "remove", "--force", a.Path); err != nil {
			return err
		}
		_, _ = git(ctx, m.Repo, nil, "", "branch", "-D", a.Branch)
		_, err = m.DB.Exec(ctx, `UPDATE worktree_attempts SET state = 'released', released_at = now(), archive_ref = $2 WHERE id::text = $1`, id, archive)
		return persistence.MapError(err)
	}
	_, err = m.DB.Exec(ctx, `UPDATE worktree_attempts SET state = 'released', released_at = now() WHERE id::text = $1`, id)
	return persistence.MapError(err)
}

// Commits lista os commits da tentativa desde a SHA de partida, do mais antigo ao mais novo.
func (m *Manager) Commits(ctx context.Context, a Attempt) ([]string, error) {
	out, err := git(ctx, a.Path, nil, "", "rev-list", "--reverse", a.BaseSHA+"..HEAD")
	if err != nil {
		return nil, err
	}
	var cs []string
	for _, l := range strings.Split(out, "\n") {
		if l = strings.TrimSpace(l); l != "" {
			cs = append(cs, l)
		}
	}
	return cs, nil
}

// Orphans lista as tentativas ativas sem executor vivo (`live` tem os IDs com escritor em andamento).
// Não remove nada: a coleta preserva (Release faz checkpoint e arquivo antes de limpar).
func (m *Manager) Orphans(ctx context.Context, projectID string, live map[string]bool) ([]Attempt, error) {
	rows, err := m.DB.Query(ctx, `SELECT `+cols+` FROM worktree_attempts WHERE project_id = $1 AND state = 'active' ORDER BY created_at`, projectID)
	if err != nil {
		return nil, persistence.MapError(err)
	}
	defer rows.Close()
	var out []Attempt
	for rows.Next() {
		a, err := scan(rows)
		if err != nil {
			return nil, err
		}
		if !live[a.ID] {
			out = append(out, a)
		}
	}
	return out, rows.Err()
}
