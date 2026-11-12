package httpapi

import (
	"encoding/json"
	"errors"
	"mime"
	"net/http"
	"strconv"

	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
	"github.com/ViniciusBernucci/le-fabrique/internal/projects"
	"github.com/ViniciusBernucci/le-fabrique/internal/security"
	"github.com/ViniciusBernucci/le-fabrique/internal/workflows"
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
	var locked *security.LockedError
	var field *projects.FieldError
	var invalid *projects.ValidationError
	var wfField *workflows.FieldError
	switch {
	case errors.As(err, &field):
		writeError(w, r, http.StatusBadRequest, "invalid_request", field.Message, "Corrija o campo \""+field.Field+"\" e tente novamente.")
	case errors.As(err, &wfField):
		writeError(w, r, http.StatusBadRequest, "invalid_request", wfField.Message, "Corrija o campo \""+wfField.Field+"\" e tente novamente.")
	case errors.As(err, &invalid):
		writeError(w, r, http.StatusUnprocessableEntity, "validation_failed", "O projeto tem pendências e não pode ser ativado.", "Resolva os itens pendentes em GET /api/projects/{id}/validation.")
	case errors.Is(err, projects.ErrSourceIncomplete):
		writeError(w, r, http.StatusUnprocessableEntity, "source_incomplete", "Faltam plataforma, conexão compatível ou lista na origem.", "Complete a etapa Origem e salve antes de verificar.")
	case errors.Is(err, projects.ErrActiveLocked):
		writeError(w, r, http.StatusConflict, "project_active", "O projeto está ativo: repositório e origem não podem mudar agora.", "Suspenda o projeto, altere a configuração e ative novamente.")
	case errors.Is(err, projects.ErrInvalidState):
		writeError(w, r, http.StatusConflict, "invalid_state", "O estado atual do projeto não permite esta operação.", "Recarregue o projeto e confira o estado.")
	case errors.As(err, &locked):
		secs := int(locked.RetryAfter.Seconds()) + 1
		w.Header().Set("Retry-After", strconv.Itoa(secs))
		writeError(w, r, http.StatusTooManyRequests, "too_many_attempts", "Muitas tentativas.", "Aguarde alguns minutos antes de tentar novamente.")
	case errors.Is(err, security.ErrInvalidCredentials):
		writeError(w, r, http.StatusUnauthorized, "invalid_credentials", "Usuário, senha ou código inválidos.", "Confira os dados e tente novamente.")
	case errors.Is(err, security.ErrInvalidCode):
		writeError(w, r, http.StatusUnauthorized, "invalid_code", "Código inválido ou já utilizado.", "Use o código atual do app autenticador ou um código de recuperação não usado.")
	case errors.Is(err, security.ErrChallenge):
		writeError(w, r, http.StatusUnauthorized, "challenge_expired", "A verificação expirou ou foi esgotada.", "Entre novamente com usuário e senha.")
	case errors.Is(err, security.ErrTOTPState):
		writeError(w, r, http.StatusConflict, "totp_state", "Operação incompatível com o estado do segundo fator.", "Recomece o login ou a substituição do 2FA.")
	case errors.Is(err, security.ErrWeakPassword), errors.Is(err, security.ErrInvalidUsername), errors.Is(err, security.ErrInvalidSecret):
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Os dados enviados são inválidos.", "Corrija os dados e tente novamente.")
	case errors.Is(err, security.ErrConnectionRevoked):
		writeError(w, r, http.StatusConflict, "connection_revoked", "A conexão está revogada.", "Crie uma nova conexão para voltar a usar este destino.")
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

// decodeJSON lê o corpo como JSON estrito (tipo, campos desconhecidos). Em caso
// de erro já responde e devolve false; o corpo nunca é ecoado na resposta.
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	mt, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mt != "application/json" {
		writeError(w, r, http.StatusUnsupportedMediaType, "unsupported_media_type", "Envie o corpo como application/json.", "Defina Content-Type: application/json.")
		return false
	}
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Corpo JSON inválido.", "Confira os campos conforme contracts/openapi.yaml.")
		return false
	}
	return true
}
