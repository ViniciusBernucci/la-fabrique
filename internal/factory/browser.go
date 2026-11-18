package factory

import (
	"archive/tar"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ViniciusBernucci/le-fabrique/internal/agents"
	"github.com/ViniciusBernucci/le-fabrique/internal/envconfig"
	"github.com/ViniciusBernucci/le-fabrique/internal/execution"
	"github.com/ViniciusBernucci/le-fabrique/internal/runner"
)

// Revisão visual do Frontend (ADR-0069): o agente revisor ganha o servidor MCP oficial do Chrome DevTools e um Chromium
// no próprio terminal isolado. A fábrica sobe o app em teste dentro do mesmo terminal e o agente o navega em localhost;
// a saída de rede continua só pelo proxy de saída. As evidências (capturas de tela) ficam em .fabrica/evidencias.
const (
	browserSkill  = "lf-revisar-frontend"
	browserPort   = "4173"
	evidenceDir   = factoryDir + "/evidencias"
	mcpServerName = "chrome-devtools"
	appStartLimit = 90 * time.Second
)

// mcpConfig aponta o Claude para o Chrome DevTools MCP do terminal (Chromium sem interface, perfil descartável).
const mcpConfig = `{"mcpServers":{"` + mcpServerName + `":{"command":"chrome-devtools-mcp","args":["--headless","--isolated","--executablePath","/usr/bin/chromium","--chromeArg=--no-sandbox","--chromeArg=--disable-dev-shm-usage","--no-usage-statistics"]}}}`

// errNoAppEnv: o projeto não tem ambiente de teste aprovado. Não derruba a revisão: ela segue sem navegador e os
// cenários visuais ficam PENDENTES (a skill manda não aprovar o que não foi visto).
var errNoAppEnv = errors.New("o projeto não tem ambiente de teste aprovado (comando para subir o app e URL do navegador): configure e aprove o ambiente visual do projeto")

func needsBrowser(eff agents.Effective) bool {
	for _, s := range eff.Skills {
		if s == browserSkill {
			return true
		}
	}
	return false
}

// appSpec diz como subir o app em teste e em que URL ele responde. No teste do fluxo serve a pasta da feature
// fictícia (funciona em qualquer projeto); nas execuções reais usa o ambiente de teste aprovado do projeto.
type appSpec struct {
	Start  string // comando que sobe o app (vazio = servidor estático da pasta Static)
	Static string
	URL    string
}

func (e *Executor) appSpecFor(ctx context.Context, info runner.Info) (appSpec, error) {
	if info.Run.Kind == runner.KindTest {
		return appSpec{Static: "fabrica-teste/frontend", URL: "http://localhost:" + browserPort + "/"}, nil
	}
	st, err := (&envconfig.Store{DB: e.DB}).View(ctx, info.Run.ProjectID)
	if err != nil {
		return appSpec{}, err
	}
	if st.Active == nil || strings.TrimSpace(st.Active.Config.Start) == "" || strings.TrimSpace(st.Active.Config.BrowserTarget) == "" {
		return appSpec{}, errNoAppEnv
	}
	return appSpec{Start: st.Active.Config.Start, URL: st.Active.Config.BrowserTarget}, nil
}

// noBrowserPrompt é o aviso quando não há ambiente de teste: revisão estática, cenários visuais PENDENTES.
const noBrowserPrompt = `## Sem navegador nesta revisão
O projeto não tem ambiente de teste aprovado (comando para subir o app e URL), então NÃO há navegador disponível. Revise o código e os critérios de forma estática e declare TODOS os cenários que dependem de ver a interface renderizada como PENDENTES, dizendo que faltou o ambiente. Não aprove nenhum critério visual sem tê-lo visto.`

func browserPrompt(spec appSpec) string {
	var b strings.Builder
	b.WriteString("## Navegador isolado (MCP)\n")
	fmt.Fprintf(&b, "O app em teste já está rodando neste terminal em %s. Você tem o servidor MCP `%s` (Chromium sem interface): use as ferramentas ", spec.URL, mcpServerName)
	b.WriteString("`mcp__" + mcpServerName + "__*` (navigate_page, take_snapshot, click, fill, take_screenshot, list_console_messages, list_network_requests…) para abrir a página, interagir pelos controles reais e observar a resposta renderizada.\n")
	b.WriteString("Comece abrindo a página com `new_page` (informe a URL); as demais ferramentas pedem o `pageId` da página aberta. ")
	b.WriteString("Navegue SOMENTE no endereço acima (qualquer outro destino não tem rota). Para cada critério verificado, salve uma captura de tela em `" + evidenceDir + "/<criterio>.png` (parâmetro filePath de take_screenshot) e cite-a no resultado. ")
	b.WriteString("Aprovação técnica exige interação observada no navegador; se o navegador ou o app falharem, diga que o cenário ficou PENDENTE e por quê (não aprove sem ter visto a página).\n")
	return b.String()
}

const staticServer = `const http=require("http"),fs=require("fs"),path=require("path");
const root=path.resolve(process.argv[2]||"."),port=+process.argv[3]||4173;
const types={".html":"text/html; charset=utf-8",".css":"text/css",".js":"text/javascript",".json":"application/json",".svg":"image/svg+xml",".png":"image/png"};
http.createServer((q,r)=>{let p=path.join(root,decodeURIComponent(q.url.split("?")[0]));
 if(!p.startsWith(root)){r.statusCode=403;return r.end()}
 if(fs.existsSync(p)&&fs.statSync(p).isDirectory())p=path.join(p,"index.html");
 fs.readFile(p,(e,b)=>{if(e){r.statusCode=404;return r.end("não encontrado")}
  r.setHeader("content-type",types[path.extname(p)]||"application/octet-stream");r.end(b)})}).listen(port,"127.0.0.1");
`

// startApp sobe o app em segundo plano dentro do terminal e espera ele responder.
func (e *Executor) startApp(ctx context.Context, info runner.Info, node string, sb *execution.Sandbox, spec appSpec) error {
	done := e.timed(info, node, "app.start", map[string]any{"url": spec.URL, "static": spec.Static, "start": spec.Start})
	cmd := spec.Start
	if cmd == "" {
		if _, err := sb.Exec(ctx, execution.ExecSpec{Cmd: []string{"sh", "-c", "cat > /tmp/serve.js"}, Stdin: strings.NewReader(staticServer)}); err != nil {
			done(nil, err)
			return err
		}
		root := spec.Static
		if r, err := sb.Exec(ctx, execution.ExecSpec{Cmd: []string{"sh", "-c", "test -d " + root}}); err != nil || r.ExitCode != 0 {
			root = "." // a trilha de frontend ainda não criou a pasta: serve a raiz e a revisão dirá o que falta
		}
		cmd = "node /tmp/serve.js " + root + " " + browserPort
	}
	if r, err := sb.Exec(ctx, execution.ExecSpec{Cmd: []string{"sh", "-c", "nohup sh -c '" + strings.ReplaceAll(cmd, "'", `'\''`) + "' >/tmp/app.log 2>&1 &"}}); err != nil || r.ExitCode != 0 {
		if err == nil {
			err = fmt.Errorf("saída %d: %s", r.ExitCode, tail(r.Stderr, 200))
		}
		done(nil, err)
		return err
	}
	probe := `fetch(process.argv[1]).then(r=>process.exit(r.status<500?0:1)).catch(()=>process.exit(1))`
	deadline := time.Now().Add(appStartLimit)
	for time.Now().Before(deadline) {
		r, err := sb.Exec(ctx, execution.ExecSpec{Cmd: []string{"node", "-e", probe, spec.URL}, Timeout: 10 * time.Second})
		if err == nil && r.ExitCode == 0 {
			done(nil, nil)
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Second):
		}
	}
	logTail := ""
	if r, err := sb.Exec(ctx, execution.ExecSpec{Cmd: []string{"sh", "-c", "tail -c 600 /tmp/app.log"}}); err == nil {
		logTail = strings.TrimSpace(r.Stdout)
	}
	err := fmt.Errorf("o app não respondeu em %s em %s: %s", spec.URL, appStartLimit, logTail)
	done(map[string]any{"app_log": logTail}, err)
	return err
}

// collectEvidence copia as capturas de tela do terminal para a pasta da execução no host.
func (e *Executor) collectEvidence(ctx context.Context, info runner.Info, node string, sb *execution.Sandbox) {
	r, err := sb.Exec(ctx, execution.ExecSpec{Cmd: []string{"sh", "-c", "test -d " + evidenceDir + " && tar -c -C " + factoryDir + " evidencias"}, Timeout: time.Minute})
	if err != nil || r.ExitCode != 0 || r.Stdout == "" {
		e.diag(info, node, "evidence.none", map[string]any{"detalhe": "o revisor não deixou capturas em " + evidenceDir})
		return
	}
	dest := filepath.Join(e.Cfg.Workspaces, ".factory", info.Run.ExecutionID, "evidencias", node)
	if err := os.MkdirAll(dest, 0o750); err != nil {
		return
	}
	tr := tar.NewReader(strings.NewReader(r.Stdout))
	var files []string
	for {
		h, err := tr.Next()
		if err != nil {
			break
		}
		clean := filepath.Clean(h.Name)
		if h.Typeflag != tar.TypeReg || strings.Contains(clean, "..") || h.Size > 10<<20 {
			continue
		}
		var buf bytes.Buffer
		if _, err := io.Copy(&buf, io.LimitReader(tr, 10<<20)); err != nil {
			continue
		}
		target := filepath.Join(dest, filepath.Base(clean))
		if os.WriteFile(target, buf.Bytes(), 0o640) == nil {
			files = append(files, filepath.Base(clean))
		}
	}
	e.diag(info, node, "evidence.saved", map[string]any{"dir": dest, "arquivos": strings.Join(files, ", ")})
	if len(files) > 0 {
		info.Emit(runner.EventNode, map[string]any{"node": node, "kind": "note", "text": fmt.Sprintf("evidências visuais salvas em %s: %s", dest, strings.Join(files, ", "))})
	}
}
