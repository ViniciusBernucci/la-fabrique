package projects

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/ViniciusBernucci/le-fabrique/internal/integrations"
	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
)

// Resultados de cada passo da verificação.
const (
	StepOK      = "ok"
	StepFailed  = "failed"
	StepSkipped = "skipped"
)

// SourceStep é um passo da verificação remota da origem.
type SourceStep struct {
	ID     string `json:"id"`
	Label  string `json:"label"`
	Status string `json:"status"`
	Detail string `json:"detail"`
}

// PreviewItem é um item elegível como a fábrica o veria. Priority é a da
// origem; Mapped, a da fábrica (nil quando o mapa não cobre).
type PreviewItem struct {
	Key      string `json:"key"`
	Title    string `json:"title"`
	Status   string `json:"status"`
	Priority string `json:"priority"`
	Mapped   *int   `json:"mapped_priority"`
	Subtask  bool   `json:"subtask"`
	URL      string `json:"url"`
}

// SourceCheck é o resultado da verificação remota da origem.
type SourceCheck struct {
	OK        bool          `json:"ok"`
	Steps     []SourceStep  `json:"steps"`
	Items     []PreviewItem `json:"items"`
	HasMore   bool          `json:"has_more"`
	CheckedAt time.Time     `json:"checked_at"`
}

// Fingerprint identifica a configuração da origem e a versão do segredo da
// conexão. A verificação só vale enquanto ela não muda.
func Fingerprint(p Project, secretVersion int) string {
	statuses := append([]string{}, p.Source.EligibleStatuses...)
	sort.Strings(statuses)
	keys := make([]string, 0, len(p.Source.PriorityMap))
	for k, v := range p.Source.PriorityMap {
		keys = append(keys, fmt.Sprintf("%s=%d", k, v))
	}
	sort.Strings(keys)
	conn := ""
	if p.Source.ConnectionID != nil {
		conn = *p.Source.ConnectionID
	}
	h := sha256.Sum256([]byte(strings.Join([]string{
		p.Source.Kind, conn, fmt.Sprint(secretVersion), p.Source.Container, p.Source.EligibilityMarker,
		strings.Join(statuses, "\x1f"), strings.Join(keys, "\x1f"), p.Source.Access,
	}, "\x1e")))
	return hex.EncodeToString(h[:])
}

const previewLimit = 20

// ErrSourceIncomplete: faltam plataforma, conexão ou lista para verificar.
var ErrSourceIncomplete = errors.New("origem incompleta")

func failure(err error) string {
	var e *integrations.Error
	if !errors.As(err, &e) {
		return "Falha inesperada ao consultar a origem."
	}
	switch e.Kind {
	case integrations.ErrAuth:
		return "A plataforma recusou a credencial. Rotacione ou substitua a conexão."
	case integrations.ErrPermission:
		return "A credencial não tem permissão para este projeto ou lista. Conceda leitura ou use outra conexão."
	case integrations.ErrNotFound:
		return "Projeto ou lista não encontrado, ou invisível para esta credencial."
	case integrations.ErrRateLimited:
		if e.RetryAfter > 0 {
			return fmt.Sprintf("A plataforma limitou as requisições. Tente de novo em cerca de %d s.", int(e.RetryAfter.Seconds())+1)
		}
		return "A plataforma limitou as requisições. Tente de novo em instantes."
	case integrations.ErrUnavailable:
		return "A plataforma não respondeu. Tente novamente."
	default:
		return "A plataforma recusou os parâmetros (projeto, lista ou marcador)."
	}
}

func isDone(i integrations.Item) bool { return i.Category == integrations.CategoryDone }

// CheckSource prova o acesso à origem e monta o preview dos itens elegíveis:
// credencial, projeto/lista, estados e prioridades do mapa, e a listagem. Não
// escreve na origem. Falhas viram passos "failed" com motivo, não erros.
func CheckSource(ctx context.Context, tr integrations.Tracker, p Project, now time.Time) SourceCheck {
	res := SourceCheck{CheckedAt: now, Steps: []SourceStep{}, Items: []PreviewItem{}}
	add := func(id, label, status, detail string) {
		res.Steps = append(res.Steps, SourceStep{id, label, status, detail})
	}
	id, err := tr.Identity(ctx)
	if err != nil {
		add("credential", "Credencial aceita", StepFailed, failure(err))
		return res
	}
	add("credential", "Credencial aceita", StepOK, "Conta "+id.Name+".")

	statuses, err := tr.Statuses(ctx, p.Source.Container)
	if err != nil {
		add("container", "Projeto ou lista acessível", StepFailed, failure(err))
		return res
	}
	add("container", "Projeto ou lista acessível", StepOK, fmt.Sprintf("%d estados encontrados.", len(statuses)))

	known := map[string]bool{}
	for _, s := range statuses {
		known[strings.ToLower(s.Name)] = true
	}
	var missing []string
	for _, st := range p.Source.EligibleStatuses {
		if !known[strings.ToLower(st)] {
			missing = append(missing, st)
		}
	}
	switch {
	case len(p.Source.EligibleStatuses) == 0:
		add("statuses", "Estados elegíveis existem na origem", StepSkipped, "Nenhum estado elegível definido; vale só o marcador.")
	case len(missing) > 0:
		add("statuses", "Estados elegíveis existem na origem", StepFailed,
			"Estados inexistentes na origem: "+strings.Join(missing, ", ")+". Use: "+joinStatusNames(statuses)+".")
	default:
		add("statuses", "Estados elegíveis existem na origem", StepOK, "Todos existem.")
	}

	prios, err := tr.Priorities(ctx)
	if err != nil {
		add("priorities", "Prioridades do mapa existem na origem", StepFailed, failure(err))
		return res
	}
	pk := map[string]bool{}
	for _, n := range prios {
		pk[strings.ToLower(n)] = true
	}
	var badPrio []string
	for k := range p.Source.PriorityMap {
		if !pk[k] {
			badPrio = append(badPrio, k)
		}
	}
	sort.Strings(badPrio)
	switch {
	case len(p.Source.PriorityMap) == 0:
		add("priorities", "Prioridades do mapa existem na origem", StepSkipped, "Sem mapa de prioridade: todos os itens usarão a prioridade padrão do projeto.")
	case len(badPrio) > 0:
		add("priorities", "Prioridades do mapa existem na origem", StepFailed,
			"Prioridades inexistentes na origem: "+strings.Join(badPrio, ", ")+". Use: "+strings.Join(prios, ", ")+".")
	default:
		add("priorities", "Prioridades do mapa existem na origem", StepOK, "Todas existem.")
	}

	eligible := map[string]bool{}
	for _, st := range p.Source.EligibleStatuses {
		eligible[strings.ToLower(st)] = true
	}
	cursor := ""
	for page := 0; page < 3 && len(res.Items) < previewLimit; page++ {
		pg, err := tr.ListItems(ctx, p.Source.Container, integrations.ListOptions{Label: p.Source.EligibilityMarker, Cursor: cursor, Limit: 50})
		if err != nil {
			add("items", "Itens elegíveis listados", StepFailed, failure(err))
			return res
		}
		for _, it := range pg.Items {
			if isDone(it) || (len(eligible) > 0 && !eligible[strings.ToLower(it.Status)]) {
				continue
			}
			if len(res.Items) == previewLimit {
				res.HasMore = true
				break
			}
			pi := PreviewItem{Key: it.Key, Title: it.Title, Status: it.Status, Priority: it.Priority, Subtask: it.Subtask, URL: it.URL}
			if v, ok := p.Source.PriorityMap[strings.ToLower(it.Priority)]; ok {
				pi.Mapped = &v
			}
			res.Items = append(res.Items, pi)
		}
		cursor = pg.NextCursor
		if cursor == "" {
			break
		}
	}
	if cursor != "" {
		res.HasMore = true
	}
	unmapped := 0
	for _, it := range res.Items {
		if it.Mapped == nil && len(p.Source.PriorityMap) > 0 {
			unmapped++
		}
	}
	detail := fmt.Sprintf("%d item(ns) elegível(is) no preview.", len(res.Items))
	if unmapped > 0 {
		detail += fmt.Sprintf(" %d sem prioridade no mapa.", unmapped)
	}
	add("items", "Itens elegíveis listados", StepOK, detail)

	if p.Source.Access == AccessReadWrite {
		add("write", "Permissão de escrita", StepSkipped, "Não há como provar escrita sem escrever; será confirmada no primeiro uso (criar subtarefa ou mover estado).")
	}
	res.OK = true
	for _, s := range res.Steps {
		res.OK = res.OK && s.Status != StepFailed
	}
	return res
}

func joinStatusNames(ss []integrations.Status) string {
	n := make([]string, 0, len(ss))
	for _, s := range ss {
		n = append(n, s.Name)
	}
	return strings.Join(n, ", ")
}

// RecordSourceCheck grava o resultado vinculado à configuração verificada
// (fingerprint) e o registra na trilha. Não altera a versão do projeto.
func RecordSourceCheck(ctx context.Context, db DB, adminID, projectID, fingerprint string, check SourceCheck) error {
	return persistence.WithTx(ctx, db, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `UPDATE projects SET source_check_ok = $2, source_check_fingerprint = $3, source_checked_at = $4 WHERE id = $1`,
			projectID, check.OK, fingerprint, check.CheckedAt); err != nil {
			return persistence.MapError(err)
		}
		return record(ctx, tx, projectID, adminID, ChangeSourceChecked, map[string]any{"ok": check.OK, "items": len(check.Items)})
	})
}
