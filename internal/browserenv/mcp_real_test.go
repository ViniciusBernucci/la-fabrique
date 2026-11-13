package browserenv

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"
)

const mcpImage = "mcr.microsoft.com/playwright/mcp:latest"

// Usa Docker de verdade com a imagem oficial do Playwright MCP (Chromium). Pulado sem runtime ou sem a imagem.
func TestRealChromiumMCPReviewsTheAuthorizedAppOnlyAndCleansUp(t *testing.T) {
	rt := DockerCLI{}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	if _, err := rt.Run(ctx, "info", "--format", "{{.ServerVersion}}"); err != nil {
		t.Skip("Docker indisponível")
	}
	for _, img := range []string{mcpImage, "node:24-alpine"} {
		if out, _ := rt.Run(ctx, "images", "-q", img); strings.TrimSpace(out) == "" {
			t.Skip("imagem ausente: " + img)
		}
	}
	const exec = "real-mcp"
	t.Cleanup(func() { _ = CleanupExecution(context.Background(), rt, exec) })
	_ = CleanupExecution(ctx, rt, exec)

	spec := Spec{ProjectID: "p-real", ExecutionID: exec, Image: mcpImage, Entrypoint: "sleep", Command: []string{"3600"},
		Allowed: []string{"http://webapp:3000"}, MemoryMB: 1536, CPUs: 2}
	var mcp *MCPDriver
	rv, ev := RunReview(ctx, rt, spec, "rev-real", 20*time.Second,
		func(env *Env) Driver {
			// A aplicação de teste entra na rede interna da execução com o nome autorizado.
			app := "lf-brw-app-" + exec
			if _, err := rt.Run(ctx, "run", "-d", "--name", app, "--label", "lf.managed=true", "--label", "lf.kind=app", "--label", "lf.exec="+exec, "--label", "lf.project=p-real",
				"node:24-alpine", "node", "-e",
				`require("http").createServer((q,r)=>{r.setHeader("content-type","text/html; charset=utf-8");r.end("<html><meta charset=utf-8><title>App de teste</title><body><h1>Olá fábrica</h1><button>Salvar</button></body></html>")}).listen(3000)`); err != nil {
				t.Fatal(err)
			}
			if err := env.Attach(ctx, app, "webapp"); err != nil {
				t.Fatal(err)
			}
			time.Sleep(time.Second)
			d, err := StartMCP(ctx, "docker", env)
			if err != nil {
				t.Fatalf("MCP: %v", err)
			}
			mcp = d
			return d
		},
		func(tl *Tools) (bool, error) {
			if err := tl.Navigate(ctx, "http://webapp:3000/"); err != nil {
				return false, err
			}
			out, err := tl.Interact(ctx, "snapshot", nil)
			if err != nil || !strings.Contains(out, "Olá fábrica") {
				t.Errorf("o snapshot deve conter a página: %v %q", err, out)
			}
			if err := tl.Screenshot(ctx, "home"); err != nil {
				return false, err
			}
			_ = tl.Logs(ctx)
			// A ferramenta bloqueia destinos fora da aplicação autorizada.
			if err := tl.Navigate(ctx, "https://example.com"); err == nil {
				t.Error("destino externo deve ser bloqueado pelas ferramentas")
			}
			// E, mesmo ignorando as ferramentas, o navegador não sai da rede interna.
			if err := mcp.Navigate(ctx, "http://1.1.1.1/"); err == nil {
				t.Error("o navegador não pode alcançar a internet")
			}
			return true, nil
		})
	if mcp != nil {
		defer mcp.Close()
	}
	if !rv.Approved || rv.Pending || !rv.Independent {
		t.Fatalf("%+v %+v", rv, ev)
	}
	var png bool
	for _, it := range ev {
		if it.Kind == "screenshot" && bytes.HasPrefix(it.Data, []byte("\x89PNG")) {
			png = true
		}
	}
	if !png {
		t.Fatalf("a evidência visual deve ser um PNG real: %+v", ev)
	}
	// Tudo da execução foi recolhido (container do navegador, app e rede).
	for _, args := range [][]string{{"ps", "-aq", "--filter", "label=lf.exec=" + exec}, {"network", "ls", "-q", "--filter", "label=lf.exec=" + exec}} {
		if out, _ := rt.Run(ctx, args...); strings.TrimSpace(out) != "" {
			t.Fatalf("sobrou recurso da execução: %v %q", args, out)
		}
	}
}
