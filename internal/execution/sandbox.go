package execution

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// NetworkMode define o alcance de rede do sandbox.
type NetworkMode string

const (
	// NetNone: sem interface de rede além do loopback (padrão).
	NetNone NetworkMode = "none"
	// NetIsolated: rede interna própria do sandbox (sem saída para a internet,
	// sem alcançar outros sandboxes nem a rede do stack). Serve a serviços
	// auxiliares do mesmo sandbox.
	NetIsolated NetworkMode = "isolated"
)

// Profile descreve a stack e as quotas de um sandbox.
type Profile struct {
	Name        string
	Image       string
	CPUs        float64
	MemoryMB    int
	PidsLimit   int
	TmpMB       int // /tmp em tmpfs
	WorkspaceMB int // workspace em tmpfs com teto de tamanho (disco efêmero)
	Network     NetworkMode
}

// Perfis iniciais. As imagens são as já fixadas em docs/operacao/imagens-base.md.
// Os valores são provisórios: não há medida da VPS (T02-01).
var Profiles = map[string]Profile{
	"base": {Name: "base", Image: "alpine:3.22", CPUs: 1, MemoryMB: 256, PidsLimit: 128, TmpMB: 64, WorkspaceMB: 256, Network: NetNone},
	"go":   {Name: "go", Image: "golang:1.27.3-alpine", CPUs: 2, MemoryMB: 1024, PidsLimit: 512, TmpMB: 512, WorkspaceMB: 1024, Network: NetNone},
	"node": {Name: "node", Image: "node:24-alpine", CPUs: 2, MemoryMB: 1024, PidsLimit: 512, TmpMB: 512, WorkspaceMB: 1024, Network: NetNone},
}

// Identidade sem privilégios usada dentro do sandbox.
const (
	sandboxUID = "10001"
	workspace  = "/workspace"
)

var idPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)

// Mount monta um volume nomeado (nunca um caminho do host) no sandbox. Serve ao
// diretório de configuração de uma conta de CLI de IA.
type Mount struct {
	Volume   string
	Target   string
	ReadOnly bool
}

var (
	volumeName  = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]{0,127}$`)
	mountTarget = regexp.MustCompile(`^/[A-Za-z0-9_./-]{1,200}$`)
)

func validMount(m Mount) bool {
	if !volumeName.MatchString(m.Volume) || !mountTarget.MatchString(m.Target) || strings.Contains(m.Target, "..") {
		return false
	}
	for _, reserved := range []string{workspace, "/tmp", "/proc", "/sys", "/dev", "/etc", "/var/run"} {
		if m.Target == reserved || strings.HasPrefix(m.Target, reserved+"/") {
			return false
		}
	}
	return true
}

// Spec identifica o sandbox: um por projeto e feature.
type Spec struct {
	ProjectID string
	FeatureID string
	Profile   Profile
	Mounts    []Mount
}

// Docker é o supervisor de sandboxes sobre o CLI do Docker.
type Docker struct {
	Bin string // padrão "docker"
}

func (d *Docker) bin() string {
	if d.Bin == "" {
		return "docker"
	}
	return d.Bin
}

func (d *Docker) run(ctx context.Context, stdin io.Reader, args ...string) (string, string, error) {
	cmd := exec.CommandContext(ctx, d.bin(), args...)
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr, cmd.Stdin = &out, &errb, stdin
	err := cmd.Run()
	return out.String(), errb.String(), err
}

// Available informa se o runtime responde.
func (d *Docker) Available(ctx context.Context) bool {
	_, _, err := d.run(ctx, nil, "info", "--format", "{{.ServerVersion}}")
	return err == nil
}

// Sandbox é um ambiente provisionado.
type Sandbox struct {
	ID        string
	Container string
	Volume    string
	Network   string // vazio quando NetNone
	Profile   Profile
	d         *Docker

	qmu        sync.Mutex
	quarantine string // motivo; vazio = utilizável
}

// ErrQuarantined: o estado do sandbox é incerto (um processo pode ainda estar
// vivo ou escrevendo) e ele não pode ser reutilizado até ser verificado.
var ErrQuarantined = errors.New("sandbox em quarentena: estado incerto")

// Quarantine bloqueia novas execuções neste sandbox/workspace.
func (s *Sandbox) Quarantine(reason string) {
	s.qmu.Lock()
	s.quarantine = reason
	s.qmu.Unlock()
}

// Quarantined informa se o sandbox está bloqueado e por quê.
func (s *Sandbox) Quarantined() (bool, string) {
	s.qmu.Lock()
	defer s.qmu.Unlock()
	return s.quarantine != "", s.quarantine
}

func (s *Sandbox) usable() error {
	if q, why := s.Quarantined(); q {
		return fmt.Errorf("%w (%s)", ErrQuarantined, why)
	}
	return nil
}

func randomID() string {
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// ErrInvalidSpec indica identificadores ou perfil inválidos.
var ErrInvalidSpec = errors.New("especificação de sandbox inválida")

// Provision cria o container efêmero, o volume de workspace próprio e, se
// pedido, a rede interna. Em qualquer falha desfaz o que criou.
func (d *Docker) Provision(ctx context.Context, spec Spec) (*Sandbox, error) {
	p := spec.Profile
	if !idPattern.MatchString(spec.ProjectID) || !idPattern.MatchString(spec.FeatureID) ||
		p.Image == "" || p.CPUs <= 0 || p.MemoryMB < 16 || p.PidsLimit < 8 || p.WorkspaceMB < 8 || p.TmpMB < 1 {
		return nil, ErrInvalidSpec
	}
	if p.Network != NetNone && p.Network != NetIsolated {
		return nil, ErrInvalidSpec
	}
	for _, m := range spec.Mounts {
		if !validMount(m) {
			return nil, ErrInvalidSpec
		}
	}
	id := randomID()
	s := &Sandbox{ID: id, Container: "lf-sbx-" + id, Volume: "lf-ws-" + id, Profile: p, d: d}
	fail := func(err error) (*Sandbox, error) {
		_ = s.Destroy(context.WithoutCancel(ctx))
		return nil, err
	}
	labels := []string{"--label", "lf.managed=true", "--label", "lf.project=" + spec.ProjectID, "--label", "lf.feature=" + spec.FeatureID}

	// Workspace: volume exclusivo, em tmpfs com teto de tamanho e dono sem privilégios.
	volArgs := append([]string{"volume", "create", "--driver", "local",
		"--opt", "type=tmpfs", "--opt", "device=tmpfs",
		"--opt", fmt.Sprintf("o=size=%dm,uid=%s,gid=%s,mode=0700", p.WorkspaceMB, sandboxUID, sandboxUID)}, labels...)
	volArgs = append(volArgs, s.Volume)
	if _, e, err := d.run(ctx, nil, volArgs...); err != nil {
		return fail(fmt.Errorf("criar workspace: %s", strings.TrimSpace(e)))
	}

	network := "none"
	if p.Network == NetIsolated {
		s.Network = "lf-net-" + id
		if _, e, err := d.run(ctx, nil, append([]string{"network", "create", "--internal", "--driver", "bridge"}, append(labels, s.Network)...)...); err != nil {
			return fail(fmt.Errorf("criar rede: %s", strings.TrimSpace(e)))
		}
		network = s.Network
	}

	args := []string{"create", "--name", s.Container,
		"--user", sandboxUID + ":" + sandboxUID,
		"--read-only", "--cap-drop", "ALL", "--security-opt", "no-new-privileges:true",
		"--init", "--pids-limit", strconv.Itoa(p.PidsLimit),
		"--memory", fmt.Sprintf("%dm", p.MemoryMB), "--memory-swap", fmt.Sprintf("%dm", p.MemoryMB),
		"--cpus", strconv.FormatFloat(p.CPUs, 'f', -1, 64),
		"--network", network,
		"--tmpfs", fmt.Sprintf("/tmp:rw,nosuid,size=%dm", p.TmpMB),
		"--mount", fmt.Sprintf("type=volume,src=%s,dst=%s", s.Volume, workspace),
		"--workdir", workspace, "--env", "HOME=/tmp",
		"--hostname", "sandbox",
	}
	for _, m := range spec.Mounts {
		opt := fmt.Sprintf("type=volume,src=%s,dst=%s", m.Volume, m.Target)
		if m.ReadOnly {
			opt += ",readonly"
		}
		args = append(args, "--mount", opt)
	}
	args = append(args, labels...)
	args = append(args, p.Image, "sleep", "2147483647")
	if _, e, err := d.run(ctx, nil, args...); err != nil {
		return fail(fmt.Errorf("criar container: %s", strings.TrimSpace(e)))
	}
	if _, e, err := d.run(ctx, nil, "start", s.Container); err != nil {
		return fail(fmt.Errorf("iniciar container: %s", strings.TrimSpace(e)))
	}
	return s, nil
}

// ExecSpec descreve um comando dentro do sandbox. Env NÃO deve conter segredos:
// valores de -e aparecem na lista de processos do host.
type ExecSpec struct {
	Cmd     []string
	Env     map[string]string
	Dir     string // relativo a /workspace ou absoluto dentro do sandbox
	Stdin   io.Reader
	Timeout time.Duration
}

// ExecResult é o resultado de um comando.
type ExecResult struct {
	ExitCode int
	Stdout   string
	Stderr   string
	TimedOut bool
}

const maxOutput = 4 << 20

type limitBuffer struct {
	bytes.Buffer
	truncated bool
}

func (l *limitBuffer) Write(p []byte) (int, error) {
	if l.Len()+len(p) > maxOutput {
		l.truncated = true
		p = p[:max(0, maxOutput-l.Len())]
	}
	l.Buffer.Write(p)
	return len(p), nil
}

var envKey = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]{0,63}$`)

// Exec roda um comando como o usuário sem privilégios do sandbox.
func (s *Sandbox) Exec(ctx context.Context, es ExecSpec) (ExecResult, error) {
	if len(es.Cmd) == 0 {
		return ExecResult{}, ErrInvalidSpec
	}
	if err := s.usable(); err != nil {
		return ExecResult{}, err
	}
	if es.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, es.Timeout)
		defer cancel()
	}
	args := []string{"exec", "-i", "--user", sandboxUID + ":" + sandboxUID}
	if es.Dir != "" {
		args = append(args, "--workdir", es.Dir)
	}
	for k, v := range es.Env {
		if !envKey.MatchString(k) {
			return ExecResult{}, ErrInvalidSpec
		}
		args = append(args, "--env", k+"="+v)
	}
	args = append(args, s.Container)
	args = append(args, es.Cmd...)
	cmd := exec.CommandContext(ctx, s.d.bin(), args...)
	var out, errb limitBuffer
	cmd.Stdout, cmd.Stderr, cmd.Stdin = &out, &errb, es.Stdin
	err := cmd.Run()
	res := ExecResult{Stdout: out.String(), Stderr: errb.String()}
	var ee *exec.ExitError
	switch {
	case err == nil:
	case errors.Is(ctx.Err(), context.DeadlineExceeded):
		res.TimedOut, res.ExitCode = true, -1
		return res, nil
	case errors.As(err, &ee):
		res.ExitCode = ee.ExitCode()
	default:
		return res, fmt.Errorf("exec: %w", err)
	}
	return res, nil
}

// Seed extrai um tar (ex.: git archive do commit) no workspace como usuário sem privilégios.
func (s *Sandbox) Seed(ctx context.Context, tarball io.Reader) error {
	res, err := s.Exec(ctx, ExecSpec{Cmd: []string{"tar", "-x", "-C", workspace}, Stdin: tarball, Timeout: 2 * time.Minute})
	if err != nil {
		return err
	}
	if res.ExitCode != 0 {
		return fmt.Errorf("extrair código: %s", strings.TrimSpace(res.Stderr))
	}
	return nil
}

// Destroy remove container, workspace e rede; é idempotente.
func (s *Sandbox) Destroy(ctx context.Context) error {
	var errs []string
	for _, a := range [][]string{
		{"rm", "-f", "-v", s.Container},
		{"volume", "rm", "-f", s.Volume},
	} {
		if _, e, err := s.d.run(ctx, nil, a...); err != nil && !strings.Contains(e, "No such") {
			errs = append(errs, strings.TrimSpace(e))
		}
	}
	if s.Network != "" {
		if _, e, err := s.d.run(ctx, nil, "network", "rm", s.Network); err != nil && !strings.Contains(e, "not found") && !strings.Contains(e, "No such") {
			errs = append(errs, strings.TrimSpace(e))
		}
	}
	if len(errs) > 0 {
		return errors.New("destruir sandbox: " + strings.Join(errs, "; "))
	}
	return nil
}

// Exists informa se o container ainda existe (para provar a limpeza).
func (s *Sandbox) Exists(ctx context.Context) bool {
	_, _, err := s.d.run(ctx, nil, "inspect", "--type", "container", s.Container)
	return err == nil
}

// Inspect devolve o valor de um template de `docker inspect` do container.
func (s *Sandbox) Inspect(ctx context.Context, format string) (string, error) {
	out, e, err := s.d.run(ctx, nil, "inspect", "--format", format, s.Container)
	if err != nil {
		return "", fmt.Errorf("inspect: %s", strings.TrimSpace(e))
	}
	return strings.TrimSpace(out), nil
}

// EnsureVolume cria (se faltar) um volume nomeado e entrega a posse ao usuário
// sem privilégios do sandbox. Idempotente.
func (d *Docker) EnsureVolume(ctx context.Context, name string) error {
	if !volumeName.MatchString(name) {
		return ErrInvalidSpec
	}
	if _, e, err := d.run(ctx, nil, "volume", "create", "--label", "lf.managed=true", name); err != nil {
		return fmt.Errorf("criar volume: %s", strings.TrimSpace(e))
	}
	_, e, err := d.run(ctx, nil, "run", "--rm", "--network", "none", "--cap-drop", "ALL", "--cap-add", "CHOWN",
		"--security-opt", "no-new-privileges:true", "--mount", "type=volume,src="+name+",dst=/v",
		"alpine:3.22", "chown", sandboxUID+":"+sandboxUID, "/v")
	if err != nil {
		return fmt.Errorf("preparar volume: %s", strings.TrimSpace(e))
	}
	return nil
}

// RemoveVolume apaga um volume nomeado (idempotente).
func (d *Docker) RemoveVolume(ctx context.Context, name string) error {
	if !volumeName.MatchString(name) {
		return ErrInvalidSpec
	}
	if _, e, err := d.run(ctx, nil, "volume", "rm", "-f", name); err != nil && !strings.Contains(e, "No such") {
		return fmt.Errorf("remover volume: %s", strings.TrimSpace(e))
	}
	return nil
}

// ManagedContainers lista os containers EM EXECUÇÃO que a fábrica criou para a feature do projeto
// (pelos rótulos lf.*). Serve à recuperação, que não guarda o objeto Sandbox de um worker que sumiu.
func (d *Docker) ManagedContainers(ctx context.Context, projectID, featureID string) ([]string, error) {
	if !idPattern.MatchString(projectID) || !idPattern.MatchString(featureID) {
		return nil, ErrInvalidSpec
	}
	out, e, err := d.run(ctx, nil, "ps", "--filter", "label=lf.managed=true", "--filter", "label=lf.project="+projectID,
		"--filter", "label=lf.feature="+featureID, "--format", "{{.Names}}")
	if err != nil {
		return nil, fmt.Errorf("listar sandboxes: %s", strings.TrimSpace(e))
	}
	var names []string
	for _, n := range strings.Fields(out) {
		if strings.HasPrefix(n, "lf-sbx-") {
			names = append(names, n)
		}
	}
	return names, nil
}

// Attach reencontra um sandbox existente pelo nome do container (só para verificação e parada;
// não recria nada).
func (d *Docker) Attach(container string) *Sandbox {
	id := strings.TrimPrefix(container, "lf-sbx-")
	return &Sandbox{ID: id, Container: container, Volume: "lf-ws-" + id, d: d}
}
