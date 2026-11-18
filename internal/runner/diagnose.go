package runner

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
)

// Hint é uma causa provável de falha, reconhecida pelo texto, com o que fazer.
type Hint struct {
	Cause string `json:"cause"`
	Fix   string `json:"fix"`
}

type hintRule struct {
	any  []string // algum destes trechos (minúsculo) no texto dispara a regra
	hint Hint
}

var hintRules = []hintRule{
	{[]string{"não há conta claude", "não há conta codex"}, Hint{"Nenhuma conta de IA ativa para o provedor do agente.", "Conecte a conta em Conexões (botão de login oficial do Claude ou do Codex) e teste de novo."}},
	{[]string{"contas claude ativas", "contas codex ativas"}, Hint{"Há mais de uma conta ativa e o agente não fixa qual usar.", "Escolha a conta no cadastro do agente (campo Conta) ou revogue a que não quer usar."}},
	{[]string{"cannot connect to the docker daemon", "acesso ao docker", "o docker não responde"}, Hint{"O worker não consegue falar com o Docker.", "No desenvolvimento, confirme que o Docker está rodando e que o worker monta /var/run/docker.sock (compose.dev)."}},
	{[]string{"no such image", "pull access denied", "unable to find image", "imagem ausente", "(ausente)"}, Hint{"Uma imagem de sandbox não existe na máquina.", "Para o terminal do agente rode `make agent-image` (e `make egress-image` para o proxy de saída); para scripts e checks aponte FACTORY_TOOL_IMAGE para uma imagem existente."}},
	{[]string{"make: not found", "make: command not found", "sh: make", "código 127"}, Hint{"A imagem dos checks não tem o comando (make ou a toolchain do projeto).", "Defina FACTORY_TOOL_IMAGE com uma imagem que tenha make, git e a toolchain do projeto e reinicie o worker."}},
	{[]string{"not logged in", "please run /login", "invalid api key", "oauth token has expired", "authentication_error", "failed to authenticate", "unauthorized"}, Hint{"O CLI não está autenticado na conta (login ausente ou expirado).", "Refaça o login da conta em Conexões; o volume da conta precisa ter o login oficial."}},
	{[]string{"permission denied"}, Hint{"Permissão negada dentro do terminal ou na pasta de workspaces.", "Se for o volume da conta, refaça o login da conta (o dono do volume é o usuário 10002). Se for a pasta de workspaces, ajuste a permissão de escrita do WORKSPACES_DIR."}},
	{[]string{"rate limit", "usage limit", "quota", "429", "overloaded"}, Hint{"Limite de uso ou cota da assinatura atingido, ou serviço sobrecarregado.", "Aguarde a janela de uso renovar ou troque a conta do agente."}},
	{[]string{"passou do tempo máximo", "tempo máximo excedido", "deadline exceeded"}, Hint{"O passo estourou o tempo máximo.", "Aumente o tempo máximo do agente (ou do nó) ou simplifique a tarefa."}},
	{[]string{"não está no catálogo"}, Hint{"O agente cita uma skill (ou o nó um agente) que não existe no catálogo.", "Importe a skill em Skills, ou corrija o nome no cadastro do agente."}},
	{[]string{"não é um repositório git", "pasta do repositório", "não tem pasta de repositório"}, Hint{"O projeto não tem um repositório preparado para a fábrica.", "Prepare o repositório em Projeto > Setup. (O botão Testar o fluxo usa um repositório descartável quando falta.)"}},
	{[]string{"envio ao github está desligado"}, Hint{"O push e a Draft PR estão desligados no worker.", "Ligue FACTORY_PUSH=true no worker se quiser que a fábrica envie a branch e abra a PR. No modo teste isso é simulado."}},
	{[]string{"conexão github", "não tem uma conexão github"}, Hint{"O repositório do projeto não tem uma conexão GitHub utilizável.", "Ligue uma conexão GitHub ativa em Projeto > Setup > Repositório."}},
	{[]string{"aplicar as alterações do agente", "git apply"}, Hint{"As alterações do agente não aplicam na cópia de trabalho.", "Veja o diagnóstico patch.export (tamanho do patch e o erro do git). Costuma ser arquivo binário grande ou caminho inválido."}},
	{[]string{"limite de tamanho recolhível"}, Hint{"O agente alterou arquivos demais para recolher como patch.", "Reduza o escopo da tarefa ou divida em etapas."}},
	{[]string{"unknown option", "unrecognized option", "unexpected argument", "unknown argument"}, Hint{"O CLI recusou uma opção: a versão instalada pode diferir da verificada.", "Compare a versão do CLI na imagem (claude --version) com a da ADR-0011 e ajuste a imagem."}},
	{[]string{"o cliente terminou com código de saída diferente de zero"}, Hint{"O CLI saiu com erro.", "Leia o stderr no diagnóstico cli.run: ele mostra o motivo real."}},
	{[]string{"fabrica-teste", "test -f", "grep -q"}, Hint{"Os checks do teste não acharam os arquivos da feature fictícia.", "Veja o terminal dos agentes de implementação: o CLI pode não ter criado os arquivos (prompt ignorado, permissão de escrita, erro de autenticação)."}},
	{[]string{"proxy de saída"}, Hint{"O filtro de saída do terminal do agente não está disponível.", "Rode `make egress-image` e reinicie o worker (ou FACTORY_EGRESS_FILTER=false para liberar a internet, o que não recomendo)."}},
	{[]string{"em quarentena"}, Hint{"O sandbox ficou em estado incerto.", "Cancele a execução e teste de novo; verifique processos presos com `docker ps` (rótulo lf.managed)."}},
}

// HintsFor devolve as causas prováveis reconhecidas no texto (sem repetir).
func HintsFor(text string) []Hint {
	low := strings.ToLower(text)
	var out []Hint
	seen := map[string]bool{}
	for _, r := range hintRules {
		for _, k := range r.any {
			if strings.Contains(low, k) && !seen[r.hint.Cause] {
				seen[r.hint.Cause] = true
				out = append(out, r.hint)
				break
			}
		}
	}
	return out
}

func payload(e persistence.Event) map[string]any {
	m := map[string]any{}
	_ = json.Unmarshal(e.Payload, &m)
	return m
}

func str(m map[string]any, k string) string {
	switch v := m[k].(type) {
	case string:
		return v
	case nil:
		return ""
	default:
		b, _ := json.Marshal(v)
		return string(b)
	}
}

func clip(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// Failure localiza a falha de uma execução: o último erro de nó (ou o motivo da execução).
func Failure(run Run, events []persistence.Event) (node, reason string) {
	for _, e := range events {
		p := payload(e)
		if e.Type == EventNode && str(p, "kind") == "error" {
			node, reason = str(p, "node"), str(p, "text")
		}
	}
	if reason == "" {
		reason = run.Failure
	}
	return node, reason
}

// Report monta o relatório de diagnóstico em Markdown: pronto para colar numa conversa de suporte. Reúne o
// veredito, onde e por que falhou, as causas prováveis, o pré-voo do ambiente, a linha do tempo com durações
// e o detalhe técnico por nó. Nunca inclui segredos (os eventos já chegam redigidos).
func Report(run Run, events []persistence.Event) string {
	var b strings.Builder
	w := func(format string, a ...any) { fmt.Fprintf(&b, format+"\n", a...) }
	dur := run.UpdatedAt.Sub(run.CreatedAt).Round(time.Second)
	w("# Relatório de diagnóstico da execução")
	w("")
	w("- Execução: `%s` (projeto `%s`, tipo `%s`)", run.ExecutionID, run.ProjectID, run.Kind)
	w("- Título: %s", run.Title)
	w("- Estado: **%s** · resultado: `%s` · duração: %s", run.Status, run.Outcome, dur)
	w("- Criada em %s · atualizada em %s", run.CreatedAt.Format(time.RFC3339), run.UpdatedAt.Format(time.RFC3339))
	if run.Failure != "" {
		w("- Falha registrada: %s", run.Failure)
	}

	node, reason := Failure(run, events)
	bad := run.Status == "failed" || (run.Status == "completed" && run.Outcome == "failure") || node != ""
	w("")
	w("## Onde falhou")
	if !bad {
		w("Nenhuma falha registrada.")
	} else {
		w("- Nó: `%s`", orDash(node))
		w("- Motivo: %s", orDash(reason))
	}

	var all strings.Builder
	for _, e := range events {
		all.Write(e.Payload)
		all.WriteByte('\n')
	}
	hints := HintsFor(reason + "\n" + all.String())
	if bad && len(hints) > 0 {
		w("")
		w("## Causas prováveis e o que fazer")
		for _, h := range hints {
			w("- **%s** %s", h.Cause, h.Fix)
		}
	}

	w("")
	w("## Pré-voo do ambiente")
	pre := 0
	for _, e := range events {
		if e.Type != EventDiag {
			continue
		}
		p := payload(e)
		if str(p, "step") != "preflight" {
			continue
		}
		pre++
		mark := "✔"
		if str(p, "ok") == "false" {
			mark = "✖"
		}
		line := fmt.Sprintf("- %s %s — %s", mark, str(p, "check"), str(p, "detail"))
		if d := str(p, "dica"); d != "" {
			line += " (dica: " + d + ")"
		}
		w("%s", line)
	}
	if pre == 0 {
		w("(o pré-voo não rodou nesta execução)")
	}

	w("")
	w("## Linha do tempo (passos do fluxo)")
	for _, e := range events {
		p := payload(e)
		switch e.Type {
		case EventStarted:
			w("- #%d início — %s", e.Seq, str(p, "title"))
		case EventStep:
			port := ""
			if str(p, "port") != "" {
				port = " → " + str(p, "port")
			}
			note := ""
			if str(p, "note") != "" {
				note = " (" + str(p, "note") + ")"
			}
			w("- #%d `%s` [%s]%s%s · %s ms", e.Seq, str(p, "node"), str(p, "type"), port, note, str(p, "ms"))
		case EventDecision:
			w("- #%d decisão em `%s`: %s", e.Seq, str(p, "node"), str(p, "choice"))
		case EventFinished:
			w("- #%d fim: %s / %s %s", e.Seq, str(p, "status"), str(p, "outcome"), str(p, "failure"))
		case EventCancel:
			w("- #%d cancelada", e.Seq)
		}
	}

	w("")
	w("## Detalhe técnico por nó")
	nodes := []string{}
	byNode := map[string][]persistence.Event{}
	for _, e := range events {
		if e.Type != EventDiag && e.Type != EventNode {
			continue
		}
		n := str(payload(e), "node")
		if n == "" {
			continue
		}
		if _, ok := byNode[n]; !ok {
			nodes = append(nodes, n)
		}
		byNode[n] = append(byNode[n], e)
	}
	for _, n := range nodes {
		w("")
		w("### `%s`", n)
		for _, e := range byNode[n] {
			p := payload(e)
			if e.Type == EventNode {
				switch str(p, "kind") {
				case "error":
					w("- ✖ %s", clip(str(p, "text"), 600))
				case "prompt":
					w("- orientações enviadas (%d caracteres): %s", len(str(p, "text")), clip(strings.ReplaceAll(str(p, "text"), "\n", " ⏎ "), 500))
				case "agent":
					w("- agente: %s · %s · modelo %s · conta %s · skills %s", str(p, "agent"), str(p, "provider"), orDash(str(p, "model")), str(p, "account"), str(p, "skills"))
				}
				continue
			}
			keys := make([]string, 0, len(p))
			for k := range p {
				if k != "node" && k != "step" && k != "at" {
					keys = append(keys, k)
				}
			}
			sort.Strings(keys)
			parts := make([]string, 0, len(keys))
			for _, k := range keys {
				v := clip(str(p, k), 400)
				if v != "" {
					parts = append(parts, k+"="+v)
				}
			}
			w("- `%s` %s", str(p, "step"), strings.Join(parts, " · "))
		}
	}
	w("")
	w("---")
	w("Gerado a partir dos eventos da execução (`execution_events`). O arquivo `run.log` da execução, em `<workspaces>/.factory/%s/run.log`, tem as mesmas linhas de diagnóstico.", run.ExecutionID)
	return b.String()
}

func orDash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "—"
	}
	return s
}
