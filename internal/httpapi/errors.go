package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
)

// ErrorBody é o formato de erro de todos os endpoints (ver contracts/openapi.yaml):
// código legível por máquina, mensagem, ação sugerida e ID de correlação.
type ErrorBody struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail descreve uma falha.
type ErrorDetail struct {
	Code          string `json:"code"`
	Message       string `json:"message"`
	Action        string `json:"action,omitempty"`
	CorrelationID string `json:"correlation_id"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, r *http.Request, status int, code, message, action string) {
	writeJSON(w, status, ErrorBody{Error: ErrorDetail{
		Code: code, Message: message, Action: action, CorrelationID: correlationID(r.Context()),
	}})
}

// writeDomainError traduz erros de persistência para respostas HTTP sem expor detalhes internos.
func (s *Server) writeDomainError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, persistence.ErrNotFound):
		writeError(w, r, http.StatusNotFound, "not_found", "Recurso não encontrado.", "Verifique o identificador.")
	case errors.Is(err, persistence.ErrVersionConflict):
		writeError(w, r, http.StatusConflict, "version_conflict", "O recurso foi alterado por outra operação.", "Recarregue e tente novamente.")
	case errors.Is(err, persistence.ErrDuplicate):
		writeError(w, r, http.StatusConflict, "duplicate", "Já existe um registro com esses dados.", "Use outro identificador.")
	case errors.Is(err, persistence.ErrInvalidScope), errors.Is(err, persistence.ErrInvalidInput):
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Os dados enviados são inválidos.", "Corrija os parâmetros e tente novamente.")
	default:
		s.log.ErrorContext(r.Context(), "erro interno", "error", err.Error(), "correlation_id", correlationID(r.Context()))
		writeError(w, r, http.StatusInternalServerError, "internal_error", "Erro interno.", "Tente novamente; se persistir, informe o ID de correlação.")
	}
}
