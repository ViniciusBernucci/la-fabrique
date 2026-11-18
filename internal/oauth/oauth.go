// Package oauth conecta GitHub, ClickUp e Jira por OAuth, sem o usuário colar
// senha ou token (ADR-0060). GitHub usa o device flow (só client_id); ClickUp e
// Jira usam o fluxo com redirecionamento (client_id + client_secret no .env).
// O token resultante vai direto para o cofre (security.Vault) e nunca é devolvido.
package oauth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
	"github.com/ViniciusBernucci/le-fabrique/internal/security"
)

// Config vem do ambiente; provedor sem client_id fica indisponível.
type Config struct {
	GitHubClientID      string
	ClickUpClientID     string
	ClickUpClientSecret string
	JiraClientID        string
	JiraClientSecret    string
	// RedirectBase é a origem pública do painel (ex.: http://localhost:5180); o
	// callback registrado nos apps é RedirectBase + "/api/oauth/callback".
	RedirectBase string
}

// FromEnv lê a configuração das variáveis de ambiente.
func FromEnv(lookup func(string) (string, bool)) Config {
	get := func(k string) string { v, _ := lookup(k); return strings.TrimSpace(v) }
	base := get("OAUTH_REDIRECT_BASE")
	if base == "" {
		port := get("WEB_PORT")
		if port == "" {
			port = "5180"
		}
		base = "http://localhost:" + port
	}
	return Config{
		GitHubClientID: get("GITHUB_OAUTH_CLIENT_ID"), ClickUpClientID: get("CLICKUP_OAUTH_CLIENT_ID"),
		ClickUpClientSecret: get("CLICKUP_OAUTH_CLIENT_SECRET"), JiraClientID: get("JIRA_OAUTH_CLIENT_ID"),
		JiraClientSecret: get("JIRA_OAUTH_CLIENT_SECRET"), RedirectBase: strings.TrimRight(base, "/"),
	}
}

// CallbackURL é o redirect_uri registrado nos apps OAuth.
func (c Config) CallbackURL() string { return c.RedirectBase + "/api/oauth/callback" }

// Endpoints são substituíveis em testes.
type Endpoints struct {
	GitHubDevice, GitHubToken, GitHubAPI  string
	ClickUpAuth, ClickUpToken, ClickUpAPI string
	JiraAuth, JiraToken, JiraResources    string
}

// DefaultEndpoints são os endereços reais.
func DefaultEndpoints() Endpoints {
	return Endpoints{
		GitHubDevice: "https://github.com/login/device/code", GitHubToken: "https://github.com/login/oauth/access_token",
		GitHubAPI:   "https://api.github.com",
		ClickUpAuth: "https://app.clickup.com/api", ClickUpToken: "https://api.clickup.com/api/v2/oauth/token",
		ClickUpAPI: "https://api.clickup.com/api/v2",
		JiraAuth:   "https://auth.atlassian.com/authorize", JiraToken: "https://auth.atlassian.com/oauth/token",
		JiraResources: "https://api.atlassian.com/oauth/token/accessible-resources",
	}
}

// Erros do fluxo.
var (
	ErrUnavailable = errors.New("provedor OAuth não configurado")
	ErrUnknownFlow = errors.New("fluxo desconhecido ou expirado")
	ErrBadState    = errors.New("state inválido")
)

// Provedores suportados.
const (
	GitHub  = "github"
	ClickUp = "clickup"
	Jira    = "jira"
)

const (
	flowTTL       = 10 * time.Minute
	githubScopes  = "repo read:org workflow"
	jiraScopes    = "read:jira-work write:jira-work read:me offline_access"
	refreshMargin = 2 * time.Minute
	maxResponse   = 1 << 20
	statusPending = "pending"
	statusDone    = "done"
	statusFailed  = "failed"
)

// Provider descreve a disponibilidade de um provedor para a tela.
type Provider struct {
	Kind      string `json:"kind"`
	Flow      string `json:"flow"` // device | redirect
	Available bool   `json:"available"`
	Hint      string `json:"hint,omitempty"`
}

// Started é o que a tela precisa para guiar o usuário.
type Started struct {
	FlowID          string `json:"flow_id"`
	Flow            string `json:"flow"`
	UserCode        string `json:"user_code,omitempty"`
	VerificationURI string `json:"verification_uri,omitempty"`
	Interval        int    `json:"interval,omitempty"`
	AuthorizeURL    string `json:"authorize_url,omitempty"`
	ExpiresIn       int    `json:"expires_in"`
}

// Status é o andamento de um fluxo.
type Status struct {
	Status     string               `json:"status"`
	Error      string               `json:"error,omitempty"`
	Connection *security.Connection `json:"connection,omitempty"`
}

type flow struct {
	id, kind, name, state string
	deviceCode            string
	interval              time.Duration
	nextPoll              time.Time
	expires               time.Time
	status                Status
}

// Service guarda os fluxos em memória (duram minutos; reiniciar a API cancela
// os pendentes, e o usuário apenas repete).
type Service struct {
	cfg   Config
	ep    Endpoints
	hc    *http.Client
	vault *security.Vault
	now   func() time.Time

	mu    sync.Mutex
	flows map[string]*flow
}

// New cria o serviço.
func New(cfg Config, vault *security.Vault) *Service {
	return &Service{
		cfg: cfg, ep: DefaultEndpoints(), vault: vault, now: time.Now, flows: map[string]*flow{},
		hc: &http.Client{Timeout: 15 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }},
	}
}

// SetEndpoints troca os endereços (testes).
func (s *Service) SetEndpoints(e Endpoints) { s.ep = e }

// SetClock troca o relógio (testes).
func (s *Service) SetClock(now func() time.Time) { s.now = now }

// Providers lista o que está configurado.
func (s *Service) Providers() []Provider {
	return []Provider{
		{Kind: GitHub, Flow: "device", Available: s.cfg.GitHubClientID != "", Hint: "Configuração única: em github.com/settings/developers crie um OAuth App, marque \"Enable Device Flow\", copie o Client ID para GITHUB_OAUTH_CLIENT_ID no .env e reinicie a API."},
		{Kind: ClickUp, Flow: "redirect", Available: s.cfg.ClickUpClientID != "" && s.cfg.ClickUpClientSecret != "",
			Hint: "Defina CLICKUP_OAUTH_CLIENT_ID e CLICKUP_OAUTH_CLIENT_SECRET; redirect URL: " + s.cfg.CallbackURL()},
		{Kind: Jira, Flow: "redirect", Available: s.cfg.JiraClientID != "" && s.cfg.JiraClientSecret != "",
			Hint: "Defina JIRA_OAUTH_CLIENT_ID e JIRA_OAUTH_CLIENT_SECRET; callback URL: " + s.cfg.CallbackURL()},
	}
}

func randToken() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func (s *Service) sweep() {
	for id, f := range s.flows {
		if s.now().After(f.expires.Add(flowTTL)) {
			delete(s.flows, id)
		}
	}
}

// Start inicia o fluxo do provedor.
func (s *Service) Start(ctx context.Context, kind, name string) (Started, error) {
	name = strings.TrimSpace(name)
	if name == "" || len(name) > 80 {
		return Started{}, fmt.Errorf("%w: informe um nome de até 80 caracteres", persistence.ErrInvalidInput)
	}
	f := &flow{id: randToken(), kind: kind, name: name, state: randToken(), expires: s.now().Add(flowTTL), status: Status{Status: statusPending}}
	var out Started
	switch kind {
	case GitHub:
		if s.cfg.GitHubClientID == "" {
			return Started{}, ErrUnavailable
		}
		var r struct {
			DeviceCode      string `json:"device_code"`
			UserCode        string `json:"user_code"`
			VerificationURI string `json:"verification_uri"`
			ExpiresIn       int    `json:"expires_in"`
			Interval        int    `json:"interval"`
			Error           string `json:"error"`
		}
		if err := s.postForm(ctx, s.ep.GitHubDevice, url.Values{"client_id": {s.cfg.GitHubClientID}, "scope": {githubScopes}}, &r); err != nil || r.DeviceCode == "" {
			return Started{}, errors.New("o GitHub não iniciou o fluxo (verifique GITHUB_OAUTH_CLIENT_ID e o Device Flow do app)")
		}
		if r.Interval < 1 {
			r.Interval = 5
		}
		f.deviceCode, f.interval = r.DeviceCode, time.Duration(r.Interval)*time.Second
		f.nextPoll = s.now().Add(f.interval)
		if r.ExpiresIn > 0 {
			f.expires = s.now().Add(time.Duration(r.ExpiresIn) * time.Second)
		}
		out = Started{Flow: "device", UserCode: r.UserCode, VerificationURI: r.VerificationURI, Interval: r.Interval}
	case ClickUp:
		if s.cfg.ClickUpClientID == "" || s.cfg.ClickUpClientSecret == "" {
			return Started{}, ErrUnavailable
		}
		q := url.Values{"client_id": {s.cfg.ClickUpClientID}, "redirect_uri": {s.cfg.CallbackURL()}, "state": {f.state}}
		out = Started{Flow: "redirect", AuthorizeURL: s.ep.ClickUpAuth + "?" + q.Encode()}
	case Jira:
		if s.cfg.JiraClientID == "" || s.cfg.JiraClientSecret == "" {
			return Started{}, ErrUnavailable
		}
		q := url.Values{"audience": {"api.atlassian.com"}, "client_id": {s.cfg.JiraClientID}, "scope": {jiraScopes},
			"redirect_uri": {s.cfg.CallbackURL()}, "state": {f.state}, "response_type": {"code"}, "prompt": {"consent"}}
		out = Started{Flow: "redirect", AuthorizeURL: s.ep.JiraAuth + "?" + q.Encode()}
	default:
		return Started{}, fmt.Errorf("%w: OAuth não se aplica a este tipo", persistence.ErrInvalidInput)
	}
	out.FlowID = f.id
	out.ExpiresIn = int(f.expires.Sub(s.now()).Seconds())
	s.mu.Lock()
	s.sweep()
	s.flows[f.id] = f
	s.mu.Unlock()
	return out, nil
}

// Poll devolve o andamento. No device flow, consulta o GitHub respeitando o intervalo.
func (s *Service) Poll(ctx context.Context, id string) (Status, error) {
	s.mu.Lock()
	f, ok := s.flows[id]
	if !ok {
		s.mu.Unlock()
		return Status{}, ErrUnknownFlow
	}
	if f.status.Status != statusPending {
		st := f.status
		s.mu.Unlock()
		return st, nil
	}
	if s.now().After(f.expires) {
		f.status = Status{Status: statusFailed, Error: "O código expirou. Inicie a conexão de novo."}
		st := f.status
		s.mu.Unlock()
		return st, nil
	}
	if f.kind != GitHub || s.now().Before(f.nextPoll) {
		st := f.status
		s.mu.Unlock()
		return st, nil
	}
	f.nextPoll = s.now().Add(f.interval)
	device, name, interval := f.deviceCode, f.name, f.interval
	s.mu.Unlock()

	var r struct {
		AccessToken string `json:"access_token"`
		Scope       string `json:"scope"`
		Error       string `json:"error"`
		Interval    int    `json:"interval"`
	}
	err := s.postForm(ctx, s.ep.GitHubToken, url.Values{"client_id": {s.cfg.GitHubClientID}, "device_code": {device},
		"grant_type": {"urn:ietf:params:oauth:grant-type:device_code"}}, &r)
	s.mu.Lock()
	defer s.mu.Unlock()
	switch {
	case err != nil:
		return f.status, nil // falha de rede: tenta de novo no próximo ciclo
	case r.AccessToken != "":
		s.mu.Unlock()
		conn, cerr := s.vault.Create(ctx, security.NewConnection{Kind: GitHub, Name: name, Target: s.githubLogin(ctx, r.AccessToken),
			Scopes: strings.ReplaceAll(r.Scope, ",", " "), Secret: security.NewSecret(r.AccessToken)})
		s.mu.Lock()
		f.status = finish(conn, cerr)
	case r.Error == "slow_down":
		f.interval = interval + 5*time.Second
		f.nextPoll = s.now().Add(f.interval)
	case r.Error == "authorization_pending" || r.Error == "":
	case r.Error == "access_denied":
		f.status = Status{Status: statusFailed, Error: "Você recusou a autorização no GitHub."}
	default: // expired_token e demais
		f.status = Status{Status: statusFailed, Error: "O código expirou ou foi recusado. Inicie a conexão de novo."}
	}
	return f.status, nil
}

func finish(conn security.Connection, err error) Status {
	if err != nil {
		if errors.Is(err, persistence.ErrDuplicate) {
			return Status{Status: statusFailed, Error: "Já existe uma conexão com esse nome. Escolha outro nome."}
		}
		return Status{Status: statusFailed, Error: "Não foi possível guardar a conexão no cofre."}
	}
	return Status{Status: statusDone, Connection: &conn}
}

// githubLogin devolve o login da conta autorizada (vira o destino da conexão).
func (s *Service) githubLogin(ctx context.Context, token string) string {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.ep.GitHubAPI+"/user", nil)
	if err != nil {
		return ""
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := s.hc.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	var u struct {
		Login string `json:"login"`
	}
	_ = json.NewDecoder(io.LimitReader(resp.Body, maxResponse)).Decode(&u)
	return u.Login
}

// Callback conclui os fluxos com redirecionamento. A rota é pública (o cookie
// SameSite=Strict não acompanha o retorno do provedor): o `state` de 256 bits,
// de uso único e vinculado a este processo, é a prova de origem.
func (s *Service) Callback(ctx context.Context, state, code string) (Status, error) {
	if state == "" || code == "" {
		return Status{}, ErrBadState
	}
	s.mu.Lock()
	var f *flow
	for _, c := range s.flows {
		if c.state == state && c.status.Status == statusPending && c.kind != GitHub {
			f = c
			break
		}
	}
	if f == nil || s.now().After(f.expires) {
		s.mu.Unlock()
		return Status{}, ErrBadState
	}
	f.state = "" // uso único
	kind, name := f.kind, f.name
	s.mu.Unlock()

	var nc security.NewConnection
	var err error
	switch kind {
	case ClickUp:
		nc, err = s.exchangeClickUp(ctx, name, code)
	case Jira:
		nc, err = s.exchangeJira(ctx, name, code)
	}
	var st Status
	if err != nil {
		st = Status{Status: statusFailed, Error: err.Error()}
	} else {
		conn, cerr := s.vault.Create(ctx, nc)
		st = finish(conn, cerr)
	}
	s.mu.Lock()
	f.status = st
	s.mu.Unlock()
	return st, nil
}

func (s *Service) exchangeClickUp(ctx context.Context, name, code string) (security.NewConnection, error) {
	var r struct {
		AccessToken string `json:"access_token"`
	}
	body := map[string]string{"client_id": s.cfg.ClickUpClientID, "client_secret": s.cfg.ClickUpClientSecret, "code": code}
	if err := s.postJSON(ctx, s.ep.ClickUpToken, body, &r); err != nil || r.AccessToken == "" {
		return security.NewConnection{}, errors.New("o ClickUp recusou a troca do código")
	}
	return security.NewConnection{Kind: ClickUp, Name: name, Secret: security.NewSecret(r.AccessToken)}, nil
}

// jiraSecret é o conteúdo guardado no cofre para o Jira (JSON).
type jiraSecret struct {
	Version      int    `json:"v"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresAt    int64  `json:"expires_at"`
	CloudID      string `json:"cloud_id"`
}

type jiraToken struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
}

func (s *Service) exchangeJira(ctx context.Context, name, code string) (security.NewConnection, error) {
	var t jiraToken
	body := map[string]string{"grant_type": "authorization_code", "client_id": s.cfg.JiraClientID, "client_secret": s.cfg.JiraClientSecret,
		"code": code, "redirect_uri": s.cfg.CallbackURL()}
	if err := s.postJSON(ctx, s.ep.JiraToken, body, &t); err != nil || t.AccessToken == "" {
		return security.NewConnection{}, errors.New("a Atlassian recusou a troca do código")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.ep.JiraResources, nil)
	if err != nil {
		return security.NewConnection{}, errors.New("falha ao descobrir o site Jira")
	}
	req.Header.Set("Authorization", "Bearer "+t.AccessToken)
	req.Header.Set("Accept", "application/json")
	resp, err := s.hc.Do(req)
	if err != nil {
		return security.NewConnection{}, errors.New("falha ao descobrir o site Jira")
	}
	defer resp.Body.Close()
	var sites []struct {
		ID, URL string
		Scopes  []string
	}
	if resp.StatusCode != http.StatusOK || json.NewDecoder(io.LimitReader(resp.Body, maxResponse)).Decode(&sites) != nil || len(sites) == 0 {
		return security.NewConnection{}, errors.New("nenhum site Jira acessível com esta conta")
	}
	js := jiraSecret{Version: 1, AccessToken: t.AccessToken, RefreshToken: t.RefreshToken, CloudID: sites[0].ID,
		ExpiresAt: s.now().Unix() + t.ExpiresIn}
	raw, _ := json.Marshal(js)
	return security.NewConnection{Kind: Jira, Name: name, Target: sites[0].URL, Scopes: jiraScopes, Secret: security.NewSecret(string(raw))}, nil
}

// NewRefresher devolve o gancho do cofre que renova o token OAuth do Jira
// (expira em 1 h; o refresh token é rotativo e precisa ser regravado).
func NewRefresher(cfg Config) security.Refresher { return New(cfg, nil).Refresher() }

// Refresher devolve o gancho de renovação deste serviço.
func (s *Service) Refresher() security.Refresher {
	cfg := s.cfg
	var mu sync.Mutex
	return func(ctx context.Context, c security.Connection, secret security.Secret) (security.Secret, bool, error) {
		if c.Kind != Jira {
			return secret, false, nil
		}
		var js jiraSecret
		if json.Unmarshal([]byte(secret.Reveal()), &js) != nil || js.RefreshToken == "" {
			return secret, false, nil // token de API antigo (email:token)
		}
		if time.Unix(js.ExpiresAt, 0).After(s.now().Add(refreshMargin)) {
			return secret, false, nil
		}
		mu.Lock()
		defer mu.Unlock()
		var t jiraToken
		body := map[string]string{"grant_type": "refresh_token", "client_id": cfg.JiraClientID, "client_secret": cfg.JiraClientSecret, "refresh_token": js.RefreshToken}
		if err := s.postJSON(ctx, s.ep.JiraToken, body, &t); err != nil || t.AccessToken == "" {
			return secret, false, errors.New("não foi possível renovar o acesso ao Jira; reconecte a conta")
		}
		js.AccessToken, js.ExpiresAt = t.AccessToken, s.now().Unix()+t.ExpiresIn
		if t.RefreshToken != "" {
			js.RefreshToken = t.RefreshToken
		}
		raw, _ := json.Marshal(js)
		return security.NewSecret(string(raw)), true, nil
	}
}

func (s *Service) postForm(ctx context.Context, u string, v url.Values, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, strings.NewReader(v.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	return s.send(req, out)
}

func (s *Service) postJSON(ctx context.Context, u string, v any, out any) error {
	raw, _ := json.Marshal(v)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, strings.NewReader(string(raw)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	return s.send(req, out)
}

// send decodifica a resposta sem nunca expor o corpo remoto nos erros.
func (s *Service) send(req *http.Request, out any) error {
	resp, err := s.hc.Do(req)
	if err != nil {
		return errors.New("falha de rede")
	}
	defer resp.Body.Close()
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxResponse)).Decode(out); err != nil {
		return errors.New("resposta inválida")
	}
	if resp.StatusCode >= 500 {
		return errors.New("serviço indisponível")
	}
	return nil
}
