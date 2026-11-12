package execution

import (
	"archive/tar"
	"bytes"
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
)

// Estes testes usam Docker de verdade e são pulados quando o runtime não existe.

func runtimeOrSkip(t *testing.T) *Docker {
	t.Helper()
	d := &Docker{}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if !d.Available(ctx) {
		t.Skip("Docker indisponível")
	}
	return d
}

func provision(t *testing.T, d *Docker, profile string, mut func(*Profile)) *Sandbox {
	t.Helper()
	p := Profiles[profile]
	if mut != nil {
		mut(&p)
	}
	s, err := d.Provision(t.Context(), Spec{ProjectID: "proj-" + t.Name()[:min(8, len(t.Name()))], FeatureID: "feat-1", Profile: p})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Destroy(context.Background()) })
	return s
}

func sh(t *testing.T, s *Sandbox, script string) ExecResult {
	t.Helper()
	res, err := s.Exec(t.Context(), ExecSpec{Cmd: []string{"sh", "-c", script}, Timeout: 60 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func TestSandboxRunsUnprivilegedAndHardened(t *testing.T) {
	d := runtimeOrSkip(t)
	s := provision(t, d, "base", nil)

	if r := sh(t, s, "id -u"); strings.TrimSpace(r.Stdout) != "10001" {
		t.Fatalf("deve rodar sem privilégios: %+v", r)
	}
	if r := sh(t, s, "touch /etc/x"); r.ExitCode == 0 {
		t.Fatal("o rootfs deve ser somente leitura")
	}
	if r := sh(t, s, "echo ok > /workspace/a && cat /workspace/a && echo t > /tmp/t"); r.ExitCode != 0 || !strings.Contains(r.Stdout, "ok") {
		t.Fatalf("workspace e /tmp devem aceitar escrita: %+v", r)
	}
	if r := sh(t, s, "grep -E 'CapEff|NoNewPrivs' /proc/self/status"); !strings.Contains(r.Stdout, "CapEff:\t0000000000000000") || !strings.Contains(r.Stdout, "NoNewPrivs:\t1") {
		t.Fatalf("sem capabilities e com no-new-privileges: %q", r.Stdout)
	}
	if r := sh(t, s, "test ! -e /var/run/docker.sock && test ! -d /Users && test ! -d /Volumes && echo isolado"); !strings.Contains(r.Stdout, "isolado") {
		t.Fatalf("sem socket do runtime nem diretórios do host: %+v", r)
	}

	// Quotas realmente aplicadas ao container.
	for format, want := range map[string]string{
		"{{.HostConfig.Memory}}":         "268435456",
		"{{.HostConfig.MemorySwap}}":     "268435456",
		"{{.HostConfig.PidsLimit}}":      "128",
		"{{.HostConfig.NanoCpus}}":       "1000000000",
		"{{.HostConfig.Privileged}}":     "false",
		"{{.HostConfig.ReadonlyRootfs}}": "true",
		"{{.HostConfig.NetworkMode}}":    "none",
		"{{.Config.User}}":               "10001:10001",
		"{{len .HostConfig.Binds}}":      "0",
	} {
		got, err := s.Inspect(t.Context(), format)
		if err != nil || got != want {
			t.Errorf("%s = %q (err %v), esperado %q", format, got, err, want)
		}
	}
	if got, _ := s.Inspect(t.Context(), "{{range .Mounts}}{{.Type}} {{end}}"); strings.Contains(got, "bind") {
		t.Fatalf("nenhuma montagem do host é permitida: %q", got)
	}
}

func TestWorkspaceHasDiskQuota(t *testing.T) {
	d := runtimeOrSkip(t)
	s := provision(t, d, "base", func(p *Profile) { p.WorkspaceMB = 16 })
	if r := sh(t, s, "dd if=/dev/zero of=/workspace/big bs=1M count=64 2>&1"); r.ExitCode == 0 {
		t.Fatalf("escrever além da quota do workspace deve falhar: %+v", r)
	}
}

func TestPidsLimitContainsForkBomb(t *testing.T) {
	d := runtimeOrSkip(t)
	s := provision(t, d, "base", func(p *Profile) { p.PidsLimit = 32 })
	r := sh(t, s, `i=0; while [ $i -lt 200 ]; do sleep 20 >/dev/null 2>&1 & i=$((i+1)); done; echo terminou`)
	if !strings.Contains(r.Stderr, "can't fork") && !strings.Contains(r.Stderr, "Resource temporarily unavailable") {
		t.Fatalf("o limite de processos não foi aplicado: %+v", r)
	}
	// O estouro fica contido: o host segue saudável e o container continua respondendo depois que os filhos morrem.
	if n, err := s.Inspect(t.Context(), "{{.HostConfig.PidsLimit}}"); err != nil || n != "32" {
		t.Fatalf("limite configurado: %q %v", n, err)
	}
	sh(t, s, "pkill sleep; true")
	time.Sleep(500 * time.Millisecond)
	if r := sh(t, s, "echo vivo"); strings.TrimSpace(r.Stdout) != "vivo" {
		t.Fatalf("o sandbox deveria seguir utilizável: %+v", r)
	}
}

func TestSandboxesDoNotSeeEachOther(t *testing.T) {
	d := runtimeOrSkip(t)
	a, b := provision(t, d, "base", nil), provision(t, d, "base", nil)
	if a.Volume == b.Volume || a.Container == b.Container {
		t.Fatal("cada feature recebe workspace e container próprios")
	}
	sh(t, a, "echo segredo-de-A > /workspace/a.txt; sleep 40 >/dev/null 2>&1 &")
	if r := sh(t, b, "ls -A /workspace | wc -l"); strings.TrimSpace(r.Stdout) != "0" {
		t.Fatalf("B não pode ver arquivos de A: %q", r.Stdout)
	}
	// O padrão [s]leep não casa com a própria linha de comando do grep.
	if r := sh(t, a, "ps | grep -c '[s]leep 40'"); strings.TrimSpace(r.Stdout) != "1" {
		t.Fatalf("controle positivo: A deveria ter o processo: %q", r.Stdout)
	}
	if r := sh(t, b, "ps | grep -c '[s]leep 40'"); strings.TrimSpace(r.Stdout) != "0" {
		t.Fatalf("B não pode ver processos de A: %q", r.Stdout)
	}
	if r := sh(t, b, "cat /workspace/a.txt 2>&1"); r.ExitCode == 0 {
		t.Fatal("B leu o arquivo de A")
	}
}

func TestNetworkNoneHasOnlyLoopback(t *testing.T) {
	d := runtimeOrSkip(t)
	s := provision(t, d, "base", nil)
	// Sem rede, a tabela de rotas fica só com o cabeçalho (os pseudo-dispositivos de túnel não têm rota).
	if r := sh(t, s, "tail -n +2 /proc/net/route | wc -l"); strings.TrimSpace(r.Stdout) != "0" {
		t.Fatalf("não pode haver rota alguma: %q", r.Stdout)
	}
	if r := sh(t, s, "wget -T 3 -q -O- http://1.1.1.1/ 2>&1"); r.ExitCode == 0 {
		t.Fatal("sem rede não pode haver saída para a internet")
	}
}

func TestIsolatedNetworkReachesNeitherOtherSandboxesNorHost(t *testing.T) {
	d := runtimeOrSkip(t)
	iso := func(p *Profile) { p.Network = NetIsolated }
	a, b := provision(t, d, "base", iso), provision(t, d, "base", iso)

	// Controle positivo: o servidor de B responde dentro do próprio B.
	sh(t, b, `(while true; do printf "HTTP/1.1 200 OK\r\nConnection: close\r\n\r\noi-de-B" | nc -l -p 8080 >/dev/null 2>&1; done) >/dev/null 2>&1 &`)
	time.Sleep(time.Second)
	if r := sh(t, b, "wget -T 3 -q -O- http://127.0.0.1:8080/"); !strings.Contains(r.Stdout, "oi-de-B") {
		t.Fatalf("o servidor de teste deveria responder em B: %+v", r)
	}
	ipB, err := b.Inspect(t.Context(), "{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}")
	if err != nil || ipB == "" {
		t.Fatalf("IP de B: %q %v", ipB, err)
	}
	if r := sh(t, a, "wget -T 3 -q -O- http://"+ipB+":8080/ 2>&1"); r.ExitCode == 0 {
		t.Fatalf("A alcançou B: %+v", r)
	}

	// Internet bloqueada na rede interna.
	if r := sh(t, a, "wget -T 3 -q -O- http://1.1.1.1/ 2>&1"); r.ExitCode == 0 {
		t.Fatal("rede interna não pode sair para a internet")
	}

	// Serviço do host escutando em todas as interfaces: o gateway da rede não pode levar até ele.
	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("host-vivo")) }))
	l, err := net.Listen("tcp", "0.0.0.0:0")
	if err != nil {
		t.Fatal(err)
	}
	srv.Listener = l
	srv.Start()
	defer srv.Close()
	port := l.Addr().(*net.TCPAddr).Port
	gw, err := d.networkGateway(t.Context(), a.Network)
	if err == nil && gw != "" {
		if r := sh(t, a, "wget -T 3 -q -O- http://"+gw+":"+itoa(port)+"/ 2>&1"); strings.Contains(r.Stdout, "host-vivo") {
			t.Fatalf("o sandbox alcançou um serviço do host pelo gateway: %q", r.Stdout)
		}
	}
	if r := sh(t, a, "wget -T 3 -q -O- http://host.docker.internal:"+itoa(port)+"/ 2>&1"); strings.Contains(r.Stdout, "host-vivo") {
		t.Fatalf("o sandbox alcançou o host por host.docker.internal: %q", r.Stdout)
	}
}

func itoa(n int) string { return strconv.Itoa(n) }

func (d *Docker) networkGateway(ctx context.Context, name string) (string, error) {
	out, _, err := d.run(ctx, nil, "network", "inspect", "--format", "{{(index .IPAM.Config 0).Gateway}}", name)
	return strings.TrimSpace(out), err
}

func tarOf(files map[string]string) *bytes.Reader {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	for name, body := range files {
		_ = tw.WriteHeader(&tar.Header{Name: name, Mode: 0o644, Size: int64(len(body))})
		_, _ = tw.Write([]byte(body))
	}
	_ = tw.Close()
	return bytes.NewReader(buf.Bytes())
}

// "Clone e build funcionam": o código chega ao workspace (extração de um commit
// exportado pelo supervisor) e compila como usuário sem privilégios, sem rede.
func TestSeedAndBuildInsideSandbox(t *testing.T) {
	d := runtimeOrSkip(t)
	s := provision(t, d, "go", nil)
	if err := s.Seed(t.Context(), tarOf(map[string]string{
		"go.mod":  "module exemplo\n\ngo 1.27\n",
		"main.go": "package main\n\nimport \"fmt\"\n\nfunc main() { fmt.Println(\"build-ok\") }\n",
	})); err != nil {
		t.Fatal(err)
	}
	env := map[string]string{"GOCACHE": "/tmp/gocache", "GOTOOLCHAIN": "local", "GOFLAGS": "-mod=mod"}
	res, err := s.Exec(t.Context(), ExecSpec{Cmd: []string{"go", "build", "-o", "/workspace/app", "."}, Env: env, Timeout: 3 * time.Minute})
	if err != nil || res.ExitCode != 0 {
		t.Fatalf("build: %+v %v", res, err)
	}
	run, err := s.Exec(t.Context(), ExecSpec{Cmd: []string{"/workspace/app"}})
	if err != nil || strings.TrimSpace(run.Stdout) != "build-ok" {
		t.Fatalf("execução: %+v %v", run, err)
	}
}

func TestExecTimeoutIsReported(t *testing.T) {
	d := runtimeOrSkip(t)
	s := provision(t, d, "base", nil)
	res, err := s.Exec(t.Context(), ExecSpec{Cmd: []string{"sleep", "30"}, Timeout: time.Second})
	if err != nil || !res.TimedOut {
		t.Fatalf("timeout deve ser sinalizado: %+v %v", res, err)
	}
}

func TestDestroyRemovesEverythingAndIsIdempotent(t *testing.T) {
	d := runtimeOrSkip(t)
	s, err := d.Provision(t.Context(), Spec{ProjectID: "p", FeatureID: "f", Profile: func() Profile { p := Profiles["base"]; p.Network = NetIsolated; return p }()})
	if err != nil {
		t.Fatal(err)
	}
	if !s.Exists(t.Context()) {
		t.Fatal("deveria existir")
	}
	if err := s.Destroy(t.Context()); err != nil {
		t.Fatal(err)
	}
	if err := s.Destroy(t.Context()); err != nil {
		t.Fatalf("destruir de novo deve ser inofensivo: %v", err)
	}
	if s.Exists(t.Context()) {
		t.Fatal("container não foi removido")
	}
	if _, _, err := d.run(t.Context(), nil, "volume", "inspect", s.Volume); err == nil {
		t.Fatal("workspace não foi removido")
	}
	if _, _, err := d.run(t.Context(), nil, "network", "inspect", s.Network); err == nil {
		t.Fatal("rede não foi removida")
	}
}

func countManaged(t *testing.T, d *Docker) string {
	t.Helper()
	var sb strings.Builder
	for _, kind := range [][]string{{"ps", "-a", "-q"}, {"volume", "ls", "-q"}, {"network", "ls", "-q"}} {
		out, _, _ := d.run(t.Context(), nil, append(kind, "--filter", "label=lf.managed=true")...)
		sb.WriteString(out)
	}
	return sb.String()
}

func TestInvalidSpecsAndFailedProvisionLeaveNothingBehind(t *testing.T) {
	d := runtimeOrSkip(t)
	before := countManaged(t, d)
	base := Profiles["base"]
	bad := []Spec{
		{ProjectID: "a b", FeatureID: "f", Profile: base},
		{ProjectID: "p", FeatureID: "f\n--privileged", Profile: base},
		{ProjectID: "p", FeatureID: "", Profile: base},
		{ProjectID: "p", FeatureID: "f", Profile: Profile{}},
		{ProjectID: "p", FeatureID: "f", Profile: func() Profile { p := base; p.Network = "host"; return p }()},
		{ProjectID: "p", FeatureID: "f", Profile: func() Profile { p := base; p.MemoryMB = 1; return p }()},
	}
	for i, sp := range bad {
		if _, err := d.Provision(t.Context(), sp); err == nil {
			t.Errorf("spec %d deveria ser recusada", i)
		}
	}
	// Falha no meio (imagem inexistente): volume e rede criados antes são desfeitos.
	p := base
	p.Image, p.Network = "lf-inexistente.invalid/x:1", NetIsolated
	if _, err := d.Provision(t.Context(), Spec{ProjectID: "p", FeatureID: "f", Profile: p}); err == nil {
		t.Fatal("imagem inexistente deveria falhar")
	}
	if after := countManaged(t, d); after != before {
		t.Fatalf("sobraram recursos: antes=%q depois=%q", before, after)
	}
}

func TestCancelKillsTheWholeProcessGroup(t *testing.T) {
	d := runtimeOrSkip(t)
	s := provision(t, d, "base", nil)
	r, err := s.Start(t.Context(), ExecSpec{Cmd: []string{"sh", "-c", "sleep 300 & sleep 301 & wait"}})
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Second)
	if res := sh(t, s, "ps -o args= | grep -c '^sleep 30'"); strings.TrimSpace(res.Stdout) != "2" {
		t.Fatalf("controle positivo: os dois filhos deveriam existir: %q", res.Stdout)
	}
	if err := r.Cancel(3 * time.Second); err != nil {
		t.Fatal(err)
	}
	if code, _ := r.Wait(); code == 0 {
		t.Fatal("o comando cancelado não pode sair com sucesso")
	}
	time.Sleep(500 * time.Millisecond)
	if res := sh(t, s, "ps -o args= | grep -c '^sleep 30'"); strings.TrimSpace(res.Stdout) != "0" {
		t.Fatalf("a árvore inteira deveria ter terminado: %q", res.Stdout)
	}
}

func TestCancelEscalatesToKillWhenTermIsIgnored(t *testing.T) {
	d := runtimeOrSkip(t)
	s := provision(t, d, "base", nil)
	r, err := s.Start(t.Context(), ExecSpec{Cmd: []string{"sh", "-c", `trap "" TERM; while true; do sleep 1; done`}})
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Second)
	start := time.Now()
	if err := r.Cancel(time.Second); err != nil {
		t.Fatalf("o SIGKILL deveria encerrar: %v", err)
	}
	if time.Since(start) > 8*time.Second {
		t.Fatal("o encerramento demorou demais")
	}
	time.Sleep(300 * time.Millisecond)
	if res := sh(t, s, "ps -o args= | grep -c '^sh -c trap'"); strings.TrimSpace(res.Stdout) != "0" {
		all := sh(t, s, "ps -o pid,pgid,stat,args")
		t.Fatalf("o processo que ignora SIGTERM deveria ter morrido: %q\n%s", res.Stdout, all.Stdout)
	}
}

func TestStreamDeliversOutputIncrementallyAndContextCancels(t *testing.T) {
	d := runtimeOrSkip(t)
	s := provision(t, d, "base", nil)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	r, err := s.Start(ctx, ExecSpec{Cmd: []string{"sh", "-c", "echo um; sleep 60; echo dois"}})
	if err != nil {
		t.Fatal(err)
	}
	got := make(chan string, 1)
	go func() {
		buf := make([]byte, 16)
		n, _ := r.Stdout.Read(buf)
		got <- string(buf[:n])
	}()
	select {
	case line := <-got:
		if !strings.Contains(line, "um") {
			t.Fatalf("primeira linha: %q", line)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("a saída deveria chegar antes do fim do comando")
	}
	cancel() // cancelar o contexto encerra o grupo
	done := make(chan struct{})
	go func() { _, _ = r.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(15 * time.Second):
		t.Fatal("o cancelamento do contexto não encerrou o comando")
	}
}

func TestAccountVolumesPersistAndStayIsolated(t *testing.T) {
	d := runtimeOrSkip(t)
	vol := "lf-acct-test-" + randomID()
	if err := d.EnsureVolume(t.Context(), vol); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = d.RemoveVolume(context.Background(), vol) }()
	spec := func(mounts ...Mount) Spec {
		return Spec{ProjectID: "p", FeatureID: "f", Profile: Profiles["base"], Mounts: mounts}
	}

	a, err := d.Provision(t.Context(), spec(Mount{Volume: vol, Target: "/accounts/a"}))
	if err != nil {
		t.Fatal(err)
	}
	if r := sh(t, a, "echo credencial-da-conta > /accounts/a/token && cat /accounts/a/token"); !strings.Contains(r.Stdout, "credencial-da-conta") {
		t.Fatalf("o diretório da conta deve aceitar escrita do usuário sem privilégios: %+v", r)
	}
	_ = a.Destroy(t.Context())

	// A configuração sobrevive ao sandbox; quem não monta o volume não a vê.
	b, err := d.Provision(t.Context(), spec(Mount{Volume: vol, Target: "/accounts/a", ReadOnly: true}))
	if err != nil {
		t.Fatal(err)
	}
	defer b.Destroy(context.Background())
	if r := sh(t, b, "cat /accounts/a/token"); !strings.Contains(r.Stdout, "credencial-da-conta") {
		t.Fatalf("a conta deveria persistir: %+v", r)
	}
	if r := sh(t, b, "echo x > /accounts/a/outro"); r.ExitCode == 0 {
		t.Fatal("montagem somente leitura não pode aceitar escrita")
	}
	c := provision(t, d, "base", nil)
	if r := sh(t, c, "ls /accounts 2>&1"); r.ExitCode == 0 {
		t.Fatalf("sandbox sem o volume não pode ver a conta: %+v", r)
	}
}

func TestInvalidMountsAreRejected(t *testing.T) {
	d := runtimeOrSkip(t)
	before := countManaged(t, d)
	for _, m := range []Mount{
		{Volume: "/Users/alguem", Target: "/accounts/a"}, // caminho do host
		{Volume: "../x", Target: "/accounts/a"},
		{Volume: "ok", Target: "/workspace"},
		{Volume: "ok", Target: "/workspace/sub"},
		{Volume: "ok", Target: "/etc/conta"},
		{Volume: "ok", Target: "/var/run/docker.sock"},
		{Volume: "ok", Target: "/accounts/../etc"},
		{Volume: "ok", Target: "relativo"},
		{Volume: "", Target: "/accounts/a"},
	} {
		if _, err := d.Provision(t.Context(), Spec{ProjectID: "p", FeatureID: "f", Profile: Profiles["base"], Mounts: []Mount{m}}); err == nil {
			t.Errorf("montagem %+v deveria ser recusada", m)
		}
	}
	if err := d.EnsureVolume(t.Context(), "../x"); err == nil {
		t.Error("nome de volume inválido")
	}
	if after := countManaged(t, d); after != before {
		t.Fatalf("recusas não podem criar recursos: %q → %q", before, after)
	}
}
