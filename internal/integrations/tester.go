package integrations

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ViniciusBernucci/le-fabrique/internal/security"
)

// Resultados de teste.
const (
	TestOK     = "ok"
	TestFailed = "failed"
	TestNotRun = "not_run"
)

// TestResult é o resultado de um teste de conexão. Detail nunca carrega corpo
// de resposta remota nem o segredo: só frases fixas e dados não sensíveis.
type TestResult struct {
	Status string
	Detail string
}

// Tester valida credenciais contra o destino com a chamada mais barata possível.
type Tester struct {
	Client *http.Client
	// Bases substituíveis em testes.
	GitHubBase  string
	ClickUpBase string
	// AllowAnyTarget desliga a restrição de host do Jira (somente testes).
	AllowAnyTarget bool
}

// NewTester usa timeout curto e NÃO segue redirecionamentos (evita reenviar o
// cabeçalho Authorization a outro host).
func NewTester() *Tester {
	return &Tester{
		Client: &http.Client{
			Timeout: 10 * time.Second,
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		GitHubBase:  "https://api.github.com",
		ClickUpBase: "https://api.clickup.com/api/v2",
	}
}

// Test executa o teste do tipo informado. Tipos sem teste remoto seguro devolvem
// TestNotRun com explicação honesta.
func (t *Tester) Test(ctx context.Context, kind, target string, secret security.Secret) TestResult {
	switch kind {
	case "github":
		return t.github(ctx, secret)
	case "clickup":
		return t.clickup(ctx, secret)
	case "jira":
		return t.jira(ctx, target, secret)
	default:
		return TestResult{Status: TestNotRun, Detail: "Sem teste remoto para este tipo; o segredo foi decifrado com sucesso. A validação real ocorre no primeiro uso pelo adaptador."}
	}
}

func (t *Tester) do(req *http.Request) (*http.Response, TestResult) {
	resp, err := t.Client.Do(req)
	if err != nil {
		// Não repassa err.Error(): o URL/cabeçalhos podem aparecer nele.
		return nil, TestResult{Status: TestFailed, Detail: "Falha de rede ao contatar o destino."}
	}
	return resp, TestResult{}
}

func statusResult(status int, service string) TestResult {
	switch {
	case status == http.StatusUnauthorized:
		return TestResult{Status: TestFailed, Detail: service + " recusou a credencial (401): ela é inválida, expirou ou foi revogada."}
	case status == http.StatusForbidden:
		return TestResult{Status: TestFailed, Detail: service + " recusou o acesso (403): verifique os escopos da credencial."}
	case status == http.StatusTooManyRequests:
		return TestResult{Status: TestFailed, Detail: service + " limitou as requisições (429); tente de novo mais tarde."}
	default:
		return TestResult{Status: TestFailed, Detail: fmt.Sprintf("%s respondeu com status inesperado (%d).", service, status)}
	}
}

func readJSON(r io.Reader, v any) bool {
	return json.NewDecoder(io.LimitReader(r, 1<<20)).Decode(v) == nil
}

func (t *Tester) github(ctx context.Context, secret security.Secret) TestResult {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(t.GitHubBase, "/")+"/user", nil)
	if err != nil {
		return TestResult{Status: TestFailed, Detail: "Configuração do teste inválida."}
	}
	req.Header.Set("Authorization", "Bearer "+secret.Reveal())
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	resp, bad := t.do(req)
	if resp == nil {
		return bad
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return statusResult(resp.StatusCode, "GitHub")
	}
	var body struct {
		Login string `json:"login"`
	}
	_ = readJSON(resp.Body, &body)
	detail := "Credencial aceita pelo GitHub."
	if body.Login != "" {
		detail = "Credencial aceita pelo GitHub (conta " + body.Login + ")."
	}
	if scopes := resp.Header.Get("X-OAuth-Scopes"); scopes != "" {
		detail += " Escopos: " + scopes + "."
	}
	return TestResult{Status: TestOK, Detail: detail}
}

func (t *Tester) clickup(ctx context.Context, secret security.Secret) TestResult {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(t.ClickUpBase, "/")+"/user", nil)
	if err != nil {
		return TestResult{Status: TestFailed, Detail: "Configuração do teste inválida."}
	}
	req.Header.Set("Authorization", secret.Reveal())
	resp, bad := t.do(req)
	if resp == nil {
		return bad
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return statusResult(resp.StatusCode, "ClickUp")
	}
	return TestResult{Status: TestOK, Detail: "Credencial aceita pelo ClickUp."}
}

// jira espera o segredo no formato "email:token" e o destino https://<site>.atlassian.net.
func (t *Tester) jira(ctx context.Context, target string, secret security.Secret) TestResult {
	u, err := url.Parse(strings.TrimSpace(target))
	if err != nil || u.Host == "" {
		return TestResult{Status: TestFailed, Detail: "Informe o destino como https://<site>.atlassian.net."}
	}
	if !t.AllowAnyTarget && (u.Scheme != "https" || !strings.HasSuffix(strings.ToLower(u.Hostname()), ".atlassian.net")) {
		return TestResult{Status: TestFailed, Detail: "Destino Jira deve ser https://<site>.atlassian.net."}
	}
	email, token, ok := strings.Cut(secret.Reveal(), ":")
	if !ok || email == "" || token == "" {
		return TestResult{Status: TestFailed, Detail: "O segredo do Jira deve ter o formato email:token."}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.Scheme+"://"+u.Host+"/rest/api/3/myself", nil)
	if err != nil {
		return TestResult{Status: TestFailed, Detail: "Configuração do teste inválida."}
	}
	req.SetBasicAuth(email, token)
	req.Header.Set("Accept", "application/json")
	resp, bad := t.do(req)
	if resp == nil {
		return bad
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return statusResult(resp.StatusCode, "Jira")
	}
	return TestResult{Status: TestOK, Detail: "Credencial aceita pelo Jira."}
}
