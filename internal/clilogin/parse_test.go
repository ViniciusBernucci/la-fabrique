package clilogin

import "testing"

// Saída real do `codex login --device-auth` (v0.164.0), com as cores do terminal.
const codexOut = "\nWelcome to Codex [v\x1b[90m0.164.0\x1b[0m]\n\x1b[90mOpenAI's command-line coding agent\x1b[0m\n\n" +
	"Follow these steps to sign in with ChatGPT using device code authorization:\n\n" +
	"1. Open this link in your browser and sign in to your account\n   \x1b[94mhttps://auth.openai.com/codex/device\x1b[0m\n\n" +
	"2. Enter this one-time code \x1b[90m(expires in 15 minutes)\x1b[0m\n   \x1b[94mA1B2-C3D4E\x1b[0m\n"

// Saída real do `claude auth login --claudeai` (2.1.325) sem navegador.
const claudeOut = "Opening browser to sign in…\nIf the browser didn't open, visit: " +
	"https://claude.com/cai/oauth/authorize?code=true&client_id=x&response_type=code&state=abc\nPaste code here if prompted > "

func TestExtractCodexNeedsLinkAndCode(t *testing.T) {
	c, ok := Extract(Codex, codexOut)
	if !ok || c.URL != "https://auth.openai.com/codex/device" || c.Code != "A1B2-C3D4E" || c.NeedsCode {
		t.Fatalf("%+v %v", c, ok)
	}
	// só o link ainda: o código não chegou, então ainda não há o que mostrar
	if _, ok := Extract(Codex, "1. Open\n   https://auth.openai.com/codex/device\n"); ok {
		t.Fatal("sem código não deve estar pronto")
	}
	// a versão do cliente não pode ser tomada como código
	if c, ok := Extract(Codex, "Codex [v0.164.0]\nhttps://auth.openai.com/codex/device\n"); ok {
		t.Fatalf("versão virou código: %+v", c)
	}
}

func TestExtractClaudeUsesLinkAndAsksForPastedCode(t *testing.T) {
	c, ok := Extract(Claude, claudeOut)
	if !ok || !c.NeedsCode || c.Code != "" || c.URL[:34] != "https://claude.com/cai/oauth/autho" {
		t.Fatalf("%+v %v", c, ok)
	}
}

func TestExtractRefusesLinksOutsideTheAllowlist(t *testing.T) {
	bad := []string{
		"visit: https://evil.example/cai/oauth/authorize?x=1\nPaste code here",
		"visit: http://claude.com/cai/oauth/authorize\n",            // sem https
		"visit: https://claude.com.evil.example/authorize\n",        // host parecido
		"visit: https://user:pw@claude.com/cai/oauth/authorize\n",   // credenciais na URL
		"visit: https://auth.openai.com/codex/device\nA1B2-C3D4E\n", // domínio de outro provedor
	}
	for _, out := range bad {
		if c, ok := Extract(Claude, out); ok {
			t.Errorf("deveria recusar: %q -> %+v", out, c)
		}
	}
}

func TestLastLineDropsLinksAndCodes(t *testing.T) {
	got := LastLine("x\nErro em https://claude.com/cai/oauth/authorize?state=segredo código A1B2-C3D4E\n\n")
	if got != "Erro em <endereço> código <código>" {
		t.Fatalf("%q", got)
	}
}
