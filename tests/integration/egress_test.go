package integration

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ViniciusBernucci/le-fabrique/internal/execution"
)

// ADR-0069 — com Docker real: o terminal numa rede interna só alcança, pelo proxy, os destinos permitidos; fora da lista é
// recusado e, sem o proxy, não há rota alguma. O proxy sobrevive à destruição do terminal.
func TestEgressProxyIsTheOnlyWayOutOfTheTerminal(t *testing.T) {
	dockerReady(t)
	ctx := context.Background()
	arch, err := exec.Command("docker", "version", "-f", "{{.Server.Arch}}").Output()
	if err != nil {
		t.Skip("docker version indisponível")
	}
	// imagem do proxy: binário para a arquitetura do Docker + a imagem mínima (o mesmo que o make egress-image faz)
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "egress-build"), 0o755); err != nil {
		t.Fatal(err)
	}
	build := exec.Command("go", "build", "-trimpath", "-o", filepath.Join(dir, "egress-build", "egressproxy"), "../../cmd/egressproxy")
	build.Env = append(os.Environ(), "CGO_ENABLED=0", "GOOS=linux", "GOARCH="+strings.TrimSpace(string(arch)))
	if out, err := build.CombinedOutput(); err != nil {
		t.Skipf("não consegui compilar o proxy: %s", out)
	}
	df, _ := os.ReadFile("../../infra/images/Dockerfile.egress")
	if err := os.WriteFile(filepath.Join(dir, "Dockerfile"), df, 0o644); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("docker", "build", "-q", "-t", "lf-test-egress:latest", dir).CombinedOutput(); err != nil {
		t.Skipf("não consegui montar a imagem do proxy: %s", out)
	}

	// "internet" de mentira: um servidor no host, alcançável como host.docker.internal
	ln, err := net.Listen("tcp", "0.0.0.0:0")
	if err != nil {
		t.Fatal(err)
	}
	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, "origem ok") })}
	go func() { _ = srv.Serve(ln) }()
	defer srv.Close()
	port := fmt.Sprint(ln.Addr().(*net.TCPAddr).Port)

	d := &execution.Docker{}
	name := fmt.Sprintf("lf-egress-test-%d", time.Now().UnixNano()%1_000_000)
	if err := d.EnsureEgressProxy(ctx, execution.EgressConfig{Name: name, Image: "lf-test-egress:latest", Allow: []string{"host.docker.internal"}, Ports: []string{port}}); err != nil {
		t.Fatal(err)
	}
	defer func() { _, _ = exec.Command("docker", "rm", "-f", name).CombinedOutput() }()
	if !d.EgressProxyRunning(ctx) {
		t.Fatal("o proxy deveria estar rodando")
	}
	// idempotente: a mesma lista reaproveita o contêiner
	if err := d.EnsureEgressProxy(ctx, execution.EgressConfig{Name: name, Image: "lf-test-egress:latest", Allow: []string{"host.docker.internal"}, Ports: []string{port}}); err != nil {
		t.Fatal(err)
	}

	prof := execution.Profile{Name: "agent", Image: "le-fabrique/worker:dev", CPUs: 1, MemoryMB: 256, PidsLimit: 128, TmpMB: 64, WorkspaceMB: 64, Network: execution.NetProxied}
	sb, err := d.Provision(ctx, execution.Spec{ProjectID: "egress-test", FeatureID: "egress-test", Profile: prof})
	if err != nil {
		t.Fatal(err)
	}
	network := sb.Network
	run := func(script string) execution.ExecResult {
		r, err := sb.Exec(ctx, execution.ExecSpec{Cmd: []string{"sh", "-c", script}, Env: execution.ProxyEnv(), Timeout: 40 * time.Second})
		if err != nil {
			t.Fatal(err)
		}
		return r
	}
	// permitido: passa pelo proxy
	if r := run(fmt.Sprintf("wget -q -T 10 -O- http://host.docker.internal:%s/", port)); !strings.Contains(r.Stdout, "origem ok") {
		t.Skipf("host.docker.internal indisponível neste Docker: exit=%d out=%q err=%q", r.ExitCode, r.Stdout, r.Stderr)
	}
	// fora da lista: o proxy recusa
	if r := run("wget -q -T 10 -O- http://example.com/"); r.ExitCode == 0 || strings.Contains(r.Stdout, "Example Domain") {
		t.Errorf("destino fora da lista deveria ser recusado: exit=%d out=%q", r.ExitCode, r.Stdout)
	}
	// sem proxy: não há rota para a internet (nem por IP)
	if r := run("env -u HTTP_PROXY -u http_proxy -u HTTPS_PROXY -u https_proxy wget -q -T 5 -O- http://1.1.1.1/"); r.ExitCode == 0 {
		t.Errorf("sem o proxy não deveria haver rota para a internet: out=%q", r.Stdout)
	}
	if err := sb.Destroy(ctx); err != nil {
		t.Fatalf("destruir: %v", err)
	}
	if out, _ := exec.Command("docker", "network", "ls", "-q", "--filter", "name="+network).Output(); strings.TrimSpace(string(out)) != "" {
		t.Errorf("a rede do terminal deveria ter sido removida: %s", out)
	}
	if !d.EgressProxyRunning(ctx) {
		t.Error("o proxy deveria continuar rodando depois de destruir o terminal")
	}
	// falha fechada: pedir rede filtrada sem proxy configurado
	if _, err := (&execution.Docker{}).Provision(ctx, execution.Spec{ProjectID: "egress-test", FeatureID: "egress-test", Profile: prof}); err == nil {
		t.Error("sem proxy configurado a criação do terminal filtrado deveria falhar")
	}
}
