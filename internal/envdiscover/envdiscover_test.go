package envdiscover

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/ViniciusBernucci/le-fabrique/internal/projectres"
)

const compose = `services:
  web:
    build: ./apps/web
    ports:
      - "5180:5173"
    depends_on:
      - api
    healthcheck:
      test: ["CMD", "wget", "-q", "http://localhost:5173"]
  api:
    build: .
    command: ./api
    ports:
      - "8480:8080"
    depends_on:
      db:
        condition: service_healthy
    healthcheck:
      test: ["CMD", "wget", "-q", "http://localhost:8080/health"]
  db:
    image: postgres:17-alpine
    healthcheck:
      test: ["CMD", "pg_isready"]
`

func TestDiscoverProposesServicesWithSourcesAndNoSecretValues(t *testing.T) {
	p := Discover(projectres.MapFiles{
		"docker-compose.yml": compose,
		"package.json":       `{"scripts":{"build":"vite build"}}`,
		"Makefile":           "migrate:\n\tgo run ./cmd/migrate\nseed:\n\tgo run ./cmd/seed\ndown:\n\tdocker compose down -v\ntest:\n\tgo test ./...\n",
		".env.example":       "DB_PASSWORD=supersecretvalue\nAPI_TOKEN=changeme\nPORT=8080\n",
	})
	raw, _ := json.Marshal(p)
	if strings.Contains(string(raw), "supersecretvalue") {
		t.Fatalf("o valor de variável não pode entrar na proposta: %s", raw)
	}
	kinds := map[string]string{}
	for _, s := range p.Services {
		kinds[s.Name] = s.Kind
		if s.Source == "" {
			t.Fatalf("serviço sem origem: %+v", s)
		}
	}
	if kinds["web"] != "frontend" || kinds["api"] != "api" || kinds["db"] != "database" {
		t.Fatalf("%v", kinds)
	}
	if p.BrowserTarget.Value != "http://web:5173" || !strings.Contains(p.BrowserTarget.Source, "docker-compose.yml") {
		t.Fatalf("%+v", p.BrowserTarget)
	}
	if p.DataSetup.Value != "make migrate && make seed" || p.DataCleanup.Value != "make down" || p.Test.Value == "" {
		t.Fatalf("%+v", p)
	}
	for _, s := range p.Services {
		if s.Readiness.Value == "" {
			t.Fatalf("a prontidão declarada deve ser lida: %+v", s)
		}
	}
	// Credenciais só por referência; valor com cara de real vira lacuna.
	var refs []string
	for _, c := range p.Credentials {
		refs = append(refs, c.Ref)
	}
	if strings.Join(refs, ",") != "env:DB_PASSWORD,env:API_TOKEN,env:PORT" {
		t.Fatalf("%v", refs)
	}
	if !strings.Contains(strings.Join(p.Gaps, "|"), "DB_PASSWORD") || len(p.Ambiguities) != 0 {
		t.Fatalf("gaps=%v amb=%v", p.Gaps, p.Ambiguities)
	}
}

func TestDiscoverShowsGapsAndAmbiguitiesForAmbiguousRepositories(t *testing.T) {
	p := Discover(projectres.MapFiles{
		"docker-compose.yml": "services:\n  web:\n    image: nginx\n  admin-ui:\n    image: nginx\n    ports:\n      - 3000\n",
		"compose.yaml":       "services:\n  other:\n    image: busybox\n",
		"package.json":       `{"scripts":{"dev":"vite"}}`,
		"Makefile":           "up:\n\tdocker compose up\n",
	})
	all := strings.Join(p.Gaps, "|") + "|" + strings.Join(p.Ambiguities, "|")
	for _, want := range []string{"mais de um arquivo de composição", "mais de um serviço de frontend", "dois jeitos de subir a aplicação", "comando de testes não encontrado", "preparo de dados", "não declara prontidão"} {
		if !strings.Contains(all, want) {
			t.Fatalf("faltou %q em:\n%s", want, all)
		}
	}
	if p.BrowserTarget.Value != "" {
		t.Fatalf("com frontends ambíguos o alvo não é escolhido sozinho: %+v", p.BrowserTarget)
	}
}
