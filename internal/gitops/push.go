package gitops

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/ViniciusBernucci/le-fabrique/internal/effects"
)

// Effects é a interface de ESCRITA: só publica a branch da feature, pelo ledger.
type Effects interface {
	Push(ctx context.Context, req PushRequest) (effects.Result, error)
}

// RemoteProof é a prova de que a ref existe no remoto com o SHA esperado.
type RemoteProof struct {
	Ref  string `json:"ref"`
	SHA  string `json:"sha"`
	Note string `json:"note,omitempty"`
}

// PushRequest pede a publicação do commit na branch da feature.
type PushRequest struct {
	ProjectID string
	FeatureID string
	Attempt   int
	Branch    string // precisa ser exatamente FeatureBranch(FeatureID, Attempt)
	SHA       string // commit a publicar
	// AllowWorkflows autoriza uma mudança em .github/ (decisão humana prévia).
	AllowWorkflows bool
}

// Guard recusa, ANTES de qualquer comando git, tudo o que não seja a publicação
// de um commit na branch da feature: branch diferente da esperada, base, SHA
// malformado. Não há force, exclusão, refspec livre nem tags.
func (r PushRequest) Guard(base string) error {
	want, err := FeatureBranch(r.FeatureID, r.Attempt)
	if err != nil {
		return err
	}
	switch {
	case r.Branch != want, r.Branch == base, ValidFeatureBranch(r.Branch) != nil:
		return fmt.Errorf("%w: %q (esperada %q)", ErrBranchForbidden, r.Branch, want)
	case !shaPattern.MatchString(r.SHA):
		return fmt.Errorf("%w: SHA", ErrBadCommit)
	case r.ProjectID == "":
		return fmt.Errorf("%w: projeto", ErrBadCommit)
	}
	return nil
}

// BusinessKey é a chave estável do efeito: o mesmo commit na mesma branch da mesma
// tentativa é um único efeito, não importa quantas vezes se tente.
func (r PushRequest) BusinessKey() string {
	return fmt.Sprintf("%s:%s:%d:push:%s:%s", r.ProjectID, r.FeatureID, r.Attempt, r.Branch, r.SHA[:16])
}

// Publisher publica pelo ledger de efeitos.
type Publisher struct {
	Work   *Workdir
	Ledger *effects.Ledger
}

// Push registra a intenção, envia e confirma por ref; sob timeout, a prova decide.
func (p *Publisher) Push(ctx context.Context, req PushRequest) (effects.Result, error) {
	if err := req.Guard(p.Work.Repo.BaseBranch); err != nil {
		return effects.Result{}, err
	}
	if err := p.checkSensitive(ctx, req); err != nil {
		return effects.Result{}, err
	}
	body := map[string]any{"repo": p.Work.Repo.Owner + "/" + p.Work.Repo.Name, "branch": req.Branch, "sha": req.SHA, "feature": req.FeatureID, "attempt": req.Attempt}
	return p.Ledger.Run(ctx, effects.Spec{ProjectID: req.ProjectID, BusinessKey: req.BusinessKey(), Operation: "push", Request: body}, &pushAdapter{w: p.Work, req: req})
}

// checkSensitive recusa publicar mudança em .github/ sem aprovação explícita.
func (p *Publisher) checkSensitive(ctx context.Context, req PushRequest) error {
	if req.AllowWorkflows {
		return nil
	}
	base, err := p.Work.BaseTip(ctx)
	if err != nil {
		return err
	}
	res, err := p.Work.git(ctx, nil, "diff", "--name-only", base, req.SHA)
	if err != nil {
		return err
	}
	for _, f := range strings.Split(strings.TrimSpace(res.Stdout), "\n") {
		if strings.HasPrefix(f, ".github/") {
			return fmt.Errorf("%w: %s", ErrWorkflowChange, f)
		}
	}
	return nil
}

type pushAdapter struct {
	w   *Workdir
	req PushRequest
}

func (a *pushAdapter) refspec() string { return a.req.SHA + ":refs/heads/" + a.req.Branch }

// Apply envia o commit. Recusa do remoto (rejeitado, sem permissão) é "sabidamente
// sem efeito"; timeout ou queda de rede deixam o resultado desconhecido (o ledger verifica).
func (a *pushAdapter) Apply(ctx context.Context, op effects.Op) (any, error) {
	if err := a.req.Guard(a.w.Repo.BaseBranch); err != nil {
		return nil, fmt.Errorf("%w: %v", effects.ErrNotApplied, err)
	}
	_, err := a.w.git(ctx, nil, "push", "--porcelain", "--no-follow-tags", "--no-verify", "--", a.w.Repo.URL(), a.refspec())
	if err != nil {
		if errors.Is(err, ErrRejected) || errors.Is(err, ErrAuth) || errors.Is(err, ErrNotFound) {
			return nil, fmt.Errorf("%w: %v", effects.ErrNotApplied, err)
		}
		return nil, err
	}
	v, verr := a.Verify(ctx, op)
	if verr != nil || v.Status != effects.VerdictConfirmed {
		return nil, fmt.Errorf("%w: o push terminou sem prova da ref", ErrGit)
	}
	return v.Proof, nil
}

// Verify procura a prova pela ref no remoto: ausente → repetir é seguro; no SHA
// (ou adiante dele) → confirmado; atrás → ausente; divergente ou ilegível → inconclusivo.
func (a *pushAdapter) Verify(ctx context.Context, _ effects.Op) (effects.Verdict, error) {
	ref := "refs/heads/" + a.req.Branch
	conn := &GitConnector{Repo: a.w.Repo, Git: a.w.Git}
	remote, exists, err := conn.RemoteRef(ctx, a.req.Branch)
	if err != nil {
		return effects.Verdict{Status: effects.VerdictInconclusive, Evidence: "Não foi possível consultar a ref " + ref + " no remoto: " + firstLine(redact(&a.w.Repo, err.Error()))}, nil
	}
	if !exists {
		return effects.Verdict{Status: effects.VerdictAbsent, Proof: RemoteProof{Ref: ref}}, nil
	}
	if remote == a.req.SHA {
		return effects.Verdict{Status: effects.VerdictConfirmed, Proof: RemoteProof{Ref: ref, SHA: remote}}, nil
	}
	// A ref existe em outro SHA: traz o objeto e compara a ancestralidade.
	local := "refs/lf/remote/" + strings.TrimPrefix(a.req.Branch, BranchPrefix)
	if _, err := a.w.git(ctx, nil, "fetch", "--no-tags", "--no-write-fetch-head", "--", a.w.Repo.URL(), "+"+ref+":"+local); err != nil {
		return effects.Verdict{Status: effects.VerdictInconclusive, Evidence: fmt.Sprintf("A ref %s está em %s (esperado %s) e não foi possível comparar o histórico.", ref, short(remote), short(a.req.SHA))}, nil
	}
	if ok, _ := a.isAncestor(ctx, a.req.SHA, local); ok {
		return effects.Verdict{Status: effects.VerdictConfirmed, Proof: RemoteProof{Ref: ref, SHA: remote, Note: "a ref avançou depois do commit esperado"}}, nil
	}
	if ok, _ := a.isAncestor(ctx, local, a.req.SHA); ok {
		return effects.Verdict{Status: effects.VerdictAbsent, Proof: RemoteProof{Ref: ref, SHA: remote, Note: "o remoto está atrás: o push avança a ref"}}, nil
	}
	return effects.Verdict{Status: effects.VerdictInconclusive, Evidence: fmt.Sprintf("A ref %s divergiu: remoto %s, esperado %s. Alguém alterou a branch da feature fora da fábrica.", ref, short(remote), short(a.req.SHA))}, nil
}

func (a *pushAdapter) isAncestor(ctx context.Context, anc, desc string) (bool, error) {
	_, err := a.w.git(ctx, nil, "merge-base", "--is-ancestor", anc, desc)
	return err == nil, err
}

func short(s string) string {
	if len(s) > 12 {
		return s[:12]
	}
	return s
}

// ProofOf reconstrói a prova gravada no ledger.
func ProofOf(r effects.Result) (RemoteProof, bool) {
	if r.Entry.Proof == nil {
		return RemoteProof{}, false
	}
	b, err := json.Marshal(r.Entry.Proof)
	if err != nil {
		return RemoteProof{}, false
	}
	var p RemoteProof
	return p, json.Unmarshal(b, &p) == nil && p.SHA != ""
}
