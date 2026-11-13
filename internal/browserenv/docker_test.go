package browserenv

import (
	"context"
	"strings"
	"testing"
	"time"
)

// Usa Docker de verdade (imagem alpine como substituta do Chromium) e é pulado sem runtime.
func TestRealDockerIsolationAndCleanup(t *testing.T) {
	rt := DockerCLI{}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	if _, err := rt.Run(ctx, "info", "--format", "{{.ServerVersion}}"); err != nil {
		t.Skip("Docker indisponível")
	}
	if out, _ := rt.Run(ctx, "images", "-q", "alpine:3.22"); strings.TrimSpace(out) == "" {
		t.Skip("imagem alpine:3.22 ausente")
	}
	mk := func(id string) *Env {
		s := Spec{ProjectID: "p-real", ExecutionID: id, Image: "alpine:3.22", Command: []string{"sleep", "300"}, Allowed: []string{"http://webapp:3000"}, MemoryMB: 64}
		e, err := Provision(ctx, rt, s)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = e.Cleanup(context.Background()) })
		if err := e.Ready(ctx, 10*time.Second); err != nil {
			t.Fatal(err)
		}
		return e
	}
	a, b := mk("real-a"), mk("real-b")
	// Sem saída para a internet (rede interna).
	if _, err := rt.Run(ctx, "exec", a.Container, "wget", "-T", "3", "-qO-", "http://1.1.1.1"); err == nil {
		t.Fatal("a rede da execução deve ser interna: sem acesso à internet")
	}
	// Sem montagens do host.
	if out, _ := rt.Run(ctx, "inspect", "--format", "{{len .Mounts}}", a.Container); strings.TrimSpace(out) != "0" {
		t.Fatalf("sem montagens: %q", out)
	}
	// Uma execução não alcança a outra (redes distintas).
	if _, err := rt.Run(ctx, "exec", a.Container, "ping", "-c", "1", "-W", "2", b.Container); err == nil {
		t.Fatal("execuções devem estar isoladas entre si")
	}
	// Limpeza da A não toca na B; repetir é inofensivo.
	if err := a.Cleanup(ctx); err != nil {
		t.Fatal(err)
	}
	if err := a.Cleanup(ctx); err != nil {
		t.Fatalf("limpeza repetível: %v", err)
	}
	if out, _ := rt.Run(ctx, "ps", "-aq", "--filter", "name="+a.Container); strings.TrimSpace(out) != "" {
		t.Fatal("o container da A deve ter sumido")
	}
	if out, _ := rt.Run(ctx, "network", "ls", "-q", "--filter", "name="+a.Network); strings.TrimSpace(out) != "" {
		t.Fatal("a rede da A deve ter sumido")
	}
	if out, _ := rt.Run(ctx, "inspect", "--format", "{{.State.Running}}", b.Container); strings.TrimSpace(out) != "true" {
		t.Fatal("a execução B deve seguir viva")
	}
	// Queda: recolhimento de órfãos preserva a execução viva.
	got, err := CollectOrphans(ctx, rt, map[string]bool{"real-b": true})
	if err != nil || len(got) != 0 {
		t.Fatalf("%v %v", got, err)
	}
	got, err = CollectOrphans(ctx, rt, map[string]bool{})
	if err != nil || len(got) != 1 || got[0] != "real-b" {
		t.Fatalf("%v %v", got, err)
	}
}
