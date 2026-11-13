// Package envconfig versiona, valida, aprova e reutiliza a configuração do ambiente de teste/visual por
// projeto (AJ08-04). Estados: proposta → em validação → validada (e aprovada) | inválida | requer
// revalidação. Só uma versão validada com evidência de prontidão, acesso do navegador e limpeza pode ser
// promovida; mudança relevante de comandos, serviços ou permissões gera diagnóstico e nova proposta, e a
// última versão validada segue disponível. Segredos aparecem só como referência `env:NOME`.
package envconfig

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/ViniciusBernucci/le-fabrique/internal/browserenv"
	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
)

// Estados.
const (
	StateProposal     = "proposal"
	StateValidating   = "validating"
	StateValidated    = "validated"
	StateInvalid      = "invalid"
	StateRevalidation = "needs_revalidation"
)

// Erros.
var (
	ErrNotValidated = errors.New("só uma versão validada com evidência pode ser aprovada")
	ErrNoActive     = errors.New("o projeto ainda não tem configuração de ambiente aprovada")
)

// Service é um serviço do ambiente.
type Service struct {
	Name      string   `json:"name"`
	Kind      string   `json:"kind"`
	Image     string   `json:"image,omitempty"`
	Build     string   `json:"build,omitempty"`
	Command   string   `json:"command,omitempty"`
	Ports     []int    `json:"ports"`
	Readiness string   `json:"readiness"`
	DependsOn []string `json:"depends_on"`
}

// Credential é uma credencial de teste por referência.
type Credential struct {
	Name string `json:"name"`
	Ref  string `json:"ref"`
}

// Config é o conteúdo versionado.
type Config struct {
	Services      []Service    `json:"services"`
	Install       string       `json:"install"`
	Start         string       `json:"start"`
	Build         string       `json:"build"`
	Test          string       `json:"test"`
	DataSetup     string       `json:"data_setup"`
	DataCleanup   string       `json:"data_cleanup"`
	Credentials   []Credential `json:"credentials"`
	BrowserTarget string       `json:"browser_target"`
	Notes         string       `json:"notes,omitempty"` // não material
}

// Problem é um motivo de recusa na validação estática.
type Problem struct{ Field, Message string }

func (p Problem) String() string { return p.Field + ": " + p.Message }

var (
	refPat    = regexp.MustCompile(`^env:[A-Z][A-Z0-9_]{1,63}$`)
	dangerous = []string{"docker.sock", "--privileged", "sudo ", "| sh", "| bash", "curl ", "wget ", "--network host", "--net=host", "--pid=host"}
	prodWords = regexp.MustCompile(`(?i)(prod(uction|ucao|ução)?|\.com\b|\.io\b|\.net\b|\.org\b|amazonaws|rds\.)`)
)

// Check valida schema, dependências e política. Vazio = apta à validação isolada.
func Check(c Config) []Problem {
	var out []Problem
	add := func(f, m string) { out = append(out, Problem{f, m}) }
	if strings.TrimSpace(c.Start) == "" {
		add("start", "comando para subir a aplicação é obrigatório")
	}
	if strings.TrimSpace(c.Test) == "" {
		add("test", "comando de testes é obrigatório")
	}
	if len(c.Services) == 0 {
		add("services", "descreva ao menos um serviço")
	}
	names := map[string]bool{}
	for _, s := range c.Services {
		names[s.Name] = true
	}
	for _, s := range c.Services {
		if strings.TrimSpace(s.Name) == "" {
			add("services", "serviço sem nome")
			continue
		}
		if strings.TrimSpace(s.Readiness) == "" && (s.Kind == "frontend" || s.Kind == "api" || s.Kind == "database") {
			add("services."+s.Name+".readiness", "prontidão obrigatória para o serviço")
		}
		for _, d := range s.DependsOn {
			if !names[d] {
				add("services."+s.Name+".depends_on", "dependência desconhecida: "+d)
			}
		}
	}
	if o, err := browserenv.ParseOrigin(c.BrowserTarget); err != nil {
		add("browser_target", "alvo do navegador inválido (use o nome do serviço na rede interna): "+err.Error())
	} else if host := strings.Split(o.Host, ":")[0]; !names[host] {
		add("browser_target", "o alvo do navegador deve ser um serviço declarado: "+host)
	}
	for _, f := range []struct{ name, v string }{{"install", c.Install}, {"start", c.Start}, {"build", c.Build}, {"test", c.Test}, {"data_setup", c.DataSetup}, {"data_cleanup", c.DataCleanup}} {
		for _, d := range dangerous {
			if strings.Contains(f.v, d) {
				add(f.name, "comando fora da política (contém "+strings.TrimSpace(d)+")")
			}
		}
		if prodWords.MatchString(f.v) {
			add(f.name, "o comando menciona destino de produção ou externo: o ambiente de teste é privado")
		}
	}
	for _, cr := range c.Credentials {
		if !refPat.MatchString(cr.Ref) {
			add("credentials."+cr.Name, "credencial deve ser referência env:NOME, nunca o valor")
		}
	}
	return out
}

// Hashes: o completo identifica a versão; o material cobre o que, ao mudar, exige revalidação.
func hashes(c Config) (content, material string) {
	b, _ := json.Marshal(c)
	h := sha256.Sum256(b)
	m := c
	m.Notes = ""
	svc := append([]Service{}, m.Services...)
	sort.Slice(svc, func(i, j int) bool { return svc[i].Name < svc[j].Name })
	m.Services = svc
	cr := append([]Credential{}, m.Credentials...)
	sort.Slice(cr, func(i, j int) bool { return cr[i].Name < cr[j].Name })
	m.Credentials = cr
	mb, _ := json.Marshal(m)
	mh := sha256.Sum256(mb)
	return hex.EncodeToString(h[:]), hex.EncodeToString(mh[:])
}

// Evidence é a prova da validação isolada.
type Evidence struct {
	SHA         string          `json:"sha"`
	Ready       map[string]bool `json:"ready"`
	BrowserOK   bool            `json:"browser_ok"` // acesso do navegador MCP à aplicação
	CleanupOK   bool            `json:"cleanup_ok"`
	Diagnostics []string        `json:"diagnostics,omitempty"`
}

// Runner sobe o ambiente isolado com a configuração candidata na revisão e devolve a evidência.
// Sempre recolhe o que criou. (A implementação real usa browserenv + o sandbox; testes usam um fake.)
type Runner interface {
	Run(ctx context.Context, c Config, sha string) (Evidence, error)
}

// DB é o que o armazenamento precisa do PostgreSQL.
type DB interface {
	persistence.DBTX
	persistence.TxStarter
}

// Store persiste as versões.
type Store struct {
	DB  DB
	Now func() time.Time
}

func (s *Store) now() time.Time {
	if s.Now != nil {
		return s.Now().UTC()
	}
	return time.Now().UTC()
}

// Version é uma versão da configuração.
type Version struct {
	ID           string     `json:"id"`
	ProjectID    string     `json:"project_id"`
	Version      int        `json:"version"`
	State        string     `json:"state"`
	Config       Config     `json:"config"`
	Hash         string     `json:"content_hash"`
	MaterialHash string     `json:"material_hash"`
	BaseSHA      string     `json:"base_sha"`
	Evidence     *Evidence  `json:"evidence,omitempty"`
	Diagnostic   string     `json:"diagnostic"`
	ApprovedBy   string     `json:"approved_by"`
	ValidatedAt  *time.Time `json:"validated_at"`
	Active       bool       `json:"active"`
}

const cols = `c.id::text, c.project_id::text, c.version, c.state, c.content, c.content_hash, c.material_hash, c.base_sha, c.evidence, c.diagnostic, c.approved_by, c.validated_at, (a.config_id = c.id)`
const from = ` FROM env_configs c LEFT JOIN env_active a ON a.project_id = c.project_id `

func scan(r pgx.Row) (Version, error) {
	var v Version
	var content, ev []byte
	var active *bool
	if err := r.Scan(&v.ID, &v.ProjectID, &v.Version, &v.State, &content, &v.Hash, &v.MaterialHash, &v.BaseSHA, &ev, &v.Diagnostic, &v.ApprovedBy, &v.ValidatedAt, &active); err != nil {
		return v, persistence.MapError(err)
	}
	_ = json.Unmarshal(content, &v.Config)
	if len(ev) > 0 {
		v.Evidence = &Evidence{}
		_ = json.Unmarshal(ev, v.Evidence)
	}
	v.Active = active != nil && *active
	return v, nil
}

// Save grava uma proposta ou edição como nova versão. Se a mudança em relação à versão aprovada é só
// não material (ex.: notas), a nova versão herda a validação; se é material, fica `needs_revalidation`
// com o diagnóstico do que mudou e a aprovada continua valendo.
func (s *Store) Save(ctx context.Context, projectID string, c Config, baseSHA string) (Version, error) {
	if c.Services == nil {
		c.Services = []Service{}
	}
	if c.Credentials == nil {
		c.Credentials = []Credential{}
	}
	content, material := hashes(c)
	cj, _ := json.Marshal(c)
	var out Version
	err := persistence.WithTx(ctx, s.DB, func(tx pgx.Tx) error {
		var next int
		if err := tx.QueryRow(ctx, `SELECT COALESCE(MAX(version),0)+1 FROM env_configs WHERE project_id = $1`, projectID).Scan(&next); err != nil {
			return err
		}
		state, diag := StateProposal, ""
		var actMat, actEv string
		var actVer int
		err := tx.QueryRow(ctx, `SELECT c.material_hash, c.version, COALESCE(c.evidence::text,'') FROM env_active a JOIN env_configs c ON c.id = a.config_id WHERE a.project_id = $1`, projectID).Scan(&actMat, &actVer, &actEv)
		hasActive := err == nil
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		var inherit []byte
		if hasActive {
			if actMat != material {
				state, diag = StateRevalidation, fmt.Sprintf("mudança relevante em relação à versão aprovada v%d (comandos, serviços ou permissões): revalide antes de aprovar", actVer)
			} else {
				state = StateValidated
				inherit = []byte(actEv)
			}
		}
		var ev any
		if len(inherit) > 0 {
			ev = string(inherit)
		}
		row := tx.QueryRow(ctx, `INSERT INTO env_configs (project_id, version, state, content, content_hash, material_hash, base_sha, diagnostic, evidence, validated_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9::jsonb, CASE WHEN $3 = 'validated' THEN $10::timestamptz END)
			RETURNING id::text`, projectID, next, state, string(cj), content, material, baseSHA, diag, ev, s.now())
		var id string
		if err := row.Scan(&id); err != nil {
			return err
		}
		v, err := scan(tx.QueryRow(ctx, `SELECT `+cols+from+`WHERE c.id::text = $1`, id))
		out = v
		return err
	})
	return out, persistence.MapError(err)
}

// Get lê uma versão.
func (s *Store) Get(ctx context.Context, projectID string, version int) (Version, error) {
	return scan(s.DB.QueryRow(ctx, `SELECT `+cols+from+`WHERE c.project_id::text = $1 AND c.version = $2`, projectID, version))
}

// Validate valida a versão: checagem estática e execução isolada na revisão `sha`. Falha de readiness,
// de acesso do navegador ou de limpeza torna a versão inválida (e impede promovê-la).
func (s *Store) Validate(ctx context.Context, projectID string, version int, sha string, r Runner) (Version, error) {
	v, err := s.Get(ctx, projectID, version)
	if err != nil {
		return v, err
	}
	finish := func(state, diag string, ev *Evidence) (Version, error) {
		var evj any
		if ev != nil {
			b, _ := json.Marshal(ev)
			evj = string(b)
		}
		_, err := s.DB.Exec(ctx, `UPDATE env_configs SET state=$2, diagnostic=$3, evidence=$4::jsonb, base_sha=$5, validated_at = CASE WHEN $2='validated' THEN $6::timestamptz END WHERE id::text=$1`,
			v.ID, state, diag, evj, sha, s.now())
		if err != nil {
			return v, persistence.MapError(err)
		}
		return s.Get(ctx, projectID, version)
	}
	if ps := Check(v.Config); len(ps) > 0 {
		var m []string
		for _, p := range ps {
			m = append(m, p.String())
		}
		return finish(StateInvalid, strings.Join(m, "; "), nil)
	}
	if _, err := s.DB.Exec(ctx, `UPDATE env_configs SET state='validating', base_sha=$2 WHERE id::text=$1`, v.ID, sha); err != nil {
		return v, persistence.MapError(err)
	}
	ev, err := r.Run(ctx, v.Config, sha)
	if err != nil {
		return finish(StateInvalid, "a validação isolada falhou: "+err.Error(), &ev)
	}
	var bad []string
	for _, svc := range v.Config.Services {
		if !ev.Ready[svc.Name] && (svc.Kind == "frontend" || svc.Kind == "api" || svc.Kind == "database") {
			bad = append(bad, "o serviço "+svc.Name+" não ficou pronto")
		}
	}
	if !ev.BrowserOK {
		bad = append(bad, "o navegador MCP não acessou a aplicação")
	}
	if !ev.CleanupOK {
		bad = append(bad, "a limpeza dos recursos temporários não foi comprovada")
	}
	if ev.SHA != sha {
		bad = append(bad, "a evidência não corresponde à revisão analisada")
	}
	if len(bad) > 0 {
		return finish(StateInvalid, strings.Join(bad, "; "), &ev)
	}
	return finish(StateValidated, "", &ev)
}

// Approve promove uma versão validada: as próximas execuções passam a fixá-la.
func (s *Store) Approve(ctx context.Context, projectID string, version int, by string) (Version, error) {
	v, err := s.Get(ctx, projectID, version)
	if err != nil {
		return v, err
	}
	if v.State != StateValidated || v.Evidence == nil {
		return v, ErrNotValidated
	}
	err = persistence.WithTx(ctx, s.DB, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `UPDATE env_configs SET approved_by = $2 WHERE id::text = $1`, v.ID, by); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `INSERT INTO env_active (project_id, config_id, approved_at) VALUES ($1,$2,$3)
			ON CONFLICT (project_id) DO UPDATE SET config_id = EXCLUDED.config_id, approved_at = EXCLUDED.approved_at`, projectID, v.ID, s.now())
		return err
	})
	if err != nil {
		return v, persistence.MapError(err)
	}
	return s.Get(ctx, projectID, version)
}

// Preflight confere, na revisão que será testada, se a configuração ainda se aplica (ex.: manifestos
// citados pela configuração não mudaram de forma relevante).
type Preflight func(c Config, sha string) error

// State é a visão do painel: a última versão validada/aprovada e a candidata.
type State struct {
	Active    *Version `json:"active"`
	Candidate *Version `json:"candidate"`
}

// View devolve a aprovada e a candidata mais recente (distintas no painel).
func (s *Store) View(ctx context.Context, projectID string) (State, error) {
	var st State
	if a, err := scan(s.DB.QueryRow(ctx, `SELECT `+cols+from+`WHERE a.project_id::text = $1 AND a.config_id = c.id`, projectID)); err == nil {
		st.Active = &a
	} else if !errors.Is(err, persistence.ErrNotFound) {
		return st, err
	}
	if c, err := scan(s.DB.QueryRow(ctx, `SELECT `+cols+from+`WHERE c.project_id::text = $1 AND (a.config_id IS NULL OR a.config_id <> c.id) AND c.version > COALESCE((SELECT v.version FROM env_active ea JOIN env_configs v ON v.id = ea.config_id WHERE ea.project_id = c.project_id), 0) ORDER BY c.version DESC LIMIT 1`, projectID)); err == nil {
		st.Candidate = &c
	} else if !errors.Is(err, persistence.ErrNotFound) {
		return st, err
	}
	return st, nil
}

// Resolve devolve a versão aprovada para uma execução fixar, depois do preflight na revisão testada. Se o
// preflight falha, a execução não usa a versão: o diagnóstico vai para revisão e a anterior só é
// reutilizada quando confirmada.
func (s *Store) Resolve(ctx context.Context, projectID, sha string, pre Preflight) (Version, error) {
	st, err := s.View(ctx, projectID)
	if err != nil {
		return Version{}, err
	}
	if st.Active == nil {
		return Version{}, ErrNoActive
	}
	if pre != nil {
		if err := pre(st.Active.Config, sha); err != nil {
			_, _ = s.DB.Exec(ctx, `UPDATE env_configs SET state = 'needs_revalidation', diagnostic = $2 WHERE id::text = $1`, st.Active.ID, "preflight na revisão "+sha+": "+err.Error())
			return Version{}, fmt.Errorf("a configuração aprovada não se aplica à revisão %s: %w", sha, err)
		}
	}
	return *st.Active, nil
}
