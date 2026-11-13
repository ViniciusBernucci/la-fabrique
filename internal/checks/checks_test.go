package checks

import "testing"

func TestSignatureIgnoresNoiseButNotTheFailure(t *testing.T) {
	a := "\x1b[31mFAIL\x1b[0m TestX (0.12s)\n    x_test.go:42: expected 5, got 7\npanic: boom at 0xc000123456\nok other 0.5s\n/tmp/build-123/abc.go:9: error: undefined: foo"
	b := "FAIL TestX (3.40s)\n    x_test.go:77: expected 9, got 8\npanic: boom at 0xdeadbeef\nok other 9s\n/var/folders/zz/abc.go:1: error: undefined: foo"
	if Signature(1, a) != Signature(1, b) {
		t.Fatal("tempos, números, endereços e caminhos temporários não mudam a assinatura")
	}
	if Signature(1, a) == Signature(2, a) {
		t.Fatal("o código de saída faz parte da assinatura")
	}
	c := "FAIL TestX\n    x_test.go:42: connection refused"
	if Signature(1, a) == Signature(1, c) {
		t.Fatal("outra falha, outra assinatura")
	}
	if Signature(1, "algo quebrou por motivo A") == Signature(1, "algo quebrou por motivo B") {
		t.Fatal("sem palavra-chave, a cauda da saída ainda diferencia as falhas")
	}
}
