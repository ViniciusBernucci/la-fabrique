package checkpoints

import (
	"context"
	"fmt"
	"strings"

	"github.com/ViniciusBernucci/le-fabrique/internal/execution"
)

// Restore recria o workspace do checkpoint num ambiente NOVO (workspace vazio) e
// compara a árvore restaurada, hash a hash, com o manifesto. Qualquer diferença
// devolve ErrMismatch (com os primeiros caminhos que divergem).
func (s *Store) Restore(ctx context.Context, cp Checkpoint, ws Workspace, opt Options) error {
	if err := cp.Manifest.Validate(); err != nil {
		return err
	}
	if out, code, err := ws.Run(ctx, []string{"ls", "-A", "/workspace"}); err != nil || code != 0 || strings.TrimSpace(out) != "" {
		if err != nil {
			return fmt.Errorf("conferir o workspace de destino: %w", err)
		}
		return ErrNotEmpty
	}
	load := func(hash string) ([]byte, error) { return s.Object(ctx, cp.ProjectID, hash) }
	if err := ws.Seed(ctx, tarOf(cp.Manifest, load)); err != nil {
		return fmt.Errorf("restaurar o workspace: %w", err)
	}
	return Verify(ctx, cp, ws, opt)
}

// Verify recaptura o workspace e compara com o manifesto do checkpoint.
func Verify(ctx context.Context, cp Checkpoint, ws Workspace, opt Options) error {
	snap, err := Capture(ctx, ws, opt)
	if err != nil {
		return fmt.Errorf("recapturar para comparar: %w", err)
	}
	if snap.Manifest.TreeHash == cp.Manifest.TreeHash {
		return nil
	}
	want := map[string]FileEntry{}
	for _, f := range cp.Manifest.Files {
		want[f.Path] = f
	}
	var diff []string
	got := map[string]bool{}
	for _, f := range snap.Manifest.Files {
		got[f.Path] = true
		w, ok := want[f.Path]
		if !ok || TreeHash([]FileEntry{w}) != TreeHash([]FileEntry{f}) {
			diff = append(diff, f.Path)
		}
	}
	for p := range want {
		if !got[p] {
			diff = append(diff, p)
		}
	}
	if len(diff) > 5 {
		diff = diff[:5]
	}
	return fmt.Errorf("%w: %s", ErrMismatch, strings.Join(diff, ", "))
}

// CaptureAndRelease é a parada segura: confirma que não há processo vivo, captura,
// grava (atomicamente) e SÓ ENTÃO destrói o sandbox. Se qualquer passo falha, o
// sandbox é mantido (nada é liberado sem um checkpoint durável).
func (s *Store) CaptureAndRelease(ctx context.Context, sb *execution.Sandbox, in Input, opt Options) (Checkpoint, error) {
	quiet, err := sb.VerifyQuiescent(ctx)
	if err != nil {
		return Checkpoint{}, err
	}
	if !quiet {
		return Checkpoint{}, ErrNotQuiescent
	}
	snap, err := Capture(ctx, SandboxWorkspace{S: sb}, opt)
	if err != nil {
		return Checkpoint{}, err
	}
	cp, err := s.Save(ctx, snap, in)
	if err != nil {
		return Checkpoint{}, err
	}
	if err := sb.Destroy(ctx); err != nil {
		return cp, fmt.Errorf("checkpoint gravado, mas a liberação do sandbox falhou: %w", err)
	}
	return cp, nil
}
