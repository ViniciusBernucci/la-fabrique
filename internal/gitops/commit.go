package gitops

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/ViniciusBernucci/le-fabrique/internal/checkpoints"
)

// Erros do commit.
var (
	ErrChecksNotPassed = errors.New("commit recusado: os checks da tarefa não passaram")
	ErrNothingToCommit = errors.New("nada a commitar: o conteúdo é igual ao do commit pai")
	ErrBadCommit       = errors.New("pedido de commit inválido")
	ErrWorkflowChange  = errors.New("a mudança toca .github/ (workflows): exige aprovação explícita")
)

// BranchPrefix marca as branches da fábrica.
const BranchPrefix = "lf/"

var (
	featureIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9-]{0,63}$`)
	taskKeyPattern   = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9:._/-]{0,100}$`)
	hex64            = regexp.MustCompile(`^[0-9a-f]{64}$`)
	checkName        = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9 ._:/-]{0,60}$`)
)

// FeatureBranch dá a branch estável da feature: lf/<feature>-a<tentativa>.
func FeatureBranch(featureID string, attempt int) (string, error) {
	if !featureIDPattern.MatchString(featureID) || attempt < 1 || attempt > 99 {
		return "", fmt.Errorf("%w: feature ou tentativa", ErrBadCommit)
	}
	return fmt.Sprintf("%sf-%s-a%d", BranchPrefix, featureID, attempt), nil
}

// ValidFeatureBranch confere que o nome é uma branch da fábrica bem formada.
func ValidFeatureBranch(name string) error {
	if !strings.HasPrefix(name, BranchPrefix) || validRefName(name) != nil || strings.Count(name, "/") != 1 {
		return fmt.Errorf("%w: %q", ErrBranchForbidden, name)
	}
	return nil
}

// Task é a tarefa que o commit entrega.
type Task struct {
	Key   string // chave estável da tarefa técnica
	Title string // uma linha
}

// Checks é o resultado dos checks da tarefa. O commit só nasce com Passed.
type Checks struct {
	Passed     bool
	Names      []string // ex.: lint, test
	ReportHash string   // SHA-256 (hex) do relatório dos checks
}

// Identity é a identidade git da fábrica (autor e committer).
type Identity struct{ Name, Email string }

// DefaultIdentity é a identidade da fábrica.
var DefaultIdentity = Identity{Name: "La fabrique", Email: "fabrique@users.noreply.invalid"}

// CommitRequest pede o commit de um snapshot do workspace como filho de Parent.
type CommitRequest struct {
	Branch   string
	Parent   string // SHA completo do commit pai (ponta da branch ou do ramo base)
	Task     Task
	Checks   Checks
	Snapshot checkpoints.Snapshot
	At       time.Time // data do commit (estável entre repetições → SHA idêntico)
	Author   Identity
}

// CommitInfo descreve o commit criado.
type CommitInfo struct {
	SHA       string
	Tree      string
	Parent    string
	Sensitive []string // caminhos de .github/ alterados em relação ao pai
}

func sane(s string) bool {
	return !strings.ContainsAny(s, "\x00\r\n<>")
}

func (r CommitRequest) validate() error {
	bad := func(why string) error { return fmt.Errorf("%w: %s", ErrBadCommit, why) }
	if err := ValidFeatureBranch(r.Branch); err != nil {
		return err
	}
	if !shaPattern.MatchString(r.Parent) {
		return bad("commit pai")
	}
	if !taskKeyPattern.MatchString(r.Task.Key) {
		return bad("chave da tarefa")
	}
	if len(r.Task.Title) > 120 || !sane(r.Task.Title) {
		return bad("título da tarefa")
	}
	if !r.Checks.Passed {
		return ErrChecksNotPassed
	}
	if !hex64.MatchString(r.Checks.ReportHash) || len(r.Checks.Names) == 0 || len(r.Checks.Names) > 20 {
		return fmt.Errorf("%w: relatório e nomes dos checks são obrigatórios", ErrChecksNotPassed)
	}
	for _, n := range r.Checks.Names {
		if !checkName.MatchString(n) || strings.Contains(n, ",") {
			return bad("nome de check")
		}
	}
	if r.At.IsZero() {
		return bad("data do commit")
	}
	if r.Author.Name == "" {
		return nil
	}
	if !sane(r.Author.Name) || !sane(r.Author.Email) {
		return bad("identidade")
	}
	return nil
}

func quote(p string) string {
	return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(p) + `"`
}

func hasGitComponent(p string) bool {
	for _, c := range strings.Split(p, "/") {
		if strings.EqualFold(c, ".git") {
			return true
		}
	}
	return false
}

// message monta a mensagem com os trailers que ligam o commit à tarefa e aos checks.
func (r CommitRequest) message() string {
	title := strings.TrimSpace(r.Task.Title)
	if title == "" {
		title = r.Task.Key
	}
	names := append([]string{}, r.Checks.Names...)
	sort.Strings(names)
	return fmt.Sprintf("%s\n\nLf-Task: %s\nLf-Checks: %s\nLf-Checks-Report: %s\nLf-Tree: %s\n",
		title, r.Task.Key, strings.Join(names, ","), r.Checks.ReportHash, r.Snapshot.Manifest.TreeHash)
}

// Commit cria o commit da tarefa com `git fast-import`: o conteúdo vem do manifesto
// e dos objetos do snapshot, sem extrair nenhum arquivo no host. Recusa checks que
// não passaram, pai desconhecido, caminhos com `.git` e conteúdo igual ao do pai.
// O commit é determinístico (data e identidade fixas): repetir dá o mesmo SHA.
func (w *Workdir) Commit(ctx context.Context, req CommitRequest) (CommitInfo, error) {
	if err := req.validate(); err != nil {
		return CommitInfo{}, err
	}
	m := req.Snapshot.Manifest
	if err := m.Validate(); err != nil {
		return CommitInfo{}, err
	}
	if _, err := w.rev(ctx, req.Parent); err != nil {
		return CommitInfo{}, fmt.Errorf("%w: o commit pai não existe no clone", ErrBadCommit)
	}
	author := req.Author
	if author.Name == "" {
		author = DefaultIdentity
	}
	var buf bytes.Buffer
	data := func(b []byte) { fmt.Fprintf(&buf, "data %d\n", len(b)); buf.Write(b); buf.WriteByte('\n') }
	msg := req.message()
	stamp := fmt.Sprintf("%d +0000", req.At.UTC().Unix())
	tmp := "refs/lf/tmp/" + req.Parent[:12] + "-" + m.TreeHash[:12]
	fmt.Fprintf(&buf, "commit %s\nauthor %s <%s> %s\ncommitter %s <%s> %s\n", tmp, author.Name, author.Email, stamp, author.Name, author.Email, stamp)
	data([]byte(msg))
	fmt.Fprintf(&buf, "from %s\ndeleteall\n", req.Parent)
	for _, f := range m.Files {
		if hasGitComponent(f.Path) {
			return CommitInfo{}, fmt.Errorf("%w: caminho com .git (%q)", ErrBadCommit, f.Path)
		}
		if strings.ContainsFunc(f.Path, func(r rune) bool { return r < 0x20 || r == 0x7f }) {
			return CommitInfo{}, fmt.Errorf("%w: caractere de controle no caminho", ErrBadCommit)
		}
		switch f.Kind {
		case checkpoints.KindFile:
			blob, ok := req.Snapshot.Objects[f.SHA256]
			if !ok {
				return CommitInfo{}, fmt.Errorf("%w: conteúdo de %q ausente", ErrBadCommit, f.Path)
			}
			mode := "100644"
			if f.Mode&0o111 != 0 {
				mode = "100755"
			}
			fmt.Fprintf(&buf, "M %s inline %s\n", mode, quote(f.Path))
			data(blob)
		case checkpoints.KindSymlink:
			fmt.Fprintf(&buf, "M 120000 inline %s\n", quote(f.Path))
			data([]byte(f.Link))
		}
	}
	buf.WriteString("done\n")
	if _, err := w.git(ctx, buf.Bytes(), "fast-import", "--quiet", "--done", "--date-format=raw"); err != nil {
		return CommitInfo{}, err
	}
	sha, err := w.rev(ctx, tmp)
	if err != nil {
		return CommitInfo{}, err
	}
	_, _ = w.git(ctx, nil, "update-ref", "-d", tmp)
	tree, err := w.git(ctx, nil, "rev-parse", "--verify", sha+"^{tree}")
	if err != nil {
		return CommitInfo{}, err
	}
	parentTree, err := w.git(ctx, nil, "rev-parse", "--verify", req.Parent+"^{tree}")
	if err != nil {
		return CommitInfo{}, err
	}
	if strings.TrimSpace(tree.Stdout) == strings.TrimSpace(parentTree.Stdout) {
		return CommitInfo{}, ErrNothingToCommit
	}
	if _, err := w.git(ctx, nil, "update-ref", "refs/lf/work/"+strings.TrimPrefix(req.Branch, BranchPrefix), sha); err != nil {
		return CommitInfo{}, err
	}
	info := CommitInfo{SHA: sha, Tree: strings.TrimSpace(tree.Stdout), Parent: req.Parent}
	changed, err := w.git(ctx, nil, "diff", "--name-only", req.Parent, sha)
	if err != nil {
		return CommitInfo{}, err
	}
	for _, p := range strings.Split(strings.TrimSpace(changed.Stdout), "\n") {
		if strings.HasPrefix(p, ".github/") && len(info.Sensitive) < 50 {
			info.Sensitive = append(info.Sensitive, p)
		}
	}
	return info, nil
}

// Trailers são os campos Lf-* lidos de um commit.
type Trailers struct {
	Task, Checks, Report, Tree string
}

// ReadTrailers lê os trailers do commit (a prova de que ele corresponde à tarefa e aos checks).
func (w *Workdir) ReadTrailers(ctx context.Context, sha string) (Trailers, error) {
	if !shaPattern.MatchString(sha) {
		return Trailers{}, fmt.Errorf("%w: SHA", ErrBadCommit)
	}
	res, err := w.git(ctx, nil, "log", "-1", "--format=%(trailers:only,unfold)", sha)
	if err != nil {
		return Trailers{}, err
	}
	var t Trailers
	for _, l := range strings.Split(res.Stdout, "\n") {
		k, v, ok := strings.Cut(l, ": ")
		if !ok {
			continue
		}
		switch k {
		case "Lf-Task":
			t.Task = v
		case "Lf-Checks":
			t.Checks = v
		case "Lf-Checks-Report":
			t.Report = v
		case "Lf-Tree":
			t.Tree = v
		}
	}
	return t, nil
}

// Matches confere se o commit corresponde à tarefa e aos checks informados.
func (t Trailers) Matches(task Task, c Checks, treeHash string) bool {
	names := append([]string{}, c.Names...)
	sort.Strings(names)
	return t.Task == task.Key && t.Report == c.ReportHash && t.Checks == strings.Join(names, ",") && t.Tree == treeHash
}
