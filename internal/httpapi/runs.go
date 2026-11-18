package httpapi

import (
	"errors"
	"net/http"
	"strings"

	"github.com/ViniciusBernucci/le-fabrique/internal/flow"
	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
	"github.com/ViniciusBernucci/le-fabrique/internal/projects"
	"github.com/ViniciusBernucci/le-fabrique/internal/releases"
	"github.com/ViniciusBernucci/le-fabrique/internal/runner"
	"github.com/ViniciusBernucci/le-fabrique/internal/stdflow"
	"github.com/ViniciusBernucci/le-fabrique/internal/workflows"
)

// Execuções do workflow de um projeto (ADR-0069): quem inicia é a pessoa, quem conduz é o worker.

func (s *Server) runID(w http.ResponseWriter, r *http.Request) (string, bool) {
	id := r.PathValue("runId")
	if !uuidPattern.MatchString(id) {
		writeError(w, r, http.StatusNotFound, "not_found", "Recurso não encontrado.", "Verifique o identificador.")
		return "", false
	}
	return id, true
}

func (s *Server) writeRunError(w http.ResponseWriter, r *http.Request, err error) {
	var fe *runner.FieldError
	if errors.As(err, &fe) {
		writeError(w, r, http.StatusBadRequest, "invalid_request", fe.Message, "Corrija o campo \""+fe.Field+"\" e tente novamente.")
		return
	}
	var re *releases.RevalidationError
	if errors.As(err, &re) {
		writeProblems(w, r, http.StatusConflict, "revalidation_failed", re.Error(), "Corrija o que mudou desde a publicação (conta, modelo, skill ou ferramenta) e publique de novo.", re.Problems)
		return
	}
	s.writeDomainError(w, r, err)
}

func (s *Server) startRun(w http.ResponseWriter, r *http.Request) {
	id, ok := s.projectID(w, r)
	if !ok {
		return
	}
	var req struct {
		Title      string `json:"title"`
		Task       string `json:"task"`
		Correction bool   `json:"correction"`
		PRDDone    bool   `json:"prd_done"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	p, err := projects.Get(r.Context(), s.db, id)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	run, err := runner.Start(r.Context(), s.db, runner.StartInput{
		ProjectID: id, Title: req.Title, Task: req.Task, Workspace: p.Repository.Path,
		CreatedBy: s.adminID(r), Correction: req.Correction, PRDDone: req.PRDDone,
	})
	if err != nil {
		s.writeRunError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, run)
}

func (s *Server) listRuns(w http.ResponseWriter, r *http.Request) {
	id, ok := s.projectID(w, r)
	if !ok {
		return
	}
	items, err := runner.List(r.Context(), s.db, id, 30)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) getRun(w http.ResponseWriter, r *http.Request) {
	id, ok := s.projectID(w, r)
	if !ok {
		return
	}
	runID, ok := s.runID(w, r)
	if !ok {
		return
	}
	run, err := runner.Get(r.Context(), s.db, id, runID)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, run)
}

func (s *Server) cancelRun(w http.ResponseWriter, r *http.Request) {
	id, ok := s.projectID(w, r)
	if !ok {
		return
	}
	runID, ok := s.runID(w, r)
	if !ok {
		return
	}
	run, err := runner.Cancel(r.Context(), s.db, id, runID, s.adminID(r))
	if err != nil {
		s.writeRunError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, run)
}

func (s *Server) decideRun(w http.ResponseWriter, r *http.Request) {
	id, ok := s.projectID(w, r)
	if !ok {
		return
	}
	runID, ok := s.runID(w, r)
	if !ok {
		return
	}
	var req struct {
		Node   string `json:"node"`
		Choice string `json:"choice"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	run, err := runner.Decide(r.Context(), s.db, id, runID, req.Node, req.Choice, s.adminID(r))
	if err != nil {
		if errors.Is(err, persistence.ErrInvalidInput) {
			writeError(w, r, http.StatusConflict, "invalid_decision", err.Error(), "Confira o nó e a escolha (approved ou rejected).")
			return
		}
		s.writeRunError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, run)
}

// startTestRun é o botão "Testar o fluxo": publica o rascunho atual (idempotente: mesmo conteúdo, mesma versão) e
// dispara a feature fictícia padronizada. Funciona com o fluxo do projeto ou com qualquer workflow de feature
// (workflow_id), em qualquer projeto: o repositório descartável do teste cobre projetos sem repositório.
func (s *Server) startTestRun(w http.ResponseWriter, r *http.Request) {
	id, ok := s.projectID(w, r)
	if !ok {
		return
	}
	var req struct {
		WorkflowID string `json:"workflow_id"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	p, err := projects.Get(r.Context(), s.db, id)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	wfID := req.WorkflowID
	if wfID == "" {
		wf, _, perr := stdflow.Provision(r.Context(), s.db, id, p.Name)
		if perr != nil {
			s.writeDomainError(w, r, perr)
			return
		}
		wfID = wf.ID
	} else if !uuidPattern.MatchString(wfID) {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Workflow inválido.", "Informe o ID de um workflow existente.")
		return
	}
	draft, err := workflows.Get(r.Context(), s.db, wfID)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	if draft.Kind != workflows.KindFeature {
		writeError(w, r, http.StatusConflict, "not_a_feature_flow", "Subfluxos não têm execução própria.",
			"Teste um fluxo de feature que use este subfluxo: o teste percorre os subfluxos dele.")
		return
	}
	if _, err := s.publisher().Publish(r.Context(), wfID, draft.Version, s.adminID(r)); err != nil {
		var pe *releases.PublishError
		if errors.As(err, &pe) {
			writeProblems(w, r, http.StatusUnprocessableEntity, "publish_blocked", "O fluxo tem pendências e não pôde ser preparado para o teste: "+pe.Error(),
				"Corrija os problemas listados no editor e teste de novo.", pe.Problems)
			return
		}
		s.writeRunError(w, r, err)
		return
	}
	in := runner.StartTestInput(id, p.Repository.Path, s.adminID(r))
	in.WorkflowID = wfID
	run, err := runner.Start(r.Context(), s.db, in)
	if err != nil {
		s.writeRunError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, run)
}

// runDiagnostics devolve o relatório de diagnóstico da execução (Markdown pronto para colar), onde e por que
// falhou e as causas prováveis com o que fazer.
func (s *Server) runDiagnostics(w http.ResponseWriter, r *http.Request) {
	id, ok := s.projectID(w, r)
	if !ok {
		return
	}
	runID, ok := s.runID(w, r)
	if !ok {
		return
	}
	run, err := runner.Get(r.Context(), s.db, id, runID)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	events, err := persistence.EventsAfter(r.Context(), s.db, id, runID, 0, 5000)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	node, reason := runner.Failure(run, events)
	hints := runner.HintsFor(reason)
	if len(hints) == 0 {
		var sb strings.Builder
		for _, e := range events {
			sb.Write(e.Payload)
			sb.WriteByte('\n')
		}
		hints = runner.HintsFor(sb.String())
	}
	if hints == nil {
		hints = []runner.Hint{}
	}
	report := s.redactor.Redact(runner.Report(run, events))
	writeJSON(w, http.StatusOK, map[string]any{"report": report, "hints": hints, "failure": map[string]string{"node": node, "reason": s.redactor.Redact(reason)}})
}

// startIssueRun inicia a fábrica para uma Issue da origem: lê os requisitos na origem (Jira/ClickUp), exige que a Issue
// esteja liberada (PRD-DOC-CONCLUIDO ou CORRECAO_PR) e roda o fluxo PUBLICADO do projeto. A ponte com a Sentinela
// passa a pedir as transições de status conforme o fluxo avança.
func (s *Server) startIssueRun(w http.ResponseWriter, r *http.Request) {
	id, ok := s.projectID(w, r)
	if !ok {
		return
	}
	key := strings.TrimSpace(r.PathValue("key"))
	if key == "" || len(key) > 100 {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Chave da Issue inválida.", "Informe a chave da Issue (ex.: ABC-12).")
		return
	}
	p, err := projects.Get(r.Context(), s.db, id)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	var remoteID, title, status string
	err = s.db.QueryRow(r.Context(), `SELECT remote_id, title, status_code FROM issue_registry WHERE project_id = $1 AND issue_key = $2`, id, key).Scan(&remoteID, &title, &status)
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "A Issue "+key+" não está registrada neste projeto.", "Rode o sync das Issues e tente de novo.")
		return
	}
	if status != flow.PRDDocConcluido && status != flow.CorrecaoPR {
		writeError(w, r, http.StatusConflict, "issue_not_released", "A Issue está em "+status+": a fábrica só executa Issues em "+flow.PRDDocConcluido+" ou "+flow.CorrecaoPR+".",
			"Complete o PRD e o contrato inicial (ou libere a correção na origem) antes de executar.")
		return
	}
	tr, err := s.sourceTracker(r, p)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	item, err := tr.GetItem(r.Context(), remoteID)
	if err != nil {
		writeError(w, r, http.StatusBadGateway, "origin_unavailable", "Não consegui ler a Issue na origem.", "Confira a conexão da origem e tente de novo.")
		return
	}
	task := "Issue " + key + ": " + title + "\n\n" + strings.TrimSpace(item.Description)
	run, err := runner.Start(r.Context(), s.db, runner.StartInput{
		ProjectID: id, Title: key + " — " + title, Task: task, Workspace: p.Repository.Path, CreatedBy: s.adminID(r),
		IssueKey: key, PRDDone: true, Correction: status == flow.CorrecaoPR,
	})
	if err != nil {
		s.writeRunError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, run)
}
