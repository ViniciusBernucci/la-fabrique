package providers

import (
	"bufio"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Verificação contra os clientes REAIS (AJ07-01). Usa a conta já logada da máquina e faz chamadas mínimas
// (consomem uma fração da cota), por isso só roda com LF_VERIFY_CLIS=1:
//
//	LF_VERIFY_CLIS=1 go test ./internal/providers -run TestLiveClients -v
//
// Os comandos vêm de BuildCommand, então o que se prova é o que a fábrica de fato lança. Versões verificadas:
// claude 2.1.323 e codex-cli 0.164.0.

const liveAcct = "0f0f0f0f-0f0f-4f0f-8f0f-0f0f0f0f0f0f"

func liveOrSkip(t *testing.T, bin string) {
	t.Helper()
	if os.Getenv("LF_VERIFY_CLIS") != "1" {
		t.Skip("defina LF_VERIFY_CLIS=1 para verificar contra o cliente real")
	}
	if _, err := exec.LookPath(bin); err != nil {
		t.Skip(bin + " não instalado")
	}
}

// hostRun executa o Command na máquina (conta logada do usuário) e devolve as linhas JSON de stdout.
func hostRun(t *testing.T, c Command, dir string) []map[string]any {
	t.Helper()
	cmd := exec.Command(c.Args[0], c.Args[1:]...)
	cmd.Dir = dir
	cmd.Stdin = strings.NewReader(c.Stdin)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%v: %v\n%s", c.Args, err, out)
	}
	return jsonLines(string(out))
}

func jsonLines(s string) []map[string]any {
	var out []map[string]any
	for _, l := range strings.Split(s, "\n") {
		var m map[string]any
		if json.Unmarshal([]byte(strings.TrimSpace(l)), &m) == nil {
			out = append(out, m)
		}
	}
	return out
}

func find(ls []map[string]any, typ, sub string) map[string]any {
	for _, m := range ls {
		if m["type"] == typ && (sub == "" || m["subtype"] == sub) {
			return m
		}
	}
	return nil
}

func text(ls []map[string]any) string {
	if r := find(ls, "result", ""); r != nil {
		s, _ := r["result"].(string)
		return s
	}
	for _, m := range ls {
		if it, ok := m["item"].(map[string]any); ok && it["type"] == "agent_message" {
			s, _ := it["text"].(string)
			return s
		}
	}
	return ""
}

func TestLiveClaudeSessionResumeAndStreamInput(t *testing.T) {
	liveOrSkip(t, "claude")
	dir := t.TempDir()
	acct := Account{ID: liveAcct, Name: "live", CLI: Claude}
	sid := "d1d1d1d1-0000-4000-8000-" + strings.Repeat("a", 12)
	base := Request{Account: acct, Mode: ModeReadOnly, Model: "haiku", Dir: dir}

	// 1) O id fixo é respeitado e informado no init.
	first := base
	first.Prompt, first.Persist, first.NewSessionID = "memorize a palavra ABACAXI e responda apenas OK", true, sid
	c, err := BuildCommand(first)
	if err != nil {
		t.Fatal(err)
	}
	ls := hostRun(t, c, dir)
	if init := find(ls, "system", "init"); init == nil || init["session_id"] != sid {
		t.Fatalf("o init deve trazer o session_id fixado: %v", init)
	}
	// 2) A retomada mantém o id e o contexto.
	second := base
	second.Prompt, second.ResumeSession = "qual palavra pedi para memorizar? responda só a palavra", sid
	c, _ = BuildCommand(second)
	ls = hostRun(t, c, dir)
	if init := find(ls, "system", "init"); init == nil || init["session_id"] != sid || !strings.Contains(strings.ToUpper(text(ls)), "ABACAXI") {
		t.Fatalf("retomada: %v %q", init, text(ls))
	}
	// 3) Sessão efêmera não pode ser retomada.
	eph := base
	eph.Prompt, eph.NewSessionID = "diga X", "d2d2d2d2-0000-4000-8000-"+strings.Repeat("b", 12)
	c, _ = BuildCommand(eph)
	hostRun(t, c, dir)
	res := base
	res.Prompt, res.ResumeSession = "oi", eph.NewSessionID
	c, _ = BuildCommand(res)
	if out, _ := exec.Command(c.Args[0], c.Args[1:]...).CombinedOutput(); !strings.Contains(string(out), "No conversation found") {
		t.Fatalf("sessão efêmera não deve ser retomável: %s", out)
	}
	// 4) Entrada em fluxo: a orientação entra no meio da execução e o cliente a reconhece (replay).
	inter := base
	inter.Prompt, inter.Interactive = "diga apenas A", true
	c, err = BuildCommand(inter)
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(c.Args[0], c.Args[1:]...)
	cmd.Dir = dir
	in, _ := cmd.StdinPipe()
	out, _ := cmd.StdoutPipe()
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	_, _ = io.WriteString(in, c.Stdin+"\n")
	sc := bufio.NewScanner(out)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	var acks, results int
	var results2 []string
	sent := false
	deadline := time.Now().Add(2 * time.Minute)
	for sc.Scan() && time.Now().Before(deadline) {
		var m map[string]any
		if json.Unmarshal(sc.Bytes(), &m) != nil {
			continue
		}
		if m["type"] == "user" && m["isReplay"] == true {
			acks++
		}
		if m["type"] == "result" {
			results++
			s, _ := m["result"].(string)
			results2 = append(results2, s)
			if !sent {
				sent = true
				_, _ = io.WriteString(in, UserMessage("agora diga apenas B")+"\n")
			} else {
				_ = in.Close()
			}
		}
	}
	_ = cmd.Wait()
	if acks != 2 || results != 2 || !strings.Contains(results2[1], "B") {
		t.Fatalf("a orientação no meio da execução deve ser reconhecida e respondida: acks=%d results=%v", acks, results2)
	}
}

func TestLiveCodexThreadResumeAndSandboxOnResume(t *testing.T) {
	liveOrSkip(t, "codex")
	dir := t.TempDir()
	acct := Account{ID: liveAcct, Name: "live", CLI: Codex}
	base := Request{Account: acct, Mode: ModeReadOnly, Dir: dir}

	first := base
	first.Prompt, first.Persist = "memorize a palavra ABACAXI e responda apenas OK", true
	c, err := BuildCommand(first)
	if err != nil {
		t.Fatal(err)
	}
	ls := hostRun(t, c, dir)
	ts := find(ls, "thread.started", "")
	id, _ := ts["thread_id"].(string)
	if id == "" {
		t.Fatalf("o Codex deve informar o thread_id: %v", ls)
	}
	second := base
	second.Prompt, second.ResumeSession = "qual palavra pedi para memorizar? responda só a palavra", id
	c, err = BuildCommand(second)
	if err != nil {
		t.Fatal(err)
	}
	ls = hostRun(t, c, dir)
	if ts := find(ls, "thread.started", ""); ts == nil || ts["thread_id"] != id || !strings.Contains(strings.ToUpper(text(ls)), "ABACAXI") {
		t.Fatalf("retomada: %v %q", ts, text(ls))
	}
	// O modo somente leitura vale na retomada (via -c sandbox_mode): nenhum arquivo é criado.
	third := base
	third.Prompt, third.ResumeSession = "crie o arquivo proibido.txt com o texto x usando o shell", id
	c, _ = BuildCommand(third)
	hostRun(t, c, dir)
	if _, err := os.Stat(filepath.Join(dir, "proibido.txt")); err == nil {
		t.Fatal("o sandbox somente leitura deve valer na retomada")
	}
}
