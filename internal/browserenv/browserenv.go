// Package browserenv provisiona o navegador isolado do review visual (AJ08-02): um container Chromium
// por execução, em rede interna própria (sem saída para a internet, sem alcançar outras execuções, o
// host nem o plano de controle), sem montagens do host e com quotas. As ferramentas de navegação só
// aceitam as origens autorizadas da aplicação de teste. A Sentinela recolhe containers, redes e volumes
// temporários por rótulo de execução, de forma repetível, em sucesso, falha, cancelamento ou queda.
package browserenv

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

// Erros.
var (
	ErrForbiddenURL = errors.New("destino fora das origens autorizadas da aplicação de teste")
	ErrInvalidSpec  = errors.New("especificação do navegador inválida")
	ErrNotReady     = errors.New("o navegador não ficou pronto")
)

// Runtime executa comandos do runtime de containers (o CLI do Docker).
type Runtime interface {
	Run(ctx context.Context, args ...string) (stdout string, err error)
}

// DockerCLI é o Runtime sobre o binário `docker`.
type DockerCLI struct{ Bin string }

// Run implementa Runtime.
func (d DockerCLI) Run(ctx context.Context, args ...string) (string, error) {
	bin := d.Bin
	if bin == "" {
		bin = "docker"
	}
	cmd := exec.CommandContext(ctx, bin, args...)
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	if err := cmd.Run(); err != nil {
		m := strings.TrimSpace(errb.String())
		if len(m) > 300 {
			m = m[:300]
		}
		return out.String(), fmt.Errorf("docker %s: %v: %s", args[0], err, m)
	}
	return strings.TrimSpace(out.String()), nil
}

// Spec identifica a execução e o navegador.
type Spec struct {
	ProjectID   string
	ExecutionID string
	Image       string // imagem do Chromium com o servidor MCP (decisão de implantação)
	MemoryMB    int
	CPUs        float64
	Pids        int
	Allowed     []string // origens autorizadas (http://app:3000) acessíveis na rede interna da execução
	Command     []string // comando do container (padrão: o da imagem)
}

var idPat = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)
var imagePat = regexp.MustCompile(`^[a-z0-9][a-z0-9._/:@-]{0,200}$`)

func (s Spec) validate() error {
	if !idPat.MatchString(s.ProjectID) || !idPat.MatchString(s.ExecutionID) || !imagePat.MatchString(s.Image) {
		return ErrInvalidSpec
	}
	for _, o := range s.Allowed {
		if _, err := ParseOrigin(o); err != nil {
			return err
		}
	}
	return nil
}

// Origin é uma origem autorizada.
type Origin struct{ Scheme, Host string }

// ParseOrigin valida a origem autorizada: http/https, sem credenciais, caminho ou consulta, e nunca o
// loopback, o link-local (metadados da nuvem) ou um IP público — só nomes da rede interna da execução.
func ParseOrigin(raw string) (Origin, error) {
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.User != nil || (u.Path != "" && u.Path != "/") || u.RawQuery != "" {
		return Origin{}, fmt.Errorf("%w: %q", ErrForbiddenURL, raw)
	}
	h := u.Hostname()
	if ip := net.ParseIP(h); ip != nil {
		return Origin{}, fmt.Errorf("%w: endereço IP direto não é permitido (%s)", ErrForbiddenURL, h)
	}
	if h == "localhost" || strings.HasSuffix(h, ".localhost") || strings.Contains(h, ".") {
		return Origin{}, fmt.Errorf("%w: use o nome da aplicação na rede interna da execução (%s)", ErrForbiddenURL, h)
	}
	return Origin{Scheme: u.Scheme, Host: strings.ToLower(u.Host)}, nil
}

// CheckURL confere um destino de navegação contra as origens autorizadas.
func CheckURL(allowed []Origin, raw string) error {
	u, err := url.Parse(raw)
	if err != nil || u.User != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return fmt.Errorf("%w: %q", ErrForbiddenURL, raw)
	}
	for _, o := range allowed {
		if o.Scheme == u.Scheme && o.Host == strings.ToLower(u.Host) {
			return nil
		}
	}
	return fmt.Errorf("%w: %s", ErrForbiddenURL, u.Host)
}

// Env é um navegador provisionado.
type Env struct {
	Spec      Spec
	Container string
	Network   string
	Origins   []Origin
	rt        Runtime
}

func names(execID string) (container, network string) {
	return "lf-brw-" + execID, "lf-brw-net-" + execID
}

func labels(s Spec) []string {
	return []string{"--label", "lf.managed=true", "--label", "lf.kind=browser", "--label", "lf.project=" + s.ProjectID, "--label", "lf.exec=" + s.ExecutionID}
}

// Provision cria a rede interna e o container do navegador. Falha no meio recolhe o que criou.
func Provision(ctx context.Context, rt Runtime, s Spec) (*Env, error) {
	if err := s.validate(); err != nil {
		return nil, err
	}
	var origins []Origin
	for _, o := range s.Allowed {
		po, _ := ParseOrigin(o)
		origins = append(origins, po)
	}
	container, network := names(s.ExecutionID)
	e := &Env{Spec: s, Container: container, Network: network, Origins: origins, rt: rt}
	args := append([]string{"network", "create", "--internal", "--driver", "bridge"}, labels(s)...)
	if _, err := rt.Run(ctx, append(args, network)...); err != nil {
		return nil, fmt.Errorf("rede interna: %w", err)
	}
	mem, cpus, pids := s.MemoryMB, s.CPUs, s.Pids
	if mem <= 0 {
		mem = 1024
	}
	if cpus <= 0 {
		cpus = 1
	}
	if pids <= 0 {
		pids = 512
	}
	run := append([]string{"run", "-d", "--name", container, "--network", network, "--read-only", "--cap-drop", "ALL",
		"--security-opt", "no-new-privileges", "--memory", fmt.Sprintf("%dm", mem), "--cpus", fmt.Sprintf("%.1f", cpus), "--pids-limit", fmt.Sprintf("%d", pids),
		"--tmpfs", "/tmp:rw,size=256m", "--user", "10001"}, labels(s)...)
	run = append(run, s.Image)
	run = append(run, s.Command...)
	if _, err := rt.Run(ctx, run...); err != nil {
		_ = e.Cleanup(context.WithoutCancel(ctx))
		return nil, fmt.Errorf("container do navegador: %w", err)
	}
	return e, nil
}

// Ready espera o container estar em execução até o prazo.
func (e *Env) Ready(ctx context.Context, wait time.Duration) error {
	deadline := time.Now().Add(wait)
	for {
		out, err := e.rt.Run(ctx, "inspect", "--format", "{{.State.Running}}", e.Container)
		if err == nil && strings.TrimSpace(out) == "true" {
			return nil
		}
		if time.Now().After(deadline) || ctx.Err() != nil {
			return ErrNotReady
		}
		time.Sleep(200 * time.Millisecond)
	}
}

// Attach conecta um serviço de teste (a aplicação) à rede interna da execução com um alias.
func (e *Env) Attach(ctx context.Context, container, alias string) error {
	if !idPat.MatchString(alias) {
		return ErrInvalidSpec
	}
	_, err := e.rt.Run(ctx, "network", "connect", "--alias", alias, e.Network, container)
	return err
}

// Check valida o destino pedido pelo revisor.
func (e *Env) Check(raw string) error { return CheckURL(e.Origins, raw) }

// Cleanup recolhe container, rede e volumes DESTA execução. Repetível: o que já não existe não é erro.
func (e *Env) Cleanup(ctx context.Context) error {
	return CleanupExecution(ctx, e.rt, e.Spec.ExecutionID)
}

// CleanupExecution recolhe os recursos rotulados da execução (container, rede e volumes), na ordem
// correta, sem tocar em recursos de outras execuções.
func CleanupExecution(ctx context.Context, rt Runtime, execID string) error {
	if !idPat.MatchString(execID) {
		return ErrInvalidSpec
	}
	var errs []string
	label := "label=lf.exec=" + execID
	if out, err := rt.Run(ctx, "ps", "-aq", "--filter", "label=lf.managed=true", "--filter", label); err == nil {
		for _, id := range strings.Fields(out) {
			if _, err := rt.Run(ctx, "rm", "-f", "-v", id); err != nil {
				errs = append(errs, err.Error())
			}
		}
	} else {
		errs = append(errs, err.Error())
	}
	if out, err := rt.Run(ctx, "network", "ls", "-q", "--filter", "label=lf.managed=true", "--filter", label); err == nil {
		for _, id := range strings.Fields(out) {
			if _, err := rt.Run(ctx, "network", "rm", id); err != nil {
				errs = append(errs, err.Error())
			}
		}
	} else {
		errs = append(errs, err.Error())
	}
	if out, err := rt.Run(ctx, "volume", "ls", "-q", "--filter", "label=lf.managed=true", "--filter", label); err == nil {
		for _, id := range strings.Fields(out) {
			if _, err := rt.Run(ctx, "volume", "rm", "-f", id); err != nil {
				errs = append(errs, err.Error())
			}
		}
	}
	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "; "))
	}
	return nil
}

// CollectOrphans recolhe tudo o que a fábrica criou para execuções que já não estão vivas (queda do
// worker). `live` informa as execuções ainda em andamento.
func CollectOrphans(ctx context.Context, rt Runtime, live map[string]bool) ([]string, error) {
	out, err := rt.Run(ctx, "ps", "-a", "--filter", "label=lf.kind=browser", "--format", `{{.Label "lf.exec"}}`)
	if err != nil {
		return nil, err
	}
	nets, _ := rt.Run(ctx, "network", "ls", "--filter", "label=lf.kind=browser", "--format", `{{.Label "lf.exec"}}`)
	seen := map[string]bool{}
	var collected []string
	for _, id := range strings.Fields(out + "\n" + nets) {
		if seen[id] || live[id] || !idPat.MatchString(id) {
			continue
		}
		seen[id] = true
		if err := CleanupExecution(ctx, rt, id); err != nil {
			return collected, err
		}
		collected = append(collected, id)
	}
	return collected, nil
}
