package opslog

import (
	"context"
	"errors"
	"testing"

	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
)

// Emit recusa nível/origem fora do enum antes de tocar o banco (não precisa de conexão real para
// este caso — a cobertura do caminho de sucesso, com inserção de verdade, está em
// tests/integration/operations_test.go).
func TestEmitValidatesLevelAndSourceBeforeTouchingTheDatabase(t *testing.T) {
	ctx := context.Background()
	if err := Emit(ctx, nil, "critico", SourceFabrica, "m", nil); !errors.Is(err, persistence.ErrInvalidInput) {
		t.Fatalf("nível inválido: %v", err)
	}
	if err := Emit(ctx, nil, LevelInfo, "desconhecida", "m", nil); !errors.Is(err, persistence.ErrInvalidInput) {
		t.Fatalf("origem inválida: %v", err)
	}
}
