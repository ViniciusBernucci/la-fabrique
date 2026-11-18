package worktrees

import (
	"context"
	"fmt"
	"strings"
)

// ResolveBase devolve a SHA de partida da tentativa na ordem: ref explícita, branch base do projeto,
// HEAD do remoto (origin/HEAD). Cada candidata é lida como branch local e depois como origin/<ref>.
func (m *Manager) ResolveBase(ctx context.Context, explicit, projectBase string) (string, error) {
	var tried []string
	for _, ref := range []string{explicit, projectBase} {
		ref = strings.TrimSpace(ref)
		if ref == "" {
			continue
		}
		tried = append(tried, ref)
		if _, err := git(ctx, m.Repo, nil, "", "check-ref-format", "--branch", ref); err != nil {
			return "", fmt.Errorf("%w: nome de branch inválido: %q", ErrInvalid, ref)
		}
		cands := []string{"refs/heads/" + ref, "refs/remotes/origin/" + ref}
		if shaPat.MatchString(ref) {
			cands = append(cands, ref)
		}
		for _, cand := range cands {
			if sha, err := git(ctx, m.Repo, nil, "", "rev-parse", "--verify", "-q", cand+"^{commit}"); err == nil && shaPat.MatchString(sha) {
				return sha, nil
			}
		}
		if explicit != "" && ref == explicit {
			return "", fmt.Errorf("%w: a referência %q não existe no repositório", ErrInvalid, ref)
		}
	}
	if sha, err := git(ctx, m.Repo, nil, "", "rev-parse", "--verify", "-q", "refs/remotes/origin/HEAD^{commit}"); err == nil && shaPat.MatchString(sha) {
		return sha, nil
	}
	return "", fmt.Errorf("%w: não encontrei a base (tentei %s e origin/HEAD)", ErrInvalid, strings.Join(tried, ", "))
}
