package httpapi

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/ViniciusBernucci/le-fabrique/internal/skills"
)

type skillView struct {
	skills.Skill
	Files []skills.FileInfo `json:"files"`
}

func (s *Server) listSkills(w http.ResponseWriter, r *http.Request) {
	items, err := skills.List(r.Context(), s.db)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) getSkill(w http.ResponseWriter, r *http.Request) {
	id, ok := s.projectID(w, r)
	if !ok {
		return
	}
	sk, err := skills.Get(r.Context(), s.db, id)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	files, err := skills.Files(r.Context(), s.db, id)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, skillView{Skill: sk, Files: files})
}

func (s *Server) listSkillImports(w http.ResponseWriter, r *http.Request) {
	items, err := skills.Imports(r.Context(), s.db, 50)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

// saveSkill valida e guarda; em recusa registra a tentativa (sem o conteúdo).
func (s *Server) saveSkill(w http.ResponseWriter, r *http.Request, pkg skills.Package, ingestErr error, kind, ref string, replace bool, status int) {
	if ingestErr != nil {
		var ie *skills.ImportError
		if errors.As(ingestErr, &ie) {
			_ = skills.RecordRejected(r.Context(), s.db, "", kind, ref, ie.Error())
		}
		s.writeDomainError(w, r, ingestErr)
		return
	}
	sk, outcome, err := skills.Save(r.Context(), s.db, pkg, kind, ref, "", replace)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	if outcome == skills.OutcomeUnchanged {
		status = http.StatusOK
	}
	w.Header().Set("X-Import-Outcome", outcome)
	writeJSON(w, status, sk)
}

// createSkill cadastra uma skill manualmente (IA, script, determinística ou híbrida).
func (s *Server) createSkill(w http.ResponseWriter, r *http.Request) {
	var req struct {
		skills.Manifest
		Instructions string `json:"instructions"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	pkg, err := skills.FromManual(req.Manifest, req.Instructions, s.policy)
	s.saveSkill(w, r, pkg, err, skills.SourceManual, "", false, http.StatusCreated)
}

// importSkill importa um pacote (tar.gz ou zip) ou um SKILL.md avulso (format=md).
// O conteúdo é tratado como dado: nada é executado.
func (s *Server) importSkill(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	format := q.Get("format")
	replace := q.Get("replace") == "true"
	ref := strings.TrimSpace(q.Get("ref"))
	r.Body = http.MaxBytesReader(w, r.Body, skills.MaxArchive+1)
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, r, http.StatusRequestEntityTooLarge, "payload_too_large", "O pacote é grande demais.", "Reduza o pacote (até 8 MiB).")
		return
	}
	var files []skills.File
	kind := skills.SourcePackage
	switch format {
	case "md":
		kind = skills.SourceFile
		files = []skills.File{{Path: "SKILL.md", Content: raw}}
	case "tar.gz", "zip":
		files, err = skills.ReadArchive(bytes.NewReader(raw), format)
	default:
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Informe format=tar.gz, zip ou md.", "Ajuste o parâmetro format.")
		return
	}
	var pkg skills.Package
	if err == nil {
		pkg, err = skills.Ingest(files, s.policy)
	}
	s.saveSkill(w, r, pkg, err, kind, ref, replace, http.StatusCreated)
}

// importSkillGit: a importação por Git depende do adaptador GitHub (T15-01).
func (s *Server) importSkillGit(w http.ResponseWriter, r *http.Request) {
	writeError(w, r, http.StatusNotImplemented, "unsupported", "A importação por Git ainda não está disponível.", "Importe o repositório como pacote (tar.gz ou zip) por enquanto; o adaptador GitHub é o T15-01.")
}
