package integration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ViniciusBernucci/le-fabrique/internal/config"
	"github.com/ViniciusBernucci/le-fabrique/internal/httpapi"
	"github.com/ViniciusBernucci/le-fabrique/internal/integrations"
	"github.com/ViniciusBernucci/le-fabrique/internal/oauth"
	"github.com/ViniciusBernucci/le-fabrique/internal/security"
)

const (
	oauthGHToken   = "gho_TOKEN-OAUTH-GITHUB-nao-pode-vazar"
	oauthCUToken   = "pk_TOKEN-OAUTH-CLICKUP-nao-pode-vazar"
	oauthJiraAcc1  = "JIRA-ACCESS-1-nao-pode-vazar"
	oauthJiraAcc2  = "JIRA-ACCESS-2-nao-pode-vazar"
	oauthJiraRef2  = "JIRA-REFRESH-2-nao-pode-vazar"
	oauthSecretApp = "APP-CLIENT-SECRET-nao-pode-vazar"
)

type fakeProviders struct {
	*httptest.Server
	ghPolls   atomic.Int32
	jiraAuth  atomic.Value // último Authorization visto em /myself
	refreshed atomic.Int32
}

func newFakeProviders(t *testing.T) *fakeProviders {
	f := &fakeProviders{}
	mux := http.NewServeMux()
	mux.HandleFunc("/gh/device", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"device_code":"dev-123","user_code":"ABCD-1234","verification_uri":"https://github.com/login/device","expires_in":900,"interval":1}`)
	})
	mux.HandleFunc("/gh/token", func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		if r.Form.Get("device_code") != "dev-123" {
			fmt.Fprint(w, `{"error":"bad_verification_code"}`)
			return
		}
		if f.ghPolls.Add(1) == 1 {
			fmt.Fprint(w, `{"error":"authorization_pending"}`)
			return
		}
		fmt.Fprintf(w, `{"access_token":%q,"scope":"repo,read:org,workflow"}`, oauthGHToken)
	})
	mux.HandleFunc("/gh/user", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+oauthGHToken {
			w.WriteHeader(401)
			return
		}
		fmt.Fprint(w, `{"login":"octocat"}`)
	})
	mux.HandleFunc("/cu/token", func(w http.ResponseWriter, r *http.Request) {
		var b map[string]string
		_ = json.NewDecoder(r.Body).Decode(&b)
		if b["code"] != "cu-code" || b["client_secret"] != oauthSecretApp {
			w.WriteHeader(400)
			fmt.Fprint(w, `{"err":"x"}`)
			return
		}
		fmt.Fprintf(w, `{"access_token":%q}`, oauthCUToken)
	})
	mux.HandleFunc("/jira/token", func(w http.ResponseWriter, r *http.Request) {
		var b map[string]string
		_ = json.NewDecoder(r.Body).Decode(&b)
		switch b["grant_type"] {
		case "authorization_code":
			if b["code"] != "jira-code" {
				w.WriteHeader(400)
				return
			}
			fmt.Fprintf(w, `{"access_token":%q,"refresh_token":"JIRA-REFRESH-1","expires_in":1}`, oauthJiraAcc1)
		case "refresh_token":
			if b["refresh_token"] != "JIRA-REFRESH-1" {
				w.WriteHeader(400)
				return
			}
			f.refreshed.Add(1)
			fmt.Fprintf(w, `{"access_token":%q,"refresh_token":%q,"expires_in":3600}`, oauthJiraAcc2, oauthJiraRef2)
		}
	})
	mux.HandleFunc("/jira/resources", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `[{"id":"cloud-1","url":"https://acme.atlassian.net","scopes":[]}]`)
	})
	mux.HandleFunc("/jira/gw/cloud-1/rest/api/3/myself", func(w http.ResponseWriter, r *http.Request) {
		f.jiraAuth.Store(r.Header.Get("Authorization"))
		if r.Header.Get("Authorization") != "Bearer "+oauthJiraAcc2 {
			w.WriteHeader(401)
			return
		}
		fmt.Fprint(w, `{"accountId":"a1"}`)
	})
	f.Server = httptest.NewServer(mux)
	t.Cleanup(f.Close)
	return f
}

func TestOAuthConnections(t *testing.T) {
	fp := newFakeProviders(t)
	tester := integrations.NewTester()
	tester.JiraOAuthBase = fp.URL + "/jira/gw"
	e := newAuthEnv(t, func(c *config.API) {
		c.OAuth = oauth.Config{GitHubClientID: "gh-id", ClickUpClientID: "cu-id", ClickUpClientSecret: oauthSecretApp,
			JiraClientID: "ji-id", JiraClientSecret: oauthSecretApp, RedirectBase: "http://localhost:5180"}
	}, httpapi.WithTester(tester), httpapi.WithOAuthSetup(func(s *oauth.Service) {
		s.SetEndpoints(oauth.Endpoints{
			GitHubDevice: fp.URL + "/gh/device", GitHubToken: fp.URL + "/gh/token", GitHubAPI: fp.URL + "/gh",
			ClickUpAuth: "https://app.clickup.com/api", ClickUpToken: fp.URL + "/cu/token",
			JiraAuth: "https://auth.atlassian.com/authorize", JiraToken: fp.URL + "/jira/token", JiraResources: fp.URL + "/jira/resources",
		})
	}))
	c := e.newClient()
	c.login(nil)

	var provs struct{ Items []oauth.Provider }
	c.expect(c.get("/api/oauth/providers"), 200).JSON(&provs)
	if len(provs.Items) != 3 || !provs.Items[0].Available || !provs.Items[1].Available || !provs.Items[2].Available {
		t.Fatalf("provedores: %+v", provs.Items)
	}

	t.Run("github device flow", func(t *testing.T) {
		var st oauth.Started
		c.expect(c.post("/api/oauth/github/start", map[string]string{"name": "meu-github"}), 200).JSON(&st)
		if st.Flow != "device" || st.UserCode != "ABCD-1234" || st.VerificationURI == "" {
			t.Fatalf("início: %+v", st)
		}
		poll := func() oauth.Status {
			var s oauth.Status
			c.expect(c.get("/api/oauth/flows/"+st.FlowID), 200).JSON(&s)
			return s
		}
		if s := poll(); s.Status != "pending" { // antes do intervalo: nem consulta o GitHub
			t.Fatalf("esperava pending: %+v", s)
		}
		if fp.ghPolls.Load() != 0 {
			t.Fatal("não deveria consultar o GitHub antes do intervalo")
		}
		time.Sleep(1100 * time.Millisecond)
		if s := poll(); s.Status != "pending" || fp.ghPolls.Load() != 1 {
			t.Fatalf("authorization_pending deveria manter pending: %+v", s)
		}
		time.Sleep(1100 * time.Millisecond)
		s := poll()
		if s.Status != "done" || s.Connection == nil || s.Connection.Kind != "github" || s.Connection.Target != "octocat" {
			t.Fatalf("esperava done: %+v", s)
		}
		secret, _, err := security.NewVault(e.db.Pool, mustSealer(t, e), security.NewRedactor()).Resolve(t.Context(), s.Connection.ID)
		if err != nil || secret.Reveal() != oauthGHToken {
			t.Fatalf("o token deve estar no cofre: %v", err)
		}
	})

	redirectFlow := func(t *testing.T, kind, name, code string) (oauth.Started, string) {
		var st oauth.Started
		c.expect(c.post("/api/oauth/"+kind+"/start", map[string]string{"name": name}), 200).JSON(&st)
		if st.Flow != "redirect" || !strings.Contains(st.AuthorizeURL, "state=") ||
			!strings.Contains(st.AuthorizeURL, url.QueryEscape("http://localhost:5180/api/oauth/callback")) {
			t.Fatalf("authorize_url: %+v", st)
		}
		u, _ := url.Parse(st.AuthorizeURL)
		state := u.Query().Get("state")
		// Retorno do provedor: navegação sem cookie de sessão.
		noRedirect := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
		resp, err := noRedirect.Get(e.srv.URL + "/api/oauth/callback?code=" + code + "&state=" + state)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		return st, resp.Header.Get("Location")
	}

	t.Run("clickup redirect flow", func(t *testing.T) {
		st, loc := redirectFlow(t, "clickup", "meu-clickup", "cu-code")
		if loc != "/connections?oauth=ok" {
			t.Fatalf("redirect: %s", loc)
		}
		var s oauth.Status
		c.expect(c.get("/api/oauth/flows/"+st.FlowID), 200).JSON(&s)
		if s.Status != "done" || s.Connection.Kind != "clickup" {
			t.Fatalf("%+v", s)
		}
	})

	t.Run("state inválido ou reutilizado é recusado", func(t *testing.T) {
		noRedirect := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
		resp, _ := noRedirect.Get(e.srv.URL + "/api/oauth/callback?code=x&state=inventado")
		resp.Body.Close()
		if loc := resp.Header.Get("Location"); !strings.HasPrefix(loc, "/connections?oauth=failed") {
			t.Fatalf("redirect: %s", loc)
		}
	})

	t.Run("jira redirect flow, uso pelo gateway e renovação do token", func(t *testing.T) {
		_, loc := redirectFlow(t, "jira", "meu-jira", "jira-code")
		if loc != "/connections?oauth=ok" {
			t.Fatalf("redirect: %s", loc)
		}
		var list struct{ Items []security.Connection }
		c.expect(c.get("/api/connections"), 200).JSON(&list)
		var jiraID string
		for _, it := range list.Items {
			if it.Kind == "jira" {
				jiraID = it.ID
				if it.Target != "https://acme.atlassian.net" {
					t.Fatalf("destino: %s", it.Target)
				}
			}
		}
		// O access token nasce com 1 s de vida: o teste força a renovação (rotativa) e usa o novo.
		var tested security.Connection
		c.expect(c.post("/api/connections/"+jiraID+"/test", nil), 200).JSON(&tested)
		if tested.LastTestStatus != "ok" || fp.refreshed.Load() != 1 {
			t.Fatalf("teste: %+v, renovações=%d", tested, fp.refreshed.Load())
		}
		if got, _ := fp.jiraAuth.Load().(string); got != "Bearer "+oauthJiraAcc2 {
			t.Fatalf("o gateway deveria receber o token renovado, veio %q", got)
		}
	})

	t.Run("segredos não vazam", func(t *testing.T) {
		var all strings.Builder
		all.Write(c.get("/api/connections").Body)
		all.WriteString(e.logs.String())
		for _, s := range []string{oauthGHToken, oauthCUToken, oauthJiraAcc1, oauthJiraAcc2, oauthJiraRef2, "JIRA-REFRESH-1", oauthSecretApp} {
			if strings.Contains(all.String(), s) {
				t.Fatalf("vazou %q", s)
			}
		}
	})

	t.Run("sem configuração o provedor fica indisponível", func(t *testing.T) {
		e2 := newAuthEnv(t, nil)
		c2 := e2.newClient()
		c2.login(nil)
		c2.expect(c2.post("/api/oauth/github/start", map[string]string{"name": "x"}), 409)
		c2.expect(c2.post("/api/oauth/mcp/start", map[string]string{"name": "x"}), 400)
	})
}

func mustSealer(t *testing.T, e *authEnv) *security.Sealer {
	t.Helper()
	sealer, err := security.NewSealer(e.cfg.VaultKey, e.cfg.VaultKeyID)
	if err != nil {
		t.Fatal(err)
	}
	return sealer
}
