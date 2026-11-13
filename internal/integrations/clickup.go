package integrations

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	clickupID    = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)
	clickupLabel = regexp.MustCompile(`^[^\x00-\x1f]{1,50}$`)
)

type clickup struct{ api *apiClient }

func newClickUp(cfg Config) (Tracker, error) {
	base := cfg.BaseURL
	if base == "" {
		base = "https://api.clickup.com/api/v2"
	}
	token := strings.TrimSpace(cfg.Secret.Reveal())
	if token == "" {
		return nil, invalid("clickup: segredo")
	}
	return &clickup{api: &apiClient{
		hc: cfg.Client, base: strings.TrimRight(base, "/"), now: time.Now, reset: "X-RateLimit-Reset",
		auth: func(r *http.Request) { r.Header.Set("Authorization", token) },
	}}, nil
}

func (c *clickup) Provider() string { return "clickup" }

func checkID(op, id string) error {
	if !clickupID.MatchString(id) {
		return invalid(op)
	}
	return nil
}

func (c *clickup) Identity(ctx context.Context) (Identity, error) {
	var out struct {
		User struct {
			ID       json.Number `json:"id"`
			Username string      `json:"username"`
		} `json:"user"`
	}
	if e := c.api.do(ctx, "clickup: identidade", "GET", "/user", nil, nil, &out); e != nil {
		return Identity{}, e
	}
	return Identity{Account: out.User.ID.String(), Name: out.User.Username}, nil
}

// ClickUp não tem tipos de item: tarefas e subtarefas são o mesmo objeto.
func (c *clickup) ItemTypes(context.Context, string) ([]ItemType, error) { return []ItemType{}, nil }

func clickupCategory(statusType string) string {
	switch statusType {
	case "closed", "done":
		return CategoryDone
	case "open":
		return CategoryTodo
	default:
		return CategoryInProgress
	}
}

func (c *clickup) list(ctx context.Context, op, container string) (clickupList, error) {
	var out clickupList
	if err := checkID(op, container); err != nil {
		return out, err
	}
	if e := c.api.do(ctx, op, "GET", "/list/"+container, nil, nil, &out); e != nil {
		return out, e
	}
	return out, nil
}

type clickupList struct {
	Statuses []struct {
		Status string `json:"status"`
		Type   string `json:"type"`
	} `json:"statuses"`
}

func (c *clickup) Statuses(ctx context.Context, container string) ([]Status, error) {
	l, err := c.list(ctx, "clickup: estados", container)
	if err != nil {
		return nil, err
	}
	out := make([]Status, 0, len(l.Statuses))
	for _, s := range l.Statuses {
		out = append(out, Status{Name: s.Status, Category: clickupCategory(s.Type)})
	}
	return out, nil
}

func (c *clickup) Fields(ctx context.Context, container string) ([]Field, error) {
	const op = "clickup: campos"
	if err := checkID(op, container); err != nil {
		return nil, err
	}
	var out struct {
		Fields []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"fields"`
	}
	if e := c.api.do(ctx, op, "GET", "/list/"+container+"/field", nil, nil, &out); e != nil {
		return nil, e
	}
	fields := make([]Field, 0, len(out.Fields))
	for _, f := range out.Fields {
		fields = append(fields, Field{ID: f.ID, Name: f.Name, Custom: true})
	}
	return fields, nil
}

// Prioridades fixas do ClickUp (1 urgente … 4 baixa).
func (c *clickup) Priorities(context.Context) ([]string, error) {
	return []string{"urgent", "high", "normal", "low"}, nil
}

type clickupTask struct {
	ID          string `json:"id"`
	CustomID    string `json:"custom_id"`
	Name        string `json:"name"`
	TextContent string `json:"text_content"`
	Status      struct {
		Status string `json:"status"`
		Type   string `json:"type"`
	} `json:"status"`
	Priority *struct {
		Priority string `json:"priority"`
	} `json:"priority"`
	Parent *string `json:"parent"`
	Tags   []struct {
		Name string `json:"name"`
	} `json:"tags"`
	DateUpdated string `json:"date_updated"`
	URL         string `json:"url"`
	List        struct {
		ID string `json:"id"`
	} `json:"list"`
	Dependencies []struct {
		TaskID    string `json:"task_id"`
		DependsOn string `json:"depends_on"`
	} `json:"dependencies"`
}

func (c *clickup) item(t clickupTask) Item {
	key := t.CustomID
	if key == "" {
		key = t.ID
	}
	it := Item{
		Provider: "clickup", ID: t.ID, Key: key, Title: t.Name, Description: t.TextContent,
		Type: "task", Status: t.Status.Status, Category: clickupCategory(t.Status.Type),
		Labels: []string{}, Dependencies: []string{}, Version: t.DateUpdated, URL: t.URL,
	}
	if t.Priority != nil {
		it.Priority = t.Priority.Priority
	}
	if t.Parent != nil && *t.Parent != "" {
		it.Subtask, it.ParentID, it.Type = true, *t.Parent, "subtask"
	}
	for _, tag := range t.Tags {
		it.Labels = append(it.Labels, tag.Name)
	}
	for _, d := range t.Dependencies {
		if d.TaskID == t.ID && d.DependsOn != "" { // este item espera depends_on
			it.Dependencies = append(it.Dependencies, d.DependsOn)
		}
	}
	return it
}

func (c *clickup) ListItems(ctx context.Context, container string, opts ListOptions) (ItemPage, error) {
	const op = "clickup: listar itens"
	if err := checkID(op, container); err != nil {
		return ItemPage{}, err
	}
	page := 0
	if opts.Cursor != "" {
		n, err := strconv.Atoi(opts.Cursor)
		if err != nil || n < 0 {
			return ItemPage{}, invalid(op)
		}
		page = n
	}
	q := url.Values{
		"page": {strconv.Itoa(page)}, "subtasks": {"true"}, "order_by": {"created"},
		"include_closed": {strconv.FormatBool(opts.IncludeDone)},
	}
	if opts.Label != "" {
		if !clickupLabel.MatchString(opts.Label) {
			return ItemPage{}, invalid(op)
		}
		q.Add("tags[]", opts.Label)
	}
	var out struct {
		Tasks    []clickupTask `json:"tasks"`
		LastPage *bool         `json:"last_page"`
	}
	if e := c.api.do(ctx, op, "GET", "/list/"+container+"/task", q, nil, &out); e != nil {
		return ItemPage{}, e
	}
	res := ItemPage{Items: make([]Item, 0, len(out.Tasks))}
	for _, t := range out.Tasks {
		res.Items = append(res.Items, c.item(t))
	}
	// Sem last_page, uma página vazia encerra; com ele, vale o que a API diz.
	if len(out.Tasks) > 0 && (out.LastPage == nil || !*out.LastPage) {
		res.NextCursor = strconv.Itoa(page + 1)
	}
	return res, nil
}

func (c *clickup) task(ctx context.Context, op, id string) (clickupTask, error) {
	var t clickupTask
	if err := checkID(op, id); err != nil {
		return t, err
	}
	if e := c.api.do(ctx, op, "GET", "/task/"+id, nil, nil, &t); e != nil {
		return t, e
	}
	return t, nil
}

func (c *clickup) GetItem(ctx context.Context, id string) (Item, error) {
	t, err := c.task(ctx, "clickup: ler item", id)
	if err != nil {
		return Item{}, err
	}
	return c.item(t), nil
}

// No ClickUp qualquer estado da lista é alcançável; Transitions devolve os estados
// da lista do item (exceto o atual) e o ID da transição é o próprio nome do estado.
func (c *clickup) Transitions(ctx context.Context, itemID string) ([]Transition, error) {
	t, err := c.task(ctx, "clickup: transições", itemID)
	if err != nil {
		return nil, err
	}
	l, err := c.list(ctx, "clickup: transições", t.List.ID)
	if err != nil {
		return nil, err
	}
	ts := []Transition{}
	for _, s := range l.Statuses {
		if s.Status != t.Status.Status {
			ts = append(ts, Transition{ID: s.Status, Name: s.Status, ToStatus: s.Status, Category: clickupCategory(s.Type)})
		}
	}
	return ts, nil
}

func (c *clickup) Transition(ctx context.Context, itemID, transitionID string) error {
	const op = "clickup: aplicar transição"
	valid, err := c.Transitions(ctx, itemID)
	if err != nil {
		return err
	}
	for _, t := range valid {
		if t.ID == transitionID {
			return asError(c.api.do(ctx, op, "PUT", "/task/"+itemID, nil, map[string]string{"status": transitionID}, nil))
		}
	}
	return invalid(op)
}

func (c *clickup) CreateSubtask(ctx context.Context, parentID string, in SubtaskInput) (Item, error) {
	const op = "clickup: criar subtarefa"
	title := strings.TrimSpace(in.Title)
	if title == "" || len(title) > 255 {
		return Item{}, invalid(op)
	}
	parent, err := c.task(ctx, op, parentID)
	if err != nil {
		return Item{}, err
	}
	if parent.Parent != nil && *parent.Parent != "" {
		return Item{}, invalid(op) // mantém a hierarquia de dois níveis, como no Jira
	}
	body := map[string]any{"name": title, "parent": parent.ID}
	if d := strings.TrimSpace(in.Description); d != "" {
		body["description"] = d
	}
	var created clickupTask
	if e := c.api.do(ctx, op, "POST", "/list/"+parent.List.ID+"/task", nil, body, &created); e != nil {
		return Item{}, e
	}
	return c.item(created), nil
}

func (c *clickup) AddDependency(ctx context.Context, itemID, dependsOnID string) error {
	const op = "clickup: criar dependência"
	if err := checkID(op, itemID); err != nil {
		return err
	}
	if err := checkID(op, dependsOnID); err != nil {
		return err
	}
	return asError(c.api.do(ctx, op, "POST", "/task/"+itemID+"/dependency", nil, map[string]string{"depends_on": dependsOnID}, nil))
}

// O ClickUp não tem links remotos nativos: o link entra como comentário. Repetir
// cria outro comentário; quem repete deve conferir antes (ledger, T13-04).
func (c *clickup) AddLink(ctx context.Context, itemID, linkURL, title string) error {
	const op = "clickup: anexar link"
	if err := checkID(op, itemID); err != nil {
		return err
	}
	u, err := url.Parse(linkURL)
	if err != nil || u.Scheme != "https" || u.Host == "" {
		return invalid(op)
	}
	text := strings.TrimSpace(clipText(title, 200)) + " " + linkURL
	return asError(c.api.do(ctx, op, "POST", "/task/"+itemID+"/comment", nil, map[string]string{"comment_text": strings.TrimSpace(text)}, nil))
}

func (c *clickup) Children(ctx context.Context, parentID string) ([]Item, error) {
	const op = "clickup: listar subtarefas"
	if err := checkID(op, parentID); err != nil {
		return nil, err
	}
	var out struct {
		Subtasks []clickupTask `json:"subtasks"`
	}
	if e := c.api.do(ctx, op, "GET", "/task/"+parentID, url.Values{"include_subtasks": {"true"}}, nil, &out); e != nil {
		return nil, e
	}
	items := make([]Item, 0, len(out.Subtasks))
	for _, t := range out.Subtasks {
		items = append(items, c.item(t))
	}
	return items, nil
}

func (c *clickup) HasLink(ctx context.Context, itemID, linkURL string) (bool, error) {
	const op = "clickup: listar links"
	if err := checkID(op, itemID); err != nil {
		return false, err
	}
	var out struct {
		Comments []struct {
			Text string `json:"comment_text"`
		} `json:"comments"`
	}
	if e := c.api.do(ctx, op, "GET", "/task/"+itemID+"/comment", nil, nil, &out); e != nil {
		return false, e
	}
	for _, cm := range out.Comments {
		if strings.Contains(cm.Text, linkURL) {
			return true, nil
		}
	}
	return false, nil
}

// CreateItem cria uma tarefa de primeiro nível na lista (container) já no estado pedido.
func (c *clickup) CreateItem(ctx context.Context, container string, in ItemInput) (Item, error) {
	const op = "clickup: criar tarefa"
	title := strings.TrimSpace(in.Title)
	if title == "" || len(title) > 255 {
		return Item{}, invalid(op)
	}
	if err := checkID(op, container); err != nil {
		return Item{}, err
	}
	body := map[string]any{"name": title}
	if d := strings.TrimSpace(in.Description); d != "" {
		body["description"] = d
	}
	if s := strings.TrimSpace(in.Status); s != "" {
		body["status"] = s
	}
	switch strings.ToLower(strings.TrimSpace(in.Priority)) {
	case "urgent":
		body["priority"] = 1
	case "high":
		body["priority"] = 2
	case "normal":
		body["priority"] = 3
	case "low":
		body["priority"] = 4
	}
	var created clickupTask
	if e := c.api.do(ctx, op, "POST", "/list/"+container+"/task", nil, body, &created); e != nil {
		return Item{}, e
	}
	return c.item(created), nil
}
