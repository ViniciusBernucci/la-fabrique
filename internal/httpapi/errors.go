package httpapi

import (
	"encoding/json"
	"errors"
	"mime"
	"net/http"
	"strconv"

	"github.com/ViniciusBernucci/le-fabrique/internal/agents"
	"github.com/ViniciusBernucci/le-fabrique/internal/budget"
	"github.com/ViniciusBernucci/le-fabrique/internal/decisions"
	"github.com/ViniciusBernucci/le-fabrique/internal/mcp"
	"github.com/ViniciusBernucci/le-fabrique/internal/models"
	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
	"github.com/ViniciusBernucci/le-fabrique/internal/projects"
	"github.com/ViniciusBernucci/le-fabrique/internal/releases"
	"github.com/ViniciusBernucci/le-fabrique/internal/security"
	"github.com/ViniciusBernucci/le-fabrique/internal/selection"
	"github.com/ViniciusBernucci/le-fabrique/internal/skills"
	"github.com/ViniciusBernucci/le-fabrique/internal/syncsource"
	"github.com/ViniciusBernucci/le-fabrique/internal/tools"
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
	var agField *agents.FieldError
	var impErr *skills.ImportError
	var valErr *skills.ValidationError
	var argErr *tools.ArgError
	var decField *decisions.FieldError
	var mcpField *mcp.FieldError
	var mcpErr *mcp.Error
	var modField *models.FieldError
	var selField *selection.FieldError
	var budField *budget.FieldError
	var budExceeded *budget.ExceededError
	switch {
	case errors.As(err, &field):
		writeError(w, r, http.StatusBadRequest, "invalid_request", field.Message, "Corrija o campo \""+field.Field+"\" e tente novamente.")
	case errors.As(err, &budField):
		writeError(w, r, http.StatusBadRequest, "invalid_request", budField.Message, "Corrija o campo \""+budField.Field+"\" e tente novamente.")
	case errors.As(err, &budExceeded):
		writeError(w, r, http.StatusConflict, "budget_exceeded", budExceeded.Error(), "Aguarde o próximo período ou aumente o orçamento.")
	case errors.Is(err, budget.ErrNoBudget):
		writeError(w, r, http.StatusConflict, "no_budget", err.Error(), "Defina um orçamento global antes de usar contas de API.")
	case errors.Is(err, budget.ErrNoPrice):
		writeError(w, r, http.StatusConflict, "model_without_price", err.Error(), "Cadastre o preço do modelo (com a fonte).")
	case errors.As(err, &selField):
		writeError(w, r, http.StatusBadRequest, "invalid_request", selField.Message, "Corrija o campo \""+selField.Field+"\" e tente novamente.")
	case errors.Is(err, selection.ErrNoSelection):
		writeError(w, r, http.StatusConflict, "no_selection", err.Error(), "Defina um fluxo padrão para o projeto.")
	case errors.Is(err, releases.ErrNotPublished):
		writeError(w, r, http.StatusConflict, "not_published", err.Error(), "Publique o workflow antes de usá-lo.")
	case errors.As(err, &modField):
		writeError(w, r, http.StatusBadRequest, "invalid_request", modField.Message, "Corrija o campo \""+modField.Field+"\" e tente novamente.")
	case errors.Is(err, models.ErrIncompatible):
		writeError(w, r, http.StatusConflict, "model_incompatible", err.Error(), "Use um modelo cadastrado, ativo e compatível com a conta e as ferramentas do agente.")
	case errors.As(err, &mcpField):
		writeError(w, r, http.StatusBadRequest, "invalid_request", mcpField.Message, "Corrija o campo \""+mcpField.Field+"\" e tente novamente.")
	case errors.Is(err, mcp.ErrNotGranted):
		writeError(w, r, http.StatusForbidden, "tool_not_granted", err.Error(), "Conceda a ferramenta ao agente (descobrir não concede acesso).")
	case errors.Is(err, mcp.ErrSchemaChanged):
		writeError(w, r, http.StatusConflict, "schema_changed", err.Error(), "Revise o novo contrato e aprove de novo.")
	case errors.Is(err, mcp.ErrStaleApproval):
		writeError(w, r, http.StatusConflict, "stale_approval", err.Error(), "Descubra de novo e aprove o esquema vigente.")
	case errors.Is(err, mcp.ErrToolMissing):
		writeError(w, r, http.StatusNotFound, "tool_not_found", err.Error(), "Descubra as ferramentas do servidor de novo.")
	case errors.As(err, &mcpErr):
		writeError(w, r, http.StatusBadGateway, "mcp_error", mcpErr.Error(), "Verifique o servidor MCP e a conexão.")
	case errors.As(err, &decField):
		writeError(w, r, http.StatusBadRequest, "invalid_request", decField.Message, "Corrija o campo \""+decField.Field+"\" e tente novamente.")
	case errors.Is(err, decisions.ErrInvalidOption):
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Opção inexistente neste pedido.", "Escolha uma das alternativas listadas.")
	case errors.Is(err, decisions.ErrAlreadyResolved):
		writeError(w, r, http.StatusConflict, "decision_conflict", "O pedido já foi resolvido com outra opção.", "Recarregue a Central: a decisão registrada não pode ser trocada.")
	case errors.Is(err, decisions.ErrStale):
		writeError(w, r, http.StatusConflict, "decision_stale", "A execução mudou desde o pedido: ele ficou obsoleto.", "Recarregue a Central; se ainda for preciso, a fábrica abrirá um novo pedido.")
	case errors.As(err, &argErr):
		writeError(w, r, http.StatusBadRequest, "invalid_request", argErr.Error(), "Corrija o campo \""+argErr.Field+"\" e tente novamente.")
	case errors.Is(err, tools.ErrUnknownTool):
		writeError(w, r, http.StatusNotFound, "tool_not_found", err.Error(), "Confira o nome em GET /tools.")
	case errors.Is(err, tools.ErrNotGranted):
		writeError(w, r, http.StatusForbidden, "tool_not_granted", err.Error(), "Conceda a ferramenta ao agente (uma ferramenta nova não herda acesso).")
	case errors.Is(err, tools.ErrScope):
		writeError(w, r, http.StatusForbidden, "scope_insufficient", err.Error(), "Conceda a permissão exigida ao agente.")
	case errors.Is(err, tools.ErrNativeName):
		writeError(w, r, http.StatusConflict, "duplicate", "O nome pertence a uma ferramenta nativa.", "Use outro nome.")
	case errors.Is(err, tools.ErrPolicy):
		writeError(w, r, http.StatusUnprocessableEntity, "policy_denied", err.Error(), "Use um comando permitido pela política de risco.")
	case errors.As(err, &impErr):
		writeError(w, r, http.StatusUnprocessableEntity, "import_rejected", impErr.Error(), "Corrija o pacote ("+impErr.Code+") e importe de novo.")
	case errors.As(err, &valErr):
		writeError(w, r, http.StatusBadRequest, "invalid_request", valErr.Error(), "Corrija o campo \""+valErr.Field+"\" e tente novamente.")
	case errors.As(err, &agField):
		writeError(w, r, http.StatusBadRequest, "invalid_request", agField.Message, "Corrija o campo \""+agField.Field+"\" e tente novamente.")
	case errors.Is(err, agents.ErrConnection):
		writeError(w, r, http.StatusConflict, "connection_incompatible", err.Error(), "Escolha uma conexão ativa e compatível com o provedor do agente.")
	case errors.Is(err, agents.ErrAgentNotFound):
		writeError(w, r, http.StatusUnprocessableEntity, "agent_not_found", "O agente do nó não existe no catálogo.", "Cadastre o agente ou corrija o nome no nó.")
	case errors.Is(err, agents.ErrOverride):
		writeError(w, r, http.StatusUnprocessableEntity, "invalid_override", err.Error(), "Corrija a sobrescrita do nó.")
	case errors.As(err, &wfField):
		writeError(w, r, http.StatusBadRequest, "invalid_request", wfField.Message, "Corrija o campo \""+wfField.Field+"\" e tente novamente.")
	case errors.As(err, &invalid):
		writeError(w, r, http.StatusUnprocessableEntity, "validation_failed", "O projeto tem pendências e não pode ser ativado.", "Resolva os itens pendentes em GET /api/projects/{id}/validation.")
	case errors.Is(err, syncsource.ErrNoConflict):
		writeError(w, r, http.StatusConflict, "no_conflict", err.Error(), "Sincronize de novo e confira os itens com conflito.")
	case errors.Is(err, syncsource.ErrNoItem):
		writeError(w, r, http.StatusNotFound, "not_found", err.Error(), "Sincronize a origem antes de editar o item.")
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
