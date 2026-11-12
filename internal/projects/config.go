package projects

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
)

// ErrInvalidState indica transição incompatível com o estado atual do projeto.
var ErrInvalidState = errors.New("estado do projeto não permite a operação")

// ErrActiveLocked: a configuração de repositório e origem só muda com o projeto
// em rascunho ou suspenso, para que a troca nunca aconteça sob execução.
var ErrActiveLocked = errors.New("projeto ativo: suspenda antes de alterar repositório ou origem")

// FieldError descreve um campo inválido com mensagem segura para o usuário.
type FieldError struct {
	Field   string
	Message string
}

func (e *FieldError) Error() string { return e.Field + ": " + e.Message }
func (e *FieldError) Unwrap() error { return persistence.ErrInvalidInput }

var (
	ownerPattern     = regexp.MustCompile(`^[A-Za-z0-9](?:[A-Za-z0-9-]{0,37}[A-Za-z0-9])?$`)
	repoNamePattern  = regexp.MustCompile(`^[A-Za-z0-9._-]{1,100}$`)
	branchPattern    = regexp.MustCompile(`^[A-Za-z0-9._/-]{1,200}$`)
	jiraKeyPattern   = regexp.MustCompile(`^[A-Z][A-Z0-9_]{1,19}$`)
	containerPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)
)

// Patch descreve alterações; campos nil permanecem. Repository e Source são
// grupos de uma etapa do assistente: quando presentes substituem o grupo todo
// (campos vazios continuam permitidos no rascunho e viram pendências).
type Patch struct {
	Name        *string
	Description *string
	State       *string
	Priority    *int
	WizardStep  *string
	Repository  *Repository
	Source      *Source
}

func (p Patch) touchesConfig() bool { return p.Repository != nil || p.Source != nil }

// normalize apara texto e valida o formato dos campos preenchidos.
func (p *Patch) normalize() error {
	if p.Name != nil {
		n := strings.TrimSpace(*p.Name)
		if n == "" || len(n) > 120 {
			return &FieldError{"name", "O nome deve ter entre 1 e 120 caracteres."}
		}
		p.Name = &n
	}
	if p.Description != nil {
		d := strings.TrimSpace(*p.Description)
		if len(d) > 500 {
			return &FieldError{"description", "A descrição deve ter até 500 caracteres."}
		}
		p.Description = &d
	}
	if p.Priority != nil && (*p.Priority < 0 || *p.Priority > 4) {
		return &FieldError{"priority", "A prioridade deve estar entre 0 e 4."}
	}
	if p.State != nil && !validStates[*p.State] {
		return &FieldError{"state", "Estado desconhecido."}
	}
	if p.WizardStep != nil && !validSteps[*p.WizardStep] {
		return &FieldError{"wizard_step", "Etapa desconhecida."}
	}
	if r := p.Repository; r != nil {
		c := *r
		c.Owner, c.Name, c.BaseBranch = strings.TrimSpace(c.Owner), strings.TrimSpace(c.Name), strings.TrimSpace(c.BaseBranch)
		if c.Owner != "" && !ownerPattern.MatchString(c.Owner) {
			return &FieldError{"repository.owner", "Dono do repositório inválido: use letras, números e hífens (até 39)."}
		}
		if c.Name != "" && (!repoNamePattern.MatchString(c.Name) || c.Name == "." || c.Name == "..") {
			return &FieldError{"repository.name", "Nome do repositório inválido: use letras, números, ponto, hífen e sublinhado."}
		}
		if c.BaseBranch != "" && !validBranch(c.BaseBranch) {
			return &FieldError{"repository.base_branch", "Branch base inválida."}
		}
		c.ConnectionID = blankToNil(c.ConnectionID)
		p.Repository = &c
	}
	if src := p.Source; src != nil {
		c := *src
		c.Kind, c.Container = strings.TrimSpace(c.Kind), strings.TrimSpace(c.Container)
		if c.Kind != "" && c.Kind != "jira" && c.Kind != "clickup" {
			return &FieldError{"source.kind", "A origem deve ser Jira ou ClickUp."}
		}
		if c.Container != "" {
			ok := containerPattern.MatchString(c.Container)
			if c.Kind == "jira" {
				ok = jiraKeyPattern.MatchString(c.Container)
			}
			if !ok {
				return &FieldError{"source.container", "Identificador da origem inválido (Jira: chave do projeto em maiúsculas, ex.: ABC; ClickUp: ID da lista)."}
			}
		}
		c.ConnectionID = blankToNil(c.ConnectionID)
		if err := normalizeEligibility(&c); err != nil {
			return err
		}
		p.Source = &c
	}
	return nil
}

var labelPattern = regexp.MustCompile(`^[^\x00-\x1f"\\]{1,50}$`)

// normalizeEligibility valida marcador, estados elegíveis, mapa de prioridade e
// escopo; cada falha diz qual campo e por quê (mapeamento inválido falha com motivo).
func normalizeEligibility(c *Source) error {
	c.EligibilityMarker = strings.TrimSpace(c.EligibilityMarker)
	if c.EligibilityMarker != "" {
		ok := labelPattern.MatchString(c.EligibilityMarker)
		if c.Kind == "jira" {
			ok = ok && !strings.ContainsAny(c.EligibilityMarker, " ")
		}
		if !ok {
			return &FieldError{"source.eligibility_marker", "Marcador inválido: use até 50 caracteres sem aspas (no Jira, sem espaços)."}
		}
	}
	seen := map[string]bool{}
	statuses := []string{}
	for _, st := range c.EligibleStatuses {
		st = strings.TrimSpace(st)
		if st == "" || len(st) > 80 {
			return &FieldError{"source.eligible_statuses", "Cada estado elegível deve ter entre 1 e 80 caracteres."}
		}
		if !seen[strings.ToLower(st)] {
			seen[strings.ToLower(st)] = true
			statuses = append(statuses, st)
		}
	}
	if len(statuses) > 30 {
		return &FieldError{"source.eligible_statuses", "No máximo 30 estados elegíveis."}
	}
	c.EligibleStatuses = statuses
	pm := map[string]int{}
	for k, v := range c.PriorityMap {
		k = strings.ToLower(strings.TrimSpace(k))
		if k == "" || len(k) > 60 {
			return &FieldError{"source.priority_map", "Cada prioridade da origem deve ter entre 1 e 60 caracteres."}
		}
		if v < 0 || v > 4 {
			return &FieldError{"source.priority_map", "A prioridade \"" + k + "\" deve mapear para um valor entre 0 e 4."}
		}
		pm[k] = v
	}
	if len(pm) > 20 {
		return &FieldError{"source.priority_map", "No máximo 20 prioridades mapeadas."}
	}
	c.PriorityMap = pm
	switch c.Access {
	case "":
		c.Access = AccessRead
	case AccessRead, AccessReadWrite:
	default:
		return &FieldError{"source.access", "O escopo deve ser leitura ou leitura e escrita."}
	}
	return nil
}

func blankToNil(s *string) *string {
	if s == nil || strings.TrimSpace(*s) == "" {
		return nil
	}
	v := strings.TrimSpace(*s)
	return &v
}

func validBranch(b string) bool {
	if !branchPattern.MatchString(b) || strings.Contains(b, "..") || strings.Contains(b, "//") ||
		strings.HasPrefix(b, "/") || strings.HasSuffix(b, "/") || strings.HasPrefix(b, "-") ||
		strings.HasSuffix(b, ".") || strings.HasSuffix(b, ".lock") {
		return false
	}
	return true
}

// Update altera o projeto com controle otimista: se expectedVersion não for a
// atual, devolve persistence.ErrVersionConflict. Alterar repositório ou origem
// de um projeto ativo devolve ErrActiveLocked.
func Update(ctx context.Context, db persistence.DBTX, id string, expectedVersion int, p Patch) (Project, error) {
	if err := p.normalize(); err != nil {
		return Project{}, err
	}
	repo := Repository{}
	if p.Repository != nil {
		repo = *p.Repository
	}
	src := Source{}
	if p.Source != nil {
		src = *p.Source
	}
	statusesJSON, _ := json.Marshal(nonNilStrings(src.EligibleStatuses))
	priorityJSON, _ := json.Marshal(nonNilMap(src.PriorityMap))
	var srcKind *string
	if src.Kind != "" {
		srcKind = &src.Kind
	}
	updated, err := scan(db.QueryRow(ctx, `
		UPDATE projects SET
			name = COALESCE($3, name),
			description = COALESCE($4, description),
			state = COALESCE($5, state),
			priority = COALESCE($6, priority),
			wizard_step = COALESCE($7, wizard_step),
			repo_owner = CASE WHEN $8::boolean THEN $9 ELSE repo_owner END,
			repo_name = CASE WHEN $8::boolean THEN $10 ELSE repo_name END,
			base_branch = CASE WHEN $8::boolean THEN $11 ELSE base_branch END,
			repo_connection_id = CASE WHEN $8::boolean THEN $12::uuid ELSE repo_connection_id END,
			source_kind = CASE WHEN $13::boolean THEN $14::text ELSE source_kind END,
			source_connection_id = CASE WHEN $13::boolean THEN $15::uuid ELSE source_connection_id END,
			source_container = CASE WHEN $13::boolean THEN $16 ELSE source_container END,
			eligibility_marker = CASE WHEN $13::boolean THEN $18 ELSE eligibility_marker END,
			eligible_statuses = CASE WHEN $13::boolean THEN $19::jsonb ELSE eligible_statuses END,
			priority_map = CASE WHEN $13::boolean THEN $20::jsonb ELSE priority_map END,
			source_access = CASE WHEN $13::boolean THEN $21 ELSE source_access END,
			activated_at = CASE WHEN $5::text = 'active' AND activated_at IS NULL THEN now() ELSE activated_at END,
			version = version + 1, updated_at = now()
		WHERE id = $1 AND version = $2 AND NOT ($17::boolean AND state = 'active')
		RETURNING `+columns,
		id, expectedVersion, p.Name, p.Description, p.State, p.Priority, p.WizardStep,
		p.Repository != nil, repo.Owner, repo.Name, repo.BaseBranch, repo.ConnectionID,
		p.Source != nil, srcKind, src.ConnectionID, src.Container,
		p.touchesConfig(), src.EligibilityMarker, string(statusesJSON), string(priorityJSON), accessOrRead(src.Access)))
	if errors.Is(err, persistence.ErrNotFound) {
		if cur, getErr := Get(ctx, db, id); getErr == nil {
			if cur.Version != expectedVersion {
				return Project{}, persistence.ErrVersionConflict
			}
			return Project{}, ErrActiveLocked
		}
	}
	return updated, err
}

func nonNilStrings(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

func nonNilMap(m map[string]int) map[string]int {
	if m == nil {
		return map[string]int{}
	}
	return m
}

func accessOrRead(a string) string {
	if a == "" {
		return AccessRead
	}
	return a
}

// Status de cada item do relatório de validação.
const (
	CheckOK      = "ok"
	CheckPending = "pending"
)

// Check é um item de validação com a ação concreta para resolvê-lo.
type Check struct {
	ID     string `json:"id"`
	Area   string `json:"area"` // basics | repository | source
	Label  string `json:"label"`
	Status string `json:"status"`
	Action string `json:"action,omitempty"`
}

// Report é o estado de validação do projeto. OK só é verdadeiro quando todos os
// itens obrigatórios estão atendidos; NotVerified lista o que ainda não é
// avaliado (entra como novos itens em T05-02 e T05-03).
type Report struct {
	OK          bool     `json:"ok"`
	Checks      []Check  `json:"checks"`
	NotVerified []string `json:"not_verified"`
}

// ValidationError bloqueia a ativação e carrega o relatório.
type ValidationError struct{ Report Report }

func (e *ValidationError) Error() string {
	n := 0
	for _, c := range e.Report.Checks {
		if c.Status != CheckOK {
			n++
		}
	}
	return fmt.Sprintf("configuração incompleta: %d pendência(s)", n)
}

var notVerified = []string{
	"Acesso remoto ao repositório GitHub (T05-02, depende do adaptador GitHub — T15-01)",
	"Permissão de escrita na origem: só é provada ao escrever, no primeiro uso (T06-03)",
	"Workflows, agentes e demais recursos associados (T05-03)",
}

type connInfo struct {
	kind, state string
	secretVer   int
	found       bool
}

func lookupConnection(ctx context.Context, db persistence.DBTX, id *string) (connInfo, error) {
	if id == nil {
		return connInfo{}, nil
	}
	var c connInfo
	err := db.QueryRow(ctx, `SELECT kind, state, secret_version FROM connections WHERE id = $1`, *id).Scan(&c.kind, &c.state, &c.secretVer)
	if errors.Is(persistence.MapError(err), persistence.ErrNotFound) {
		return connInfo{}, nil
	}
	if err != nil {
		return connInfo{}, persistence.MapError(err)
	}
	c.found = true
	return c, nil
}

// Validate avalia o preenchimento obrigatório do projeto. Não acessa serviços externos.
func Validate(ctx context.Context, db persistence.DBTX, p Project) (Report, error) {
	var checks []Check
	add := func(id, area, label string, ok bool, action string) {
		c := Check{ID: id, Area: area, Label: label, Status: CheckOK}
		if !ok {
			c.Status, c.Action = CheckPending, action
		}
		checks = append(checks, c)
	}

	add("basics.name", "basics", "Nome do projeto", strings.TrimSpace(p.Name) != "", "Informe o nome do projeto na etapa Dados.")

	add("repository.owner", "repository", "Dono do repositório GitHub", p.Repository.Owner != "", "Informe o dono (usuário ou organização) na etapa Repositório.")
	add("repository.name", "repository", "Nome do repositório", p.Repository.Name != "", "Informe o nome do repositório na etapa Repositório.")
	add("repository.base_branch", "repository", "Branch base", p.Repository.BaseBranch != "", "Informe a branch base (ex.: main) na etapa Repositório.")
	repoConn, err := lookupConnection(ctx, db, p.Repository.ConnectionID)
	if err != nil {
		return Report{}, err
	}
	connOK, connAction := connectionState(p.Repository.ConnectionID, repoConn, "github", "GitHub", "Repositório")
	add("repository.connection", "repository", "Conexão GitHub", connOK, connAction)

	add("source.kind", "source", "Plataforma da origem de tarefas", p.Source.Kind != "", "Escolha Jira ou ClickUp na etapa Origem.")
	srcConn, err := lookupConnection(ctx, db, p.Source.ConnectionID)
	if err != nil {
		return Report{}, err
	}
	srcLabel := map[string]string{"jira": "Jira", "clickup": "ClickUp"}[p.Source.Kind]
	if p.Source.Kind == "" {
		add("source.connection", "source", "Conexão da origem", false, "Escolha a plataforma e depois a conexão na etapa Origem.")
	} else {
		ok, action := connectionState(p.Source.ConnectionID, srcConn, p.Source.Kind, srcLabel, "Origem")
		add("source.connection", "source", "Conexão "+srcLabel, ok, action)
	}
	add("source.container", "source", "Projeto ou lista na origem", p.Source.Container != "", "Informe a chave do projeto (Jira) ou o ID da lista (ClickUp) na etapa Origem.")
	add("source.eligibility", "source", "Critério de elegibilidade",
		p.Source.EligibilityMarker != "" || len(p.Source.EligibleStatuses) > 0,
		"Defina um marcador (rótulo/tag) ou ao menos um estado elegível na etapa Origem.")

	// Duas origens ativas: a mesma lista não pode alimentar dois projetos ativos.
	if p.Source.Kind != "" && p.Source.Container != "" {
		other, err := activeSourceOwner(ctx, db, p)
		if err != nil {
			return Report{}, err
		}
		add("source.exclusive", "source", "Origem exclusiva", other == "",
			"Esta lista já alimenta o projeto ativo \""+other+"\". Suspenda-o ou escolha outra lista na etapa Origem.")
	}

	// Verificação remota: precisa existir, ter passado e valer para a configuração atual.
	remoteOK, remoteAction, err := remoteCheckState(ctx, db, p, srcConn)
	if err != nil {
		return Report{}, err
	}
	add("source.remote", "source", "Verificação remota da origem", remoteOK, remoteAction)

	ok := true
	for _, c := range checks {
		ok = ok && c.Status == CheckOK
	}
	return Report{OK: ok, Checks: checks, NotVerified: notVerified}, nil
}

func connectionState(id *string, c connInfo, wantKind, kindLabel, stepLabel string) (bool, string) {
	switch {
	case id == nil:
		return false, "Escolha uma conexão " + kindLabel + " ativa na etapa " + stepLabel + ", ou cadastre uma em Conexões."
	case !c.found:
		return false, "A conexão selecionada não existe mais. Escolha outra na etapa " + stepLabel + "."
	case c.state != "active":
		return false, "A conexão selecionada está revogada. Escolha outra conexão " + kindLabel + " ativa."
	case c.kind != wantKind:
		return false, "A conexão selecionada não é do tipo " + kindLabel + ". Escolha uma conexão " + kindLabel + "."
	}
	return true, ""
}

// activeSourceOwner devolve o nome de outro projeto ativo que usa a mesma origem
// (mesma plataforma e lista, na mesma conexão ou no mesmo destino), ou "".
func activeSourceOwner(ctx context.Context, db persistence.DBTX, p Project) (string, error) {
	var name string
	err := db.QueryRow(ctx, `
		SELECT o.name FROM projects o
		LEFT JOIN connections oc ON oc.id = o.source_connection_id
		LEFT JOIN connections mc ON mc.id = $4::uuid
		WHERE o.id <> $1 AND o.state = 'active' AND o.source_kind = $2 AND o.source_container = $3
		  AND (o.source_connection_id = $4::uuid OR (mc.target <> '' AND oc.target = mc.target))
		ORDER BY o.created_at LIMIT 1`,
		p.ID, p.Source.Kind, p.Source.Container, p.Source.ConnectionID).Scan(&name)
	if errors.Is(persistence.MapError(err), persistence.ErrNotFound) {
		return "", nil
	}
	return name, persistence.MapError(err)
}

func remoteCheckState(ctx context.Context, db persistence.DBTX, p Project, conn connInfo) (bool, string, error) {
	var ok *bool
	var fp string
	err := db.QueryRow(ctx, `SELECT source_check_ok, source_check_fingerprint FROM projects WHERE id = $1`, p.ID).Scan(&ok, &fp)
	if err != nil {
		return false, "", persistence.MapError(err)
	}
	const run = "Use \"Verificar origem\" na etapa Origem para provar o acesso e ver os itens elegíveis."
	switch {
	case ok == nil:
		return false, "A origem ainda não foi verificada. " + run, nil
	case fp != Fingerprint(p, conn.secretVer):
		return false, "A configuração, a conexão ou o segredo mudou depois da última verificação. " + run, nil
	case !*ok:
		return false, "A última verificação da origem falhou. Corrija os pontos indicados e verifique de novo. " + run, nil
	}
	return true, "", nil
}
