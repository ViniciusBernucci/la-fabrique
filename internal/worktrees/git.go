// Package worktrees provisiona a worktree e a branch temporária de cada tentativa que escreve código
// (AJ06-01): diretórios e branches distintos por trilha, SHA de partida fixa, escopo de caminhos,
// checkpoints recuperáveis (inclui o que não foi versionado) e limpeza por tentativa que preserva tudo.
package worktrees

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// git roda o git no diretório dir com ambiente fechado (sem helpers de credencial nem terminal).
func git(ctx context.Context, dir string, env []string, stdin string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	cmd.Env = append([]string{"PATH=" + os.Getenv("PATH"), "HOME=" + os.TempDir(), "GIT_TERMINAL_PROMPT=0", "GIT_CONFIG_NOSYSTEM=1",
		"GIT_AUTHOR_NAME=la-fabrique", "GIT_AUTHOR_EMAIL=fabrique@localhost", "GIT_COMMITTER_NAME=la-fabrique", "GIT_COMMITTER_EMAIL=fabrique@localhost"}, env...)
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(errb.String())
		if len(msg) > 300 {
			msg = msg[:300]
		}
		return "", fmt.Errorf("git %s: %v: %s", args[0], err, msg)
	}
	return strings.TrimSpace(out.String()), nil
}
