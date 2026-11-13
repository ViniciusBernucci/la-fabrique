package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/ViniciusBernucci/le-fabrique/internal/delivery"
	"github.com/ViniciusBernucci/le-fabrique/internal/effects"
	"github.com/ViniciusBernucci/le-fabrique/internal/gitops"
	"github.com/ViniciusBernucci/le-fabrique/internal/memory"
	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
	"github.com/ViniciusBernucci/le-fabrique/internal/projects"
	"github.com/ViniciusBernucci/le-fabrique/internal/queue"
	"github.com/ViniciusBernucci/le-fabrique/internal/tasksync"
)

// UseGitHubBase aponta a API do GitHub para outro endereço http/loopback (somente testes).
func (s *Server) UseGitHubBase(base string) { s.githubBase, s.githubPrivate = base, true }

var errRepoIncomplete = errors.New("o repositório do projeto não tem conexão do GitHub, dono e nome")

// repoClient monta o acesso ao GitHub do projeto com a conexão do cofre (o segredo não sai daqui).
func (s *Server) repoClient(r *http.Request, p projects.Project) (gitops.MergeVerifier, gitops.PRClient, error) {
	rp := p.Repository
	if rp.ConnectionID == nil || rp.Owner == "" || rp.Name == "" {
		return gitops.MergeVerifier{}, gitops.PRClient{}, errRepoIncomplete
	}
	secret, conn, err := s.vault.Resolve(r.Context(), *rp.ConnectionID)
	if err != nil {
		return gitops.MergeVerifier{}, gitops.PRClient{}, err
	}
	if conn.Kind != "github" {
		return gitops.MergeVerifier{}, gitops.PRClient{}, errRepoIncomplete
	}
	api, err := gitops.NewGitHubAPI(secret, s.githubBase, s.githubPrivate)
	if err != nil {
		return gitops.MergeVerifier{}, gitops.PRClient{}, err
	}
	return gitops.MergeVerifier{API: api, Owner: rp.Owner, Name: rp.Name}, gitops.PRClient{API: api, Owner: rp.Owner, Name: rp.Name}, nil
}

// ---- configuração de entrega ------------------------------------------------

type deliverySettings struct {
	ReviewAuthors []string          `json:"review_authors"`
	StatusMap     map[string]string `json:"status_map"`
	Version       int               `json:"version"`
}

var loginRe = regexp.MustCompile(`^[A-Za-z0-9](?:[A-Za-z0-9-]{0,38})$`)

func (s *Server) loadSettings(ctx context.Context, project string) (deliverySettings, error) {
	out := deliverySettings{ReviewAuthors: []string{}, StatusMap: map[string]string{}}
	var authors, smap []byte
	err := s.db.QueryRow(ctx, `SELECT review_authors, status_map, version FROM delivery_settings WHERE project_id::text = $1`, project).Scan(&authors, &smap, &out.Version)
	if errors.Is(persistence.MapError(err), persistence.ErrNotFound) {
		return out, nil
	}
	if err != nil {
		return out, persistence.MapError(err)
	}
	_ = json.Unmarshal(authors, &out.ReviewAuthors)
	_ = json.Unmarshal(smap, &out.StatusMap)
	return out, nil
}

func (s *Server) getDeliverySettings(w http.ResponseWriter, r *http.Request) {
	id, ok := s.projectID(w, r)
	if !ok {
		return
	}
	out, err := s.loadSettings(r.Context(), id)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) putDeliverySettings(w http.ResponseWriter, r *http.Request) {
	id, ok := s.projectID(w, r)
	if !ok {
		return
	}
	var in struct {
		ReviewAuthors []string          `json:"review_authors"`
		StatusMap     map[string]string `json:"status_map"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	if _, err := projects.Get(r.Context(), s.db, id); err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	seen := map[string]bool{}
	authors := []string{}
	for _, a := range in.ReviewAuthors {
		a = strings.TrimSpace(a)
		if !loginRe.MatchString(a) {
			writeError(w, r, http.StatusBadRequest, "invalid_request", "Login do GitHub inválido: "+a+".", "Use só letras, números e hífen (até 39 caracteres).")
			return
		}
		if k := strings.ToLower(a); !seen[k] {
			seen[k] = true
			authors = append(authors, a)
		}
	}
	if len(authors) > 20 {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "No máximo 20 autores.", "Remova alguns logins.")
		return
	}
	smap := map[string]string{}
	for k, v := range in.StatusMap {
		v = strings.TrimSpace(v)
		if (k != tasksync.StateAwaitingMerge && k != tasksync.StateDone) || len(v) > 100 {
			writeError(w, r, http.StatusBadRequest, "invalid_request", "Mapa de estados inválido.", "Informe só awaiting_merge e done, com nomes de até 100 caracteres.")
			return
		}
		if v != "" {
			smap[k] = v
		}
	}
	a, _ := json.Marshal(authors)
	m, _ := json.Marshal(smap)
	if _, err := s.db.Exec(r.Context(), `INSERT INTO delivery_settings (project_id, review_authors, status_map) VALUES ($1, $2::jsonb, $3::jsonb)
		ON CONFLICT (project_id) DO UPDATE SET review_authors = $2::jsonb, status_map = $3::jsonb, version = delivery_settings.version + 1, updated_at = now()`, id, string(a), string(m)); err != nil {
		s.writeDomainError(w, r, persistence.MapError(err))
		return
	}
	out, err := s.loadSettings(r.Context(), id)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

// ---- entregas -----------------------------------------------------------------

type deliveryView struct {
	ID              string         `json:"id"`
	FeatureID       string         `json:"feature_id"`
	FeatureKey      string         `json:"feature_key"`
	FeatureTitle    string         `json:"feature_title"`
	Attempt         int            `json:"attempt"`
	Branch          string         `json:"branch"`
	Revision        string         `json:"revision"`
	State           string         `json:"state"`
	Reasons         []string       `json:"reasons"`
	PRURL           string         `json:"pr_url"`
	MergeSHA        string         `json:"merge_sha"`
	Projection      string         `json:"projection"`
	ProjectionError string         `json:"projection_error"`
	ObservedAt      *time.Time     `json:"observed_at"`
	CreatedAt       time.Time      `json:"created_at"`
	Comments        map[string]int `json:"comments"`
}

const deliveryCols = `d.id::text, d.feature_id::text, f.external_id, f.title, d.attempt, d.branch, d.revision, d.state, d.reasons, d.pr_url, d.merge_sha, d.projection, d.projection_error, d.observed_at, d.created_at`

func scanDelivery(row interface{ Scan(...any) error }) (deliveryView, error) {
	var v deliveryView
	var reasons []byte
	if err := row.Scan(&v.ID, &v.FeatureID, &v.FeatureKey, &v.FeatureTitle, &v.Attempt, &v.Branch, &v.Revision, &v.State, &reasons, &v.PRURL, &v.MergeSHA, &v.Projection, &v.ProjectionError, &v.ObservedAt, &v.CreatedAt); err != nil {
		return v, persistence.MapError(err)
	}
	v.Reasons = []string{}
	_ = json.Unmarshal(reasons, &v.Reasons)
	v.Comments = map[string]int{}
	return v, nil
}

func (s *Server) commentCounts(ctx context.Context, deliveryID string) map[string]int {
	out := map[string]int{}
	rows, err := s.db.Query(ctx, `SELECT state, count(*) FROM pr_comments WHERE delivery_id::text = $1 GROUP BY state`, deliveryID)
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var st string
		var n int
		if rows.Scan(&st, &n) == nil {
			out[st] = n
		}
	}
	return out
}

func (s *Server) listDeliveries(w http.ResponseWriter, r *http.Request) {
	id, ok := s.projectID(w, r)
	if !ok {
		return
	}
	rows, err := s.db.Query(r.Context(), `SELECT `+deliveryCols+` FROM deliveries d JOIN features f ON f.id = d.feature_id AND f.project_id = d.project_id
		WHERE d.project_id::text = $1 ORDER BY d.created_at DESC, d.attempt DESC LIMIT 100`, id)
	if err != nil {
		s.writeDomainError(w, r, persistence.MapError(err))
		return
	}
	items := []deliveryView{}
	for rows.Next() {
		v, err := scanDelivery(rows)
		if err != nil {
			rows.Close()
			s.writeDomainError(w, r, err)
			return
		}
		items = append(items, v)
	}
	rows.Close()
	for i := range items {
		items[i].Comments = s.commentCounts(r.Context(), items[i].ID)
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

// deliveryOf valida o projeto e a entrega (de outro projeto é "não encontrado").
func (s *Server) deliveryOf(w http.ResponseWriter, r *http.Request) (project string, d deliveryView, ok bool) {
	project, ok = s.projectID(w, r)
	if !ok {
		return "", d, false
	}
	did := r.PathValue("deliveryId")
	if !uuidPattern.MatchString(did) {
		writeError(w, r, http.StatusNotFound, "not_found", "Recurso não encontrado.", "Verifique o identificador.")
		return "", d, false
	}
	d, err := scanDelivery(s.db.QueryRow(r.Context(), `SELECT `+deliveryCols+` FROM deliveries d JOIN features f ON f.id = d.feature_id AND f.project_id = d.project_id
		WHERE d.project_id::text = $1 AND d.id::text = $2`, project, did))
	if err != nil {
		s.writeDomainError(w, r, err)
		return "", d, false
	}
	d.Comments = s.commentCounts(r.Context(), d.ID)
	return project, d, true
}

func (s *Server) externalFor(r *http.Request, p projects.Project, settings deliverySettings) (*delivery.External, gitops.MergeVerifier, error) {
	ver, cli, err := s.repoClient(r, p)
	if err != nil {
		return nil, ver, err
	}
	ext := &delivery.External{DB: s.db, Comments: cli, Reopener: cli, Merges: ver, Queue: s.queueStore(), Redactor: s.redactor, Authorized: settings.ReviewAuthors}
	return ext, ver, nil
}

func (s *Server) getDelivery(w http.ResponseWriter, r *http.Request) {
	project, d, ok := s.deliveryOf(w, r)
	if !ok {
		return
	}
	ext := &delivery.External{DB: s.db}
	comments, err := ext.List(r.Context(), d.ID)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	history, err := ext.History(r.Context(), d.ID)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	var offer *delivery.Offer
	if d.State == "closed" || d.State == "cancelled" {
		o, err := ext.Offer(r.Context(), d.ID)
		if err != nil {
			s.writeDomainError(w, r, err)
			return
		}
		offer = &o
	}
	rows, err := s.db.Query(r.Context(), `SELECT outcome, detail, created_at FROM delivery_observations WHERE delivery_id::text = $1 ORDER BY id DESC LIMIT 10`, d.ID)
	if err != nil {
		s.writeDomainError(w, r, persistence.MapError(err))
		return
	}
	type obsRow struct {
		Outcome   string    `json:"outcome"`
		Detail    string    `json:"detail"`
		CreatedAt time.Time `json:"created_at"`
	}
	obs := []obsRow{}
	for rows.Next() {
		var o obsRow
		if rows.Scan(&o.Outcome, &o.Detail, &o.CreatedAt) == nil {
			obs = append(obs, o)
		}
	}
	rows.Close()
	writeJSON(w, http.StatusOK, map[string]any{"project_id": project, "delivery": d, "comments": comments, "offer": offer, "history": history, "observations": obs})
}

// observeDelivery é a consulta SOB DEMANDA da PR (D89): nada a dispara sozinho. Com o merge
// provado (SHA), conclui o item na origem (se o mapa de estados existir), devolve à fila quem
// esperava e promove a memória pendente da branch.
func (s *Server) observeDelivery(w http.ResponseWriter, r *http.Request) {
	project, d, ok := s.deliveryOf(w, r)
	if !ok {
		return
	}
	p, err := projects.Get(r.Context(), s.db, project)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	settings, err := s.loadSettings(r.Context(), project)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	ver, _, err := s.repoClient(r, p)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	q := &queue.Queue{DB: s.db, Gates: map[string]queue.Gate{queue.WaitMerge: queue.MergeGate{Merges: ver}}}
	comp := &delivery.Completer{DB: s.db, Merges: ver, Queue: q}
	originNote := "mapa de estados da origem não configurado: o item não foi movido"
	if settings.StatusMap[tasksync.StateDone] != "" {
		if tr, err := s.sourceTracker(r, p); err == nil {
			comp.Origin = &tasksync.Publisher{Tracker: tr, Ledger: &effects.Ledger{DB: s.db, Redactor: s.redactor}, StatusMap: settings.StatusMap, Merges: ver}
			originNote = ""
		} else {
			originNote = "origem indisponível (" + describeErr(err) + "): o item não foi movido"
		}
	}
	keyBase := project + ":" + d.FeatureID + ":" + itoa(d.Attempt)
	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()
	obs, err := comp.Observe(ctx, d.ID, keyBase, d.FeatureKey)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	resp := map[string]any{"observation": obs, "origin_note": originNote}
	if obs.Outcome == "merged" {
		mem := &memory.Store{DB: s.db, Redactor: s.redactor}
		prom, perr := mem.Promote(r.Context(), project, d.ID)
		switch {
		case perr == nil:
			resp["memory"] = map[string]int{"promoted": len(prom.Promoted), "rejected": len(prom.Rejected), "duplicates": len(prom.Duplicates)}
		default:
			resp["memory_error"] = "a promoção da memória falhou: repita a consulta"
			s.log.WarnContext(r.Context(), "promoção de memória falhou", "error", perr.Error())
		}
	}
	writeJSON(w, http.StatusOK, resp)
}

func describeErr(err error) string {
	if errors.Is(err, projects.ErrSourceIncomplete) {
		return "origem incompleta"
	}
	return "falha ao abrir a conexão"
}

func itoa(n int) string {
	b, _ := json.Marshal(n)
	return string(b)
}

func (s *Server) discoverComments(w http.ResponseWriter, r *http.Request) {
	project, d, ok := s.deliveryOf(w, r)
	if !ok {
		return
	}
	var in struct {
		TaskKey  string `json:"task_key"`
		Priority *int   `json:"priority"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	prio := 1
	if in.Priority != nil {
		prio = *in.Priority
	}
	if strings.TrimSpace(in.TaskKey) == "" {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Informe a tarefa que recebe a correção.", "Escolha a tarefa do plano ligada a esta entrega.")
		return
	}
	p, err := projects.Get(r.Context(), s.db, project)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	settings, err := s.loadSettings(r.Context(), project)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	ext, _, err := s.externalFor(r, p, settings)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()
	res, err := ext.Discover(ctx, d.ID, strings.TrimSpace(in.TaskKey), prio)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"discovery": res, "authorized": len(settings.ReviewAuthors)})
}

func (s *Server) decideDelivery(w http.ResponseWriter, r *http.Request) {
	project, d, ok := s.deliveryOf(w, r)
	if !ok {
		return
	}
	var in struct {
		Action string `json:"action"`
		Note   string `json:"note"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	p, err := projects.Get(r.Context(), s.db, project)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	settings, err := s.loadSettings(r.Context(), project)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	var ext *delivery.External
	if in.Action == delivery.ActionReopen {
		ext, _, err = s.externalFor(r, p, settings) // reabrir fala com o GitHub
	} else {
		ext = &delivery.External{DB: s.db, Queue: s.queueStore()}
	}
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	dec, err := ext.Decide(r.Context(), d.ID, in.Action, s.adminID(r), in.Note)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	if !dec.Replayed {
		s.auth.Audit(r.Context(), s.adminID(r), "delivery_"+in.Action, s.meta(r), map[string]any{"delivery_id": d.ID, "state": dec.State})
	}
	writeJSON(w, http.StatusOK, dec)
}

// ---- memória --------------------------------------------------------------------

func (s *Server) memoryStore() *memory.Store { return &memory.Store{DB: s.db, Redactor: s.redactor} }

func (s *Server) listMemory(w http.ResponseWriter, r *http.Request) {
	id, ok := s.projectID(w, r)
	if !ok {
		return
	}
	items, err := s.memoryStore().List(r.Context(), id, r.URL.Query().Get("state"), r.URL.Query().Get("q"), 100)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) invalidateMemory(w http.ResponseWriter, r *http.Request) {
	id, ok := s.projectID(w, r)
	if !ok {
		return
	}
	rid := r.PathValue("recordId")
	if !uuidPattern.MatchString(rid) {
		writeError(w, r, http.StatusNotFound, "not_found", "Recurso não encontrado.", "Verifique o identificador.")
		return
	}
	var in struct {
		Reason string `json:"reason"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	rec, err := s.memoryStore().Invalidate(r.Context(), id, rid, in.Reason)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	s.auth.Audit(r.Context(), s.adminID(r), "memory_invalidated", s.meta(r), map[string]any{"record_id": rid})
	writeJSON(w, http.StatusOK, rec)
}

func (s *Server) extractMemory(w http.ResponseWriter, r *http.Request) {
	project, d, ok := s.deliveryOf(w, r)
	if !ok {
		return
	}
	recs, err := s.memoryStore().ExtractFromDelivery(r.Context(), project, d.ID)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": recs})
}

// ---- controles operacionais --------------------------------------------------------

// refusingSuspender: a API não controla o sandbox. Parar o que está rodando exige o executor;
// sem ele, o item segue em execução e é reportado (nunca é dado como parado sem checkpoint).
type refusingSuspender struct{}

func (refusingSuspender) Suspend(context.Context, queue.Item) (string, bool, error) {
	return "", false, errors.New("o executor de sandbox não está ligado à API: não há como parar com checkpoint seguro")
}

func (s *Server) setControl(scope queue.Scope) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !scope.Global {
			id, ok := s.projectID(w, r)
			if !ok {
				return
			}
			if _, err := projects.Get(r.Context(), s.db, id); err != nil {
				s.writeDomainError(w, r, err)
				return
			}
			scope.ProjectID = id
		}
		var in struct {
			Mode   string `json:"mode"`
			Reason string `json:"reason"`
		}
		if !decodeJSON(w, r, &in) {
			return
		}
		q := s.queueStore()
		var out map[string]any
		switch in.Mode {
		case queue.ModeActive:
			ctl, resumed, err := q.ResumeScope(r.Context(), scope, s.adminID(r), in.Reason)
			if err != nil {
				s.writeDomainError(w, r, err)
				return
			}
			out = map[string]any{"control": ctl, "resumed": resumed}
		case queue.ModeDraining:
			ctl, err := q.SetMode(r.Context(), scope, queue.ModeDraining, s.adminID(r), in.Reason)
			if err != nil {
				s.writeDomainError(w, r, err)
				return
			}
			out = map[string]any{"control": ctl}
		case queue.ModeSuspended:
			res, err := q.SuspendNow(r.Context(), scope, s.adminID(r), in.Reason, refusingSuspender{})
			if err != nil {
				s.writeDomainError(w, r, err)
				return
			}
			out = map[string]any{"control": res.Control, "suspended": res.Suspended, "failed": res.Failed}
		default:
			writeError(w, r, http.StatusBadRequest, "invalid_request", "Modo desconhecido.", "Use active, draining ou suspended.")
			return
		}
		s.auth.Audit(r.Context(), s.adminID(r), "control_"+in.Mode, s.meta(r), map[string]any{"scope": scope.ProjectID, "global": scope.Global})
		writeJSON(w, http.StatusOK, out)
	}
}

// ---- relatório da feature --------------------------------------------------------------

func (s *Server) featureReport(w http.ResponseWriter, r *http.Request) {
	project, ok := s.projectID(w, r)
	if !ok {
		return
	}
	fid := r.PathValue("featureId")
	if !uuidPattern.MatchString(fid) {
		writeError(w, r, http.StatusNotFound, "not_found", "Recurso não encontrado.", "Verifique o identificador.")
		return
	}
	var revision, branch, deliveryID string
	err := s.db.QueryRow(r.Context(), `SELECT revision, branch, id::text FROM deliveries WHERE project_id::text = $1 AND feature_id::text = $2 ORDER BY attempt DESC LIMIT 1`, project, fid).Scan(&revision, &branch, &deliveryID)
	if err != nil && !errors.Is(persistence.MapError(err), persistence.ErrNotFound) {
		s.writeDomainError(w, r, persistence.MapError(err))
		return
	}
	rep, err := delivery.BuildReport(r.Context(), s.db, project, fid, revision, branch)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"feature_key": rep.ExternalKey, "title": rep.Title, "revision": revision, "delivery_id": deliveryID,
		"doc_path": delivery.FeatureDocPath(rep.ExternalKey), "markdown": rep.Render()})
}
