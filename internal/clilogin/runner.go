package clilogin

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
	"github.com/ViniciusBernucci/le-fabrique/internal/security"
)

const (
	// AccountMount é onde o volume da conta aparece dentro do container (o mesmo do sandbox, ADR-0011).
	AccountMount = "/accounts/current"
	// DefaultImage é a imagem com os clientes oficiais (infra/images/Dockerfile.cli).
	DefaultImage = "le-fabrique/cli:dev"
	imageUID     = "10002"
	maxOutput    = 64 << 10
)

// Runner roda o login oficial dos clientes no Docker. Vive no worker: é ele que tem acesso ao Docker.
type Runner struct {
	Store *Store
	Vault *security.Vault
	Log   *slog.Logger
	// Bin é o executável do Docker (padrão "docker"; testes trocam por um falso).
	Bin string
	// Image é a imagem dos clientes. ContextDir, se existir com um Dockerfile.cli, permite construí-la
	// sozinho quando faltar (desenvolvimento); vazio = só usa uma imagem já existente.
	Image      string
	ContextDir string
	// PollEvery é o ritmo de olhar a saída do cliente e o estado no banco (padrão 1s).
	PollEvery time.Duration
	// MaxParallel limita logins simultâneos (padrão 2).
	MaxParallel int

	once sync.Once
	sem  chan struct{}
}

func (r *Runner) init() {
	r.once.Do(func() {
		if r.Bin == "" {
			r.Bin = "docker"
		}
		if r.Image == "" {
			r.Image = DefaultImage
		}
		if r.PollEvery <= 0 {
			r.PollEvery = time.Second
		}
		if r.MaxParallel <= 0 {
			r.MaxParallel = 2
		}
		r.sem = make(chan struct{}, r.MaxParallel)
	})
}

func volumeName(connID string) string     { return "lf-acct-" + connID }
func containerName(loginID string) string { return "lf-login-" + loginID }

// Sweep encerra o que sobrou de uma execução anterior do worker (container e volume) e marca esses
// logins como interrompidos. Chamar uma vez ao subir.
func (r *Runner) Sweep(ctx context.Context) error {
	r.init()
	open, err := r.Store.Leftovers(ctx)
	if err != nil {
		return err
	}
	for _, l := range open {
		r.cleanup(ctx, l, true)
		_ = r.Store.Finish(ctx, l.ID, StateFailed, "O login foi interrompido porque o worker reiniciou. Tente de novo.")
	}
	return nil
}

// Tick pega os logins pendentes (até o limite de paralelismo) e os executa em segundo plano; também
// encerra os que passaram do prazo. É o corpo do job periódico do worker.
func (r *Runner) Tick(ctx context.Context) (string, error) {
	r.init()
	started := 0
	for {
		select {
		case r.sem <- struct{}{}:
		default:
			return note(started), nil
		}
		l, ok, err := r.Store.Claim(ctx)
		if err != nil || !ok {
			<-r.sem
			return note(started), err
		}
		started++
		go func() {
			defer func() { <-r.sem }()
			// O login precisa sobreviver ao ciclo do job, então não herda o contexto dele.
			r.run(context.WithoutCancel(ctx), l)
		}()
	}
}

func note(n int) string {
	if n == 0 {
		return ""
	}
	return fmt.Sprintf("%d login(s) de conta iniciado(s)", n)
}

// run conduz um login do início ao fim. Todo caminho termina com Finish (apagando o desafio).
func (r *Runner) run(ctx context.Context, l Login) {
	defer func() {
		if rec := recover(); rec != nil {
			r.fail(ctx, l, "Erro interno ao conectar a conta.")
		}
	}()
	if err := r.docker(ctx, 20*time.Second, nil, "version", "--format", "{{.Server.Version}}"); err != nil {
		r.fail(ctx, l, "O worker não consegue falar com o Docker. Monte o docker.sock no worker (ADR-0065) e tente de novo.")
		return
	}
	if err := r.ensureImage(ctx); err != nil {
		r.fail(ctx, l, err.Error())
		return
	}
	vol := volumeName(l.ConnectionID)
	if err := r.docker(ctx, 30*time.Second, nil, "volume", "create", vol); err != nil {
		r.fail(ctx, l, "Não consegui criar o volume da conta.")
		return
	}
	// O volume nasce de root; o cliente roda sem privilégios e precisa poder gravar o login nele.
	if err := r.docker(ctx, 60*time.Second, nil, "run", "--rm", "--user", "0", "--cap-drop", "ALL", "--cap-add", "CHOWN",
		"-v", vol+":"+AccountMount, "--entrypoint", "chown", r.Image, "-R", imageUID+":"+imageUID, AccountMount); err != nil {
		r.fail(ctx, l, "Não consegui preparar o volume da conta.")
		return
	}

	out, err := r.login(ctx, l)
	switch {
	case errors.Is(err, errStopped):
		return // cancelado ou vencido: quem parou já registrou o desfecho
	case err != nil:
		msg := "O login não foi concluído."
		if last := LastLine(out); last != "" {
			msg += " " + last
		}
		r.fail(ctx, l, msg)
		return
	}

	_ = r.Store.SetState(ctx, l.ID, StateVerifying)
	if ok := r.verify(ctx, l); !ok {
		r.fail(ctx, l, "O cliente terminou, mas o login não foi confirmado. Tente de novo.")
		return
	}
	// A credencial real fica no volume; o cofre exige um segredo, e aqui ele é só um marcador (ADR-0011).
	_, err = r.Vault.Create(ctx, security.NewConnection{ID: l.ConnectionID, Kind: "cli", Name: l.Name, Target: string(l.Provider),
		Secret: security.NewSecret("conta-" + string(l.Provider) + ": login oficial no volume " + vol)})
	switch {
	case errors.Is(err, persistence.ErrDuplicate):
		r.fail(ctx, l, "Já existe uma conexão ativa com esse nome. Escolha outro nome.")
		return
	case err != nil:
		r.fail(ctx, l, "O login funcionou, mas não consegui guardar a conexão no cofre.")
		return
	}
	_ = r.Store.Finish(ctx, l.ID, StateDone, "")
	r.Log.Info("conta de assinatura conectada", "provider", string(l.Provider), "login", l.ID)
}

var errStopped = errors.New("login interrompido")

// loginArgs é o comando oficial de cada cliente e a variável que aponta para o volume da conta.
func loginArgs(p Provider) (env string, cmd []string) {
	if p == Claude {
		return "CLAUDE_CONFIG_DIR=" + AccountMount, []string{"claude", "auth", "login", "--claudeai"}
	}
	return "CODEX_HOME=" + AccountMount, []string{"codex", "login", "--device-auth"}
}

func statusArgs(p Provider) (env string, cmd []string) {
	if p == Claude {
		return "CLAUDE_CONFIG_DIR=" + AccountMount, []string{"claude", "auth", "status"}
	}
	return "CODEX_HOME=" + AccountMount, []string{"codex", "login", "status"}
}

// hardening são as restrições de todo container de login: sem privilégios e com limites.
var hardening = []string{"--cap-drop", "ALL", "--security-opt", "no-new-privileges", "--pids-limit", "256", "--memory", "768m"}

// login roda o cliente oficial e conduz a pessoa até o fim: publica o desafio, entrega o código colado
// (Claude) e vigia cancelamento e prazo. Devolve a saída (só para explicar uma falha).
func (r *Runner) login(ctx context.Context, l Login) (string, error) {
	env, cmdline := loginArgs(l.Provider)
	args := append([]string{"run", "--rm", "-i", "--name", containerName(l.ID)}, hardening...)
	args = append(args, "-v", volumeName(l.ConnectionID)+":"+AccountMount, "-e", env, "-e", "HOME=/tmp", r.Image)
	args = append(args, cmdline...)

	cmd := exec.Command(r.Bin, args...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return "", err
	}
	buf := &limitedBuf{max: maxOutput}
	cmd.Stdout, cmd.Stderr = buf, buf
	if err := cmd.Start(); err != nil {
		return "", err
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	defer stdin.Close()

	published := false
	tick := time.NewTicker(r.PollEvery)
	defer tick.Stop()
	for {
		select {
		case err := <-done:
			return buf.String(), err
		case <-ctx.Done():
			r.stopAndWait(l, done)
			return buf.String(), errStopped
		case <-tick.C:
		}
		cur, err := r.Store.Get(ctx, l.ID)
		if err == nil && (!cur.Open() || r.Store.now().After(cur.ExpiresAt)) {
			r.stopAndWait(l, done)
			if cur.Open() {
				_ = r.Store.Finish(ctx, l.ID, StateFailed, "O tempo para autorizar acabou. Tente de novo.")
			}
			r.cleanup(ctx, l, true)
			return buf.String(), errStopped
		}
		if !published {
			if c, ok := Extract(l.Provider, buf.String()); ok {
				if err := r.Store.SetChallenge(ctx, l.ID, c); err == nil {
					published = true
				}
			}
		}
		if published && l.Provider == Claude {
			if code, ok, _ := r.Store.TakeCode(ctx, l.ID); ok {
				_, _ = io.WriteString(stdin, code+"\n")
			}
		}
	}
}

// verify confere, com o comando de status do próprio cliente, que o login ficou valendo no volume.
func (r *Runner) verify(ctx context.Context, l Login) bool {
	env, cmdline := statusArgs(l.Provider)
	args := append([]string{"run", "--rm"}, hardening...)
	args = append(args, "-v", volumeName(l.ConnectionID)+":"+AccountMount, "-e", env, "-e", "HOME=/tmp", r.Image)
	args = append(args, cmdline...)
	var out bytes.Buffer
	if err := r.docker(ctx, 60*time.Second, &out, args...); err != nil {
		return false
	}
	if l.Provider == Claude {
		return strings.Contains(strings.ReplaceAll(out.String(), " ", ""), `"loggedIn":true`)
	}
	return true // `codex login status` sai com erro quando não há login
}

// stop derruba o container do login (o cliente docker sozinho não garante parar o container).
func (r *Runner) stop(l Login) {
	c, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	_ = r.docker(c, 15*time.Second, nil, "kill", containerName(l.ID))
}

// stopAndWait derruba o container e espera o processo do cliente terminar: só então o volume deixa de
// estar em uso e pode ser apagado (o `docker volume rm` logo após o kill falha por corrida).
func (r *Runner) stopAndWait(l Login, done <-chan error) {
	r.stop(l)
	select {
	case <-done:
	case <-time.After(15 * time.Second):
	}
}

// cleanup apaga o container e, se pedido, o volume (login que não deu certo não deixa resto).
func (r *Runner) cleanup(ctx context.Context, l Login, volume bool) {
	c, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
	defer cancel()
	_ = r.docker(c, 15*time.Second, nil, "rm", "-f", containerName(l.ID))
	if volume {
		// O Docker pode levar um instante para soltar o volume depois de remover o container.
		for i := 0; i < 5; i++ {
			if err := r.docker(c, 15*time.Second, nil, "volume", "rm", "-f", volumeName(l.ConnectionID)); err == nil {
				break
			}
			time.Sleep(time.Second)
		}
	}
}

func (r *Runner) fail(ctx context.Context, l Login, msg string) {
	r.stop(l)
	r.cleanup(ctx, l, true)
	_ = r.Store.Finish(ctx, l.ID, StateFailed, msg)
	r.Log.Warn("login de conta de assinatura falhou", "provider", string(l.Provider), "login", l.ID)
}

// ensureImage garante a imagem dos clientes: usa a existente ou, em desenvolvimento, constrói.
func (r *Runner) ensureImage(ctx context.Context) error {
	if err := r.docker(ctx, 30*time.Second, nil, "image", "inspect", r.Image); err == nil {
		return nil
	}
	dockerfile := filepath.Join(r.ContextDir, "Dockerfile.cli")
	if r.ContextDir == "" {
		return fmt.Errorf("a imagem %s não existe. Rode `make cli-image` e tente de novo", r.Image)
	}
	if _, err := os.Stat(dockerfile); err != nil {
		return fmt.Errorf("a imagem %s não existe e o worker não tem o Dockerfile. Rode `make cli-image`", r.Image)
	}
	if err := r.docker(ctx, 15*time.Minute, nil, "build", "-q", "-t", r.Image, "-f", dockerfile, r.ContextDir); err != nil {
		return errors.New("não consegui construir a imagem dos clientes. Rode `make cli-image` para ver o erro")
	}
	return nil
}

// docker executa um comando do Docker com prazo. A saída vai para out (se houver); erros não carregam saída.
func (r *Runner) docker(ctx context.Context, timeout time.Duration, out *bytes.Buffer, args ...string) error {
	c, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	cmd := exec.CommandContext(c, r.Bin, args...)
	if out != nil {
		cmd.Stdout = out
	}
	return cmd.Run()
}

// limitedBuf acumula a saída do cliente com teto e é seguro entre goroutines.
type limitedBuf struct {
	mu  sync.Mutex
	b   bytes.Buffer
	max int
}

func (l *limitedBuf) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	n := len(p) // devolver o tamanho original: escrita curta faria o io.Copy do exec falhar
	if l.b.Len()+len(p) > l.max {
		p = p[:max(0, l.max-l.b.Len())]
	}
	l.b.Write(p)
	return n, nil
}

func (l *limitedBuf) String() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.b.String()
}
