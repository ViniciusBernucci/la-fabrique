package worktrees

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
)

var inProcess sync.Map // repositório -> *sync.Mutex

// withLock serializa as mudanças de refs e worktrees do repositório: um escritor por projeto, dentro
// do processo (mutex) e entre processos (flock no diretório git comum, que api e worker compartilham
// pelo volume). Respeita o cancelamento do contexto.
func (m *Manager) withLock(ctx context.Context, fn func() error) error {
	key := filepath.Clean(m.Repo)
	mu, _ := inProcess.LoadOrStore(key, &sync.Mutex{})
	l := mu.(*sync.Mutex)
	if err := lockCtx(ctx, l.TryLock); err != nil {
		return err
	}
	defer l.Unlock()

	common, err := git(ctx, m.Repo, nil, "", "rev-parse", "--git-common-dir")
	if err != nil {
		return err
	}
	if !filepath.IsAbs(common) {
		common = filepath.Join(m.Repo, common)
	}
	f, err := os.OpenFile(filepath.Join(common, "lf-writer.lock"), os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return fmt.Errorf("trava do repositório: %w", err)
	}
	defer f.Close()
	if err := lockCtx(ctx, func() bool { return syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB) == nil }); err != nil {
		return err
	}
	defer func() { _ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN) }()
	return fn()
}

// lockCtx tenta adquirir até o contexto acabar (máx. 2 minutos).
func lockCtx(ctx context.Context, try func() bool) error {
	deadline := time.NewTimer(2 * time.Minute)
	defer deadline.Stop()
	for {
		if try() {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-deadline.C:
			return fmt.Errorf("outro escritor está usando este repositório: tente novamente")
		case <-time.After(25 * time.Millisecond):
		}
	}
}

// Escapes lista os links simbólicos da worktree que apontam para fora dela (o agente nunca deve
// ler nem escrever além da própria worktree). Ignora o diretório .git.
func (m *Manager) Escapes(ctx context.Context, id string) ([]string, error) {
	a, err := m.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	root, err := filepath.EvalSymlinks(a.Path)
	if err != nil {
		return nil, err
	}
	var bad []string
	err = filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() && d.Name() == ".git" {
			return filepath.SkipDir
		}
		if d.Type()&os.ModeSymlink == 0 {
			return nil
		}
		target, e := filepath.EvalSymlinks(p)
		rel, _ := filepath.Rel(root, p)
		if e != nil || (target != root && !strings.HasPrefix(target, root+string(filepath.Separator))) {
			bad = append(bad, rel)
		}
		return nil
	})
	return bad, err
}
