package integrations

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

var (
	jiraKey       = regexp.MustCompile(`^[A-Z][A-Z0-9_]{1,19}$`)
	jiraIssueRef  = regexp.MustCompile(`^(?:[A-Z][A-Z0-9_]{1,19}-[0-9]{1,9}|[0-9]{1,18})$`)
	jiraLabelText = regexp.MustCompile(`^[A-Za-z0-9_.:-]{1,50}$`)
)

// jiraBlocks é o tipo de vínculo padrão do Jira para dependência técnica.
const jiraBlocks = "Blocks"

type jira struct{ api *apiClient }

func newJira(cfg Config) (Tracker, error) {
	base := cfg.BaseURL
	if base == "" {
		u, err := url.Parse(strings.TrimSpace(cfg.Target))
		if err != nil || u.Host == "" {
			return nil, invalid("jira: destino")
		}
		if !cfg.AllowAnyTarget && (u.Scheme != "https" || !strings.HasSuffix(strings.ToLower(u.Hostname()), ".atlassian.net")) {
			return nil, invalid("jira: destino deve ser https://<site>.atlassian.net")
		}
		base = u.Scheme + "://" + u.Host
	}
	email, token, ok := strings.Cut(cfg.Secret.Reveal(), ":")
	if !ok || email == "" || token == "" {
		return nil, invalid("jira: formato do segredo")
	}
	return &jira{api: &apiClient{
		hc: cfg.Client, base: strings.TrimRight(base, "/") + "/rest/api/3", now: time.Now,
		auth: func(r *http.Request) { r.SetBasicAuth(email, token) },
	}}, nil
}

func (j *jira) Provider() string { return "jira" }

func (j *jira) Identity(ctx context.Context) (Identity, error) {
	var out struct {
		AccountID   string `json:"accountId"`
		DisplayName string `json:"displayName"`
	}
	if e := j.api.do(ctx, "jira: identidade", "GET", "/myself", nil, nil, &out); e != nil {
		return Identity{}, e
	}
	return Identity{Account: out.AccountID, Name: out.DisplayName}, nil
}

func checkKey(op, container string) error {
	if !jiraKey.MatchString(container) {
		return invalid(op)
	}
	return nil
}

func (j *jira) ItemTypes(ctx context.Context, container string) ([]ItemType, error) {
	const op = "jira: tipos de item"
	if err := checkKey(op, container); err != nil {
		return nil, err
	}
	var out struct {
		IssueTypes []struct {
			ID      string `json:"id"`
			Name    string `json:"name"`
			Subtask bool   `json:"subtask"`
		} `json:"issueTypes"`
	}
	if e := j.api.do(ctx, op, "GET", "/project/"+container, nil, nil, &out); e != nil {
		return nil, e
	}
	types := make([]ItemType, 0, len(out.IssueTypes))
	for _, t := range out.IssueTypes {
		types = append(types, ItemType{ID: t.ID, Name: t.Name, Subtask: t.Subtask})
	}
	return types, nil
}

func jiraCategory(key string) string {
	switch key {
	case "done":
		return CategoryDone
	case "indeterminate":
		return CategoryInProgress
	default:
		return CategoryTodo
	}
}

func (j *jira) Statuses(ctx context.Context, container string) ([]Status, error) {
	const op = "jira: estados"
	if err := checkKey(op, container); err != nil {
		return nil, err
	}
	var out []struct {
		Statuses []struct {
			Name     string `json:"name"`
			Category struct {
				Key string `json:"key"`
			} `json:"statusCategory"`
		} `json:"statuses"`
	}
	if e := j.api.do(ctx, op, "GET", "/project/"+container+"/statuses", nil, nil, &out); e != nil {
		return nil, e
	}
	seen := map[string]bool{}
	statuses := []Status{}
	for _, t := range out {
		for _, s := range t.Statuses {
			if !seen[s.Name] {
				seen[s.Name] = true
				statuses = append(statuses, Status{Name: s.Name, Category: jiraCategory(s.Category.Key)})
			}
		}
	}
	sort.Slice(statuses, func(a, b int) bool { return statuses[a].Name < statuses[b].Name })
	return statuses, nil
}

func (j *jira) Fields(ctx context.Context, _ string) ([]Field, error) {
	var out []struct {
		ID     string `json:"id"`
		Name   string `json:"name"`
		Custom bool   `json:"custom"`
	}
	if e := j.api.do(ctx, "jira: campos", "GET", "/field", nil, nil, &out); e != nil {
		return nil, e
	}
	fields := make([]Field, 0, len(out))
	for _, f := range out {
		fields = append(fields, Field{ID: f.ID, Name: f.Name, Custom: f.Custom})
	}
	return fields, nil
}

func (j *jira) Priorities(ctx context.Context) ([]string, error) {
	var out []struct {
		Name string `json:"name"`
	}
	if e := j.api.do(ctx, "jira: prioridades", "GET", "/priority", nil, nil, &out); e != nil {
		return nil, e
	}
	names := make([]string, 0, len(out))
	for _, p := range out {
		names = append(names, p.Name)
	}
	return names, nil
}

type jiraIssue struct {
	ID     string `json:"id"`
	Key    string `json:"key"`
	Fields struct {
		Summary     string          `json:"summary"`
		Description json.RawMessage `json:"description"`
		IssueType   struct {
			Name    string `json:"name"`
			Subtask bool   `json:"subtask"`
		} `json:"issuetype"`
		Status struct {
			Name     string `json:"name"`
			Category struct {
				Key string `json:"key"`
			} `json:"statusCategory"`
		} `json:"status"`
		Priority *struct {
			Name string `json:"name"`
		} `json:"priority"`
		Labels []string `json:"labels"`
		Parent *struct {
			ID string `json:"id"`
		} `json:"parent"`
		Updated string `json:"updated"`
		Links   []struct {
			Type struct {
				Name string `json:"name"`
			} `json:"type"`
			Inward *struct {
				ID string `json:"id"`
			} `json:"inwardIssue"`
		} `json:"issuelinks"`
	} `json:"fields"`
}

const jiraFields = "summary,description,issuetype,status,priority,labels,parent,updated,issuelinks"

func (j *jira) item(i jiraIssue) Item {
	it := Item{
		Provider: "jira", ID: i.ID, Key: i.Key, Title: i.Fields.Summary,
		Description: adfText(i.Fields.Description),
		Type:        i.Fields.IssueType.Name, Subtask: i.Fields.IssueType.Subtask,
		Status: i.Fields.Status.Name, Category: jiraCategory(i.Fields.Status.Category.Key),
		Labels: append([]string{}, i.Fields.Labels...), Dependencies: []string{},
		Version: i.Fields.Updated,
		URL:     strings.TrimSuffix(j.api.base, "/rest/api/3") + "/browse/" + i.Key,
	}
	if i.Fields.Priority != nil {
		it.Priority = i.Fields.Priority.Name
	}
	if i.Fields.Parent != nil {
		it.ParentID = i.Fields.Parent.ID
	}
	// Em issuelinks, inwardIssue de um vínculo "Blocks" é quem bloqueia este item.
	for _, l := range i.Fields.Links {
		if l.Type.Name == jiraBlocks && l.Inward != nil {
			it.Dependencies = append(it.Dependencies, l.Inward.ID)
		}
	}
	return it
}

func jqlQuote(s string) string {
	return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(s) + `"`
}

func (j *jira) ListItems(ctx context.Context, container string, opts ListOptions) (ItemPage, error) {
	const op = "jira: listar itens"
	if err := checkKey(op, container); err != nil {
		return ItemPage{}, err
	}
	jql := "project = " + container
	if opts.Label != "" {
		if !jiraLabelText.MatchString(opts.Label) {
			return ItemPage{}, invalid(op)
		}
		jql += " AND labels = " + jqlQuote(opts.Label)
	}
	if !opts.IncludeDone {
		jql += " AND statusCategory != Done"
	}
	jql += " ORDER BY created ASC, key ASC"
	q := url.Values{"jql": {jql}, "fields": {jiraFields}, "maxResults": {strconv.Itoa(pageLimit(opts.Limit))}}
	if opts.Cursor != "" {
		q.Set("nextPageToken", opts.Cursor)
	}
	var out struct {
		Issues        []jiraIssue `json:"issues"`
		NextPageToken string      `json:"nextPageToken"`
		IsLast        bool        `json:"isLast"`
	}
	if e := j.api.do(ctx, op, "GET", "/search/jql", q, nil, &out); e != nil {
		return ItemPage{}, e
	}
	page := ItemPage{Items: make([]Item, 0, len(out.Issues))}
	for _, i := range out.Issues {
		page.Items = append(page.Items, j.item(i))
	}
	if !out.IsLast {
		page.NextCursor = out.NextPageToken
	}
	return page, nil
}

func checkIssue(op, id string) error {
	if !jiraIssueRef.MatchString(id) {
		return invalid(op)
	}
	return nil
}

func (j *jira) GetItem(ctx context.Context, id string) (Item, error) {
	const op = "jira: ler item"
	if err := checkIssue(op, id); err != nil {
		return Item{}, err
	}
	var out jiraIssue
	if e := j.api.do(ctx, op, "GET", "/issue/"+id, url.Values{"fields": {jiraFields}}, nil, &out); e != nil {
		return Item{}, e
	}
	return j.item(out), nil
}

func (j *jira) Transitions(ctx context.Context, itemID string) ([]Transition, error) {
	const op = "jira: transições"
	if err := checkIssue(op, itemID); err != nil {
		return nil, err
	}
	var out struct {
		Transitions []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
			To   struct {
				Name     string `json:"name"`
				Category struct {
					Key string `json:"key"`
				} `json:"statusCategory"`
			} `json:"to"`
		} `json:"transitions"`
	}
	if e := j.api.do(ctx, op, "GET", "/issue/"+itemID+"/transitions", nil, nil, &out); e != nil {
		return nil, e
	}
	ts := make([]Transition, 0, len(out.Transitions))
	for _, t := range out.Transitions {
		ts = append(ts, Transition{ID: t.ID, Name: t.Name, ToStatus: t.To.Name, Category: jiraCategory(t.To.Category.Key)})
	}
	return ts, nil
}

func (j *jira) Transition(ctx context.Context, itemID, transitionID string) error {
	const op = "jira: aplicar transição"
	valid, err := j.Transitions(ctx, itemID)
	if err != nil {
		return err
	}
	for _, t := range valid {
		if t.ID == transitionID {
			body := map[string]any{"transition": map[string]string{"id": transitionID}}
			return asError(j.api.do(ctx, op, "POST", "/issue/"+itemID+"/transitions", nil, body, nil))
		}
	}
	return invalid(op) // transição indisponível no estado atual: nada é enviado
}

func adfDoc(text string) map[string]any {
	return map[string]any{"type": "doc", "version": 1, "content": []any{
		map[string]any{"type": "paragraph", "content": []any{map[string]any{"type": "text", "text": text}}},
	}}
}

func (j *jira) CreateSubtask(ctx context.Context, parentID string, in SubtaskInput) (Item, error) {
	const op = "jira: criar subtarefa"
	if err := checkIssue(op, parentID); err != nil {
		return Item{}, err
	}
	title := strings.TrimSpace(in.Title)
	if title == "" || len(title) > 255 {
		return Item{}, invalid(op)
	}
	parent, err := j.GetItem(ctx, parentID)
	if err != nil {
		return Item{}, err
	}
	if parent.Subtask {
		return Item{}, invalid(op) // subtarefa não tem subtarefa
	}
	project, _, _ := strings.Cut(parent.Key, "-")
	types, err := j.ItemTypes(ctx, project)
	if err != nil {
		return Item{}, err
	}
	typeID := ""
	for _, t := range types {
		if t.Subtask {
			typeID = t.ID
			break
		}
	}
	if typeID == "" {
		return Item{}, invalid(op) // o projeto não tem tipo de subtarefa
	}
	fields := map[string]any{
		"project": map[string]string{"key": project}, "parent": map[string]string{"key": parent.Key},
		"issuetype": map[string]string{"id": typeID}, "summary": title,
	}
	if d := strings.TrimSpace(in.Description); d != "" {
		fields["description"] = adfDoc(d)
	}
	var created struct {
		Key string `json:"key"`
	}
	if e := j.api.do(ctx, op, "POST", "/issue", nil, map[string]any{"fields": fields}, &created); e != nil {
		return Item{}, e
	}
	return j.GetItem(ctx, created.Key)
}

func (j *jira) AddDependency(ctx context.Context, itemID, dependsOnID string) error {
	const op = "jira: criar dependência"
	if err := checkIssue(op, itemID); err != nil {
		return err
	}
	if err := checkIssue(op, dependsOnID); err != nil {
		return err
	}
	// "Blocks": outwardIssue bloqueia inwardIssue; itemID depende de dependsOnID.
	body := map[string]any{
		"type":         map[string]string{"name": jiraBlocks},
		"inwardIssue":  map[string]string{"key": itemID},
		"outwardIssue": map[string]string{"key": dependsOnID},
	}
	return asError(j.api.do(ctx, op, "POST", "/issueLink", nil, body, nil))
}

func (j *jira) AddLink(ctx context.Context, itemID, linkURL, title string) error {
	const op = "jira: anexar link"
	if err := checkIssue(op, itemID); err != nil {
		return err
	}
	u, err := url.Parse(linkURL)
	if err != nil || u.Scheme != "https" || u.Host == "" {
		return invalid(op)
	}
	// globalId estável: repetir a chamada atualiza o mesmo link em vez de duplicar.
	body := map[string]any{"globalId": "lf-link:" + linkURL, "object": map[string]string{"url": linkURL, "title": clipText(title, 200)}}
	return asError(j.api.do(ctx, op, "POST", "/issue/"+itemID+"/remotelink", nil, body, nil))
}

// adfText extrai o texto simples de um documento Atlassian (ADF).
func adfText(raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}
	var node struct {
		Type    string            `json:"type"`
		Text    string            `json:"text"`
		Content []json.RawMessage `json:"content"`
	}
	if json.Unmarshal(raw, &node) != nil {
		return ""
	}
	var b strings.Builder
	b.WriteString(node.Text)
	for i, c := range node.Content {
		part := adfText(c)
		b.WriteString(part)
		if i < len(node.Content)-1 && (node.Type == "doc" || node.Type == "bulletList" || node.Type == "orderedList") {
			b.WriteString("\n")
		}
	}
	return b.String()
}
