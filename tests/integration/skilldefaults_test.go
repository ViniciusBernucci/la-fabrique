package integration

import (
	"testing"
	"testing/fstest"

	"github.com/ViniciusBernucci/le-fabrique/internal/policy"
	"github.com/ViniciusBernucci/le-fabrique/internal/projects"
	"github.com/ViniciusBernucci/le-fabrique/internal/skills"
)

func skillResources(t *testing.T, db testDB, projectID string) int {
	t.Helper()
	var n int
	if err := db.Pool.QueryRow(ctx, `SELECT count(*) FROM project_resources WHERE project_id = $1 AND kind = 'skill'`, projectID).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func newProjectDirect(t *testing.T, db testDB, slug string) projects.Project {
	t.Helper()
	p, err := projects.Create(ctx, db.Pool, projects.NewProject{Slug: slug, Name: "Projeto " + slug})
	if err != nil {
		t.Fatal(err)
	}
	return p
}

// ADR-0066 — as skills padrão chegam ao catálogo e a TODOS os projetos, sem mexer no que a pessoa mudou.
func TestDefaultSkillsAreInstalledAndAssociatedToEveryProject(t *testing.T) {
	db := newMigratedDB(t)
	pol := policy.DefaultPolicy(policy.Options{})
	before := newProjectDirect(t, db, "ja-existia")
	if n := skillResources(t, db, before.ID); n != 0 {
		t.Fatalf("sem skills padrão instaladas o projeto nasce sem associações: %d", n)
	}

	rep, err := skills.InstallDefaults(ctx, db.Pool, pol)
	if err != nil || len(rep.Installed) != 6 || len(rep.Updated)+len(rep.Kept)+len(rep.Unchanged) != 0 {
		t.Fatalf("primeira instalação: %+v %v", rep, err)
	}
	all, err := skills.List(ctx, db.Pool)
	if err != nil || len(all) != 6 {
		t.Fatalf("catálogo: %d %v", len(all), err)
	}
	for _, s := range all {
		if !s.IsDefault || s.SourceKind != "package" || s.SourceRef != skills.DefaultRefPrefix+s.Name || s.Executor != "ai" {
			t.Errorf("skill padrão mal gravada: %+v", s)
		}
		if f, err := skills.Files(ctx, db.Pool, s.ID); err != nil || len(f) < 2 {
			t.Errorf("%s deveria guardar SKILL.md e references/: %v %v", s.Name, f, err)
		}
	}
	// o projeto que já existia recebeu as 6
	if n := skillResources(t, db, before.ID); n != 6 {
		t.Fatalf("projeto existente: %d associações", n)
	}
	// um projeto criado depois já nasce com elas
	after := newProjectDirect(t, db, "criado-depois")
	if n := skillResources(t, db, after.ID); n != 6 {
		t.Fatalf("projeto novo: %d associações", n)
	}

	// segunda subida: nada muda
	rep, err = skills.InstallDefaults(ctx, db.Pool, pol)
	if err != nil || len(rep.Unchanged) != 6 || len(rep.Installed)+len(rep.Updated)+len(rep.Kept) != 0 {
		t.Fatalf("idempotência: %+v %v", rep, err)
	}
	// quem tirou uma skill padrão de um projeto não a vê voltar na próxima subida
	if _, err := db.Pool.Exec(ctx, `DELETE FROM project_resources WHERE project_id = $1 AND kind = 'skill' AND ref = $2`, before.ID, all[0].ID); err != nil {
		t.Fatal(err)
	}
	if _, err := skills.InstallDefaults(ctx, db.Pool, pol); err != nil {
		t.Fatal(err)
	}
	if n := skillResources(t, db, before.ID); n != 5 {
		t.Fatalf("a associação removida não deve voltar: %d", n)
	}
}

func pack(name, body string) fstest.MapFS {
	return fstest.MapFS{
		name + "/SKILL.md":           {Data: []byte("---\nname: " + name + "\ndescription: Skill de teste.\n---\n" + body)},
		name + "/references/guia.md": {Data: []byte("guia")},
	}
}

func TestDefaultSkillUpdatesOnlyWhenUntouched(t *testing.T) {
	db := newMigratedDB(t)
	pol := policy.DefaultPolicy(policy.Options{})
	p := newProjectDirect(t, db, "p1")

	if rep, err := skills.InstallFrom(ctx, db.Pool, pol, pack("lf-demo", "versão 1")); err != nil || len(rep.Installed) != 1 {
		t.Fatalf("%+v %v", rep, err)
	}
	// versão nova embutida e skill intocada: atualiza (nova versão) e não mexe nas associações
	rep, err := skills.InstallFrom(ctx, db.Pool, pol, pack("lf-demo", "versão 2"))
	if err != nil || len(rep.Updated) != 1 {
		t.Fatalf("atualização: %+v %v", rep, err)
	}
	cur, _ := skills.GetByName(ctx, db.Pool, "lf-demo")
	if cur.Version != 2 || cur.Body != "versão 2" || !cur.IsDefault {
		t.Fatalf("%+v", cur)
	}
	if n := skillResources(t, db, p.ID); n != 1 {
		t.Fatalf("associações: %d", n)
	}
	// alguém substitui a skill pela tela (outra origem): a versão dele prevalece nas próximas subidas
	pkg, err := skills.Ingest([]skills.File{{Path: "SKILL.md", Content: []byte("---\nname: lf-demo\ndescription: Minha versão.\n---\nversão do usuário")}}, pol)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := skills.Save(ctx, db.Pool, pkg, "file", "lf-demo.zip", "", true); err != nil {
		t.Fatal(err)
	}
	rep, err = skills.InstallFrom(ctx, db.Pool, pol, pack("lf-demo", "versão 3"))
	if err != nil || len(rep.Kept) != 1 || len(rep.Updated) != 0 {
		t.Fatalf("a versão do usuário deve ser mantida: %+v %v", rep, err)
	}
	cur, _ = skills.GetByName(ctx, db.Pool, "lf-demo")
	if cur.Body != "versão do usuário" || cur.SourceKind != "file" {
		t.Fatalf("não pode sobrescrever: %+v", cur)
	}
}

func TestUserSkillWithSameNameIsNeverTakenOver(t *testing.T) {
	db := newMigratedDB(t)
	pol := policy.DefaultPolicy(policy.Options{})
	p := newProjectDirect(t, db, "p2")
	pkg, err := skills.Ingest([]skills.File{{Path: "SKILL.md", Content: []byte("---\nname: lf-demo\ndescription: Do usuário.\n---\ncorpo do usuário")}}, pol)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := skills.Save(ctx, db.Pool, pkg, "manual", "", "", false); err != nil {
		t.Fatal(err)
	}
	rep, err := skills.InstallFrom(ctx, db.Pool, pol, pack("lf-demo", "padrão"))
	if err != nil || len(rep.Kept) != 1 || len(rep.Installed) != 0 {
		t.Fatalf("%+v %v", rep, err)
	}
	cur, _ := skills.GetByName(ctx, db.Pool, "lf-demo")
	if cur.IsDefault || cur.Body != "corpo do usuário" {
		t.Fatalf("a skill do usuário não vira padrão nem é trocada: %+v", cur)
	}
	if n := skillResources(t, db, p.ID); n != 0 {
		t.Fatalf("e não é associada aos projetos: %d", n)
	}
}

func TestDefaultSkillsAppearInTheAPICatalog(t *testing.T) {
	e := newAuthEnv(t, nil)
	if _, err := skills.InstallDefaults(ctx, e.db.Pool, policy.DefaultPolicy(policy.Options{})); err != nil {
		t.Fatal(err)
	}
	c := e.newClient()
	c.login(nil)
	var list struct{ Items []skills.Skill }
	c.expect(c.get("/api/skills"), 200).JSON(&list)
	if len(list.Items) != 6 {
		t.Fatalf("skills: %d", len(list.Items))
	}
	for _, s := range list.Items {
		if !s.IsDefault {
			t.Errorf("%s deveria vir marcada como padrão", s.Name)
		}
	}
	// o projeto criado pela API já tem as skills associadas
	p := createProject(t, c, "via-api")
	var res struct {
		Resources []struct{ Kind, Ref, Name string }
	}
	c.expect(c.get("/api/projects/"+p.ID+"/resources"), 200).JSON(&res)
	n := 0
	for _, r := range res.Resources {
		if r.Kind == "skill" {
			n++
		}
	}
	if n != 6 {
		t.Fatalf("recursos do projeto: %+v", res.Resources)
	}
}

// Quem já tinha importado o mesmo pacote à mão não perde nada: a skill vira padrão, é associada aos
// projetos e passa a receber as versões novas (a origem é normalizada porque o conteúdo é idêntico).
func TestSkillImportedByHandWithIdenticalContentBecomesTheDefault(t *testing.T) {
	db := newMigratedDB(t)
	pol := policy.DefaultPolicy(policy.Options{})
	p := newProjectDirect(t, db, "p3")
	fsys := pack("lf-demo", "versão 1")
	files := []skills.File{
		{Path: "SKILL.md", Content: fsys["lf-demo/SKILL.md"].Data},
		{Path: "references/guia.md", Content: fsys["lf-demo/references/guia.md"].Data},
	}
	pkg, err := skills.Ingest(files, pol)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := skills.Save(ctx, db.Pool, pkg, "file", "lf-demo.zip", "", false); err != nil {
		t.Fatal(err)
	}
	rep, err := skills.InstallFrom(ctx, db.Pool, pol, fsys)
	if err != nil || len(rep.Unchanged) != 1 {
		t.Fatalf("%+v %v", rep, err)
	}
	cur, _ := skills.GetByName(ctx, db.Pool, "lf-demo")
	if !cur.IsDefault || cur.SourceKind != "package" || cur.SourceRef != skills.DefaultRefPrefix+"lf-demo" || cur.Version != 1 {
		t.Fatalf("%+v", cur)
	}
	if n := skillResources(t, db, p.ID); n != 1 {
		t.Fatalf("associações: %d", n)
	}
	// e agora recebe a versão nova
	if rep, err := skills.InstallFrom(ctx, db.Pool, pol, pack("lf-demo", "versão 2")); err != nil || len(rep.Updated) != 1 {
		t.Fatalf("%+v %v", rep, err)
	}
}
