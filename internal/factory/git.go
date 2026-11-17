package factory

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ViniciusBernucci/le-fabrique/internal/projects"
	"github.com/ViniciusBernucci/le-fabrique/internal/runner"
	"github.com/ViniciusBernucci/le-fabrique/internal/security"
	"github.com/ViniciusBernucci/le-fabrique/internal/workflows"
)

func (e *Executor) git(ctx context.Context, info runner.Info, n workflows.Node) (workflows.ExecResult, error) {
	op, msg := propStr(n, "operation"), propStr(n, "message")
	repo, proj, err := e.ensureRepo(ctx, info.Run)
	if err != nil {
		return fail(info, n, "%v", err), nil
	}
	branch := branchOf(info.Run.ExecutionID)
	emit := func(text string) {
		info.Emit(runner.EventNode, map[string]any{"node": n.ID, "kind": "command", "text": text})
	}
	switch op {
	case "branch":
		emit("branch " + branch)
		return workflows.ExecResult{Outputs: map[string]any{"ok": true, "branch": branch}}, nil
	case "commit":
		sha, committed, err := e.commit(ctx, repo, info.Run.Title, msg)
		if err != nil {
			return fail(info, n, "commit: %v", err), nil
		}
		if committed {
			emit("commit " + sha[:12])
		} else {
			emit("nada a commitar: sem alterações")
		}
		return workflows.ExecResult{Outputs: map[string]any{"ok": true, "branch": branch, "commit": sha, "alteracoes": committed}}, nil
	case "push", "draft_pr":
		if !e.Cfg.AllowPush {
			return fail(info, n, "o envio ao GitHub está desligado: ligue FACTORY_PUSH=true no worker para a fábrica enviar a branch e abrir a Draft PR"), nil
		}
		out := map[string]any{"ok": true, "branch": branch}
		if _, committed, err := e.commit(ctx, repo, info.Run.Title, "Alterações da fábrica: "+info.Run.Title); err != nil {
			return fail(info, n, "commit antes do envio: %v", err), nil
		} else if committed {
			emit("commit das alterações pendentes")
		}
		emit("push " + branch)
		token, err := e.pushBranch(ctx, repo, proj, branch)
		if err != nil {
			return fail(info, n, "push: %v", err), nil
		}
		if op == "draft_pr" {
			title := msg
			if title == "" {
				title = info.Run.Title
			}
			emit("abrindo Draft PR")
			pr, err := e.openDraftPR(ctx, token, proj, branch, title, info.Run.Task)
			if err != nil {
				return fail(info, n, "Draft PR: %v", err), nil
			}
			out["pr_url"], out["pr_numero"] = pr.URL, pr.Number
			emit("Draft PR " + pr.URL)
		}
		return workflows.ExecResult{Outputs: out}, nil
	}
	return fail(info, n, "operação git desconhecida: %q", op), nil
}

// commit grava as alterações da cópia de trabalho; sem alterações não cria commit vazio.
func (e *Executor) commit(ctx context.Context, repo, title, msg string) (sha string, committed bool, err error) {
	if _, err := e.git1(ctx, repo, nil, nil, "add", "-A"); err != nil {
		return "", false, err
	}
	status, err := e.git1(ctx, repo, nil, nil, "status", "--porcelain")
	if err != nil {
		return "", false, err
	}
	if strings.TrimSpace(status) != "" {
		if msg == "" {
			msg = "Alterações da fábrica: " + title
		}
		if _, err := e.git1(ctx, repo, nil, nil, "commit", "--quiet", "-m", msg); err != nil {
			return "", false, err
		}
		committed = true
	}
	sha, err = e.git1(ctx, repo, nil, nil, "rev-parse", "HEAD")
	return strings.TrimSpace(sha), committed, err
}

func (e *Executor) remote(p projects.Project) (string, error) {
	r := p.Repository
	if r.Owner == "" || r.Name == "" {
		return "", errors.New("o projeto não tem dono/nome do repositório no GitHub (Setup > Repositório)")
	}
	if e.Cfg.RemoteURL != nil {
		return e.Cfg.RemoteURL(r.Owner, r.Name), nil
	}
	return "https://github.com/" + r.Owner + "/" + r.Name + ".git", nil
}

func (e *Executor) token(ctx context.Context, p projects.Project) (security.Secret, error) {
	if p.Repository.ConnectionID == nil {
		return security.Secret{}, errors.New("o repositório do projeto não tem uma conexão GitHub")
	}
	if e.Cfg.Vault == nil {
		return security.Secret{}, errors.New("o worker não tem acesso ao cofre de conexões")
	}
	sec, _, err := e.Cfg.Vault.Resolve(ctx, *p.Repository.ConnectionID)
	if err != nil {
		return security.Secret{}, fmt.Errorf("a conexão GitHub do projeto: %w", err)
	}
	return sec, nil
}

// pushBranch envia a branch da execução (nunca a base). O token vai por cabeçalho de ambiente do git, não
// pela linha de comando, e o erro devolvido passa pelo redator.
func (e *Executor) pushBranch(ctx context.Context, repo string, p projects.Project, branch string) (security.Secret, error) {
	remote, err := e.remote(p)
	if err != nil {
		return security.Secret{}, err
	}
	var env []string
	var tok security.Secret
	if e.Cfg.RemoteURL == nil {
		if tok, err = e.token(ctx, p); err != nil {
			return security.Secret{}, err
		}
		basic := base64.StdEncoding.EncodeToString([]byte("x-access-token:" + tok.Reveal()))
		env = []string{"GIT_CONFIG_COUNT=1", "GIT_CONFIG_KEY_0=http.https://github.com/.extraheader", "GIT_CONFIG_VALUE_0=AUTHORIZATION: basic " + basic}
	}
	if _, err := e.git1(ctx, repo, nil, env, "push", "--quiet", remote, "HEAD:refs/heads/"+branch); err != nil {
		return tok, errors.New(redact(err.Error(), tok))
	}
	return tok, nil
}

func redact(s string, tok security.Secret) string {
	if t := tok.Reveal(); t != "" {
		s = strings.ReplaceAll(s, t, "[segredo]")
	}
	return s
}

type pullRequest struct {
	URL    string `json:"html_url"`
	Number int    `json:"number"`
}

func (e *Executor) openDraftPR(ctx context.Context, tok security.Secret, p projects.Project, branch, title, body string) (pullRequest, error) {
	base := e.Cfg.GitHubAPI
	if base == "" {
		base = "https://api.github.com"
	}
	r := p.Repository
	if tok.Reveal() == "" {
		var err error
		if tok, err = e.token(ctx, p); err != nil {
			return pullRequest{}, err
		}
	}
	baseBranch := r.BaseBranch
	if baseBranch == "" {
		baseBranch = "main"
	}
	do := func(method, path string, in any, out any) (int, error) {
		var rd io.Reader
		if in != nil {
			b, _ := json.Marshal(in)
			rd = bytes.NewReader(b)
		}
		req, err := http.NewRequestWithContext(ctx, method, strings.TrimRight(base, "/")+path, rd)
		if err != nil {
			return 0, err
		}
		req.Header.Set("Authorization", "Bearer "+tok.Reveal())
		req.Header.Set("Accept", "application/vnd.github+json")
		req.Header.Set("Content-Type", "application/json")
		cl := &http.Client{Timeout: 30 * time.Second}
		resp, err := cl.Do(req)
		if err != nil {
			return 0, errors.New(redact(err.Error(), tok))
		}
		defer resp.Body.Close()
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		if out != nil && resp.StatusCode < 300 {
			_ = json.Unmarshal(raw, out)
		}
		if resp.StatusCode >= 300 {
			return resp.StatusCode, fmt.Errorf("o GitHub respondeu %d: %s", resp.StatusCode, redact(tail(string(raw), 300), tok))
		}
		return resp.StatusCode, nil
	}
	repoPath := "/repos/" + url.PathEscape(r.Owner) + "/" + url.PathEscape(r.Name)
	var pr pullRequest
	status, err := do("POST", repoPath+"/pulls", map[string]any{"title": title, "head": branch, "base": baseBranch, "body": tail(body, 4000), "draft": true}, &pr)
	if err == nil {
		return pr, nil
	}
	if status != http.StatusUnprocessableEntity {
		return pullRequest{}, err
	}
	// já existe uma PR desta branch: devolve a existente
	var list []pullRequest
	if _, lerr := do("GET", repoPath+"/pulls?state=open&head="+url.QueryEscape(r.Owner+":"+branch), nil, &list); lerr == nil && len(list) > 0 {
		return list[0], nil
	}
	return pullRequest{}, err
}
