package skills

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
	"github.com/ViniciusBernucci/le-fabrique/internal/policy"
)

// defaultsFS guarda as skills padrão da fábrica (ADR-0066): uma pasta por skill, com SKILL.md e as
// subpastas (references/, assets/…) exatamente como num pacote importado. "all:" inclui arquivos que
// começam com "." ou "_".
//
//go:embed all:defaults
var defaultsFS embed.FS

// DefaultRefPrefix marca, no source_ref, a skill que veio do pacote embutido e ainda não foi trocada
// por ninguém. É o que autoriza a fábrica a atualizá-la numa versão nova.
const DefaultRefPrefix = "padrao:"

// InstallReport resume uma instalação das skills padrão.
type InstallReport struct {
	Installed []string // novas no catálogo (e associadas a todos os projetos)
	Updated   []string // padrão ainda intocada, atualizada para a versão embutida
	Unchanged []string // já estava igual
	Kept      []string // o catálogo tem outra versão (editada ou importada): mantida
}

// DefaultNames lista as skills padrão embutidas.
func DefaultNames() []string {
	ents, _ := fs.ReadDir(defaultsFS, "defaults")
	var out []string
	for _, e := range ents {
		if e.IsDir() {
			out = append(out, e.Name())
		}
	}
	return out
}

// InstallDefaults instala as skills padrão embutidas. Idempotente: pode rodar a cada subida.
func InstallDefaults(ctx context.Context, db DB, pol *policy.Policy) (InstallReport, error) {
	sub, err := fs.Sub(defaultsFS, "defaults")
	if err != nil {
		return InstallReport{}, err
	}
	return InstallFrom(ctx, db, pol, sub)
}

// InstallFrom instala as skills padrão de fsys (uma pasta por skill). Regras:
//   - skill nova: entra no catálogo, vira padrão e é associada a TODOS os projetos existentes (os novos
//     a recebem na criação);
//   - padrão ainda intocada (source_ref "padrao:…") com conteúdo diferente: atualiza para a embutida;
//   - qualquer outra versão com o mesmo nome (editada ou importada por alguém): mantida, sem tocar.
//
// A associação só acontece na instalação: quem tirar uma skill padrão de um projeto não a vê voltar.
func InstallFrom(ctx context.Context, db DB, pol *policy.Policy, fsys fs.FS) (InstallReport, error) {
	var rep InstallReport
	ents, err := fs.ReadDir(fsys, ".")
	if err != nil {
		return rep, err
	}
	for _, e := range ents {
		if !e.IsDir() {
			continue
		}
		files, err := readTree(fsys, e.Name())
		if err != nil {
			return rep, fmt.Errorf("skill padrão %s: %w", e.Name(), err)
		}
		pkg, err := Ingest(files, pol)
		if err != nil {
			return rep, fmt.Errorf("skill padrão %s inválida: %w", e.Name(), err)
		}
		if pkg.Manifest.Name != e.Name() {
			return rep, fmt.Errorf("skill padrão %s: o nome no SKILL.md (%s) deve ser igual ao da pasta", e.Name(), pkg.Manifest.Name)
		}
		ref := DefaultRefPrefix + pkg.Manifest.Name
		cur, err := GetByName(ctx, db, pkg.Manifest.Name)
		switch {
		case errors.Is(err, persistence.ErrNotFound):
			sk, _, err := Save(ctx, db, pkg, "package", ref, "", false)
			if err != nil {
				return rep, err
			}
			if err := markDefault(ctx, db, sk.ID); err != nil {
				return rep, err
			}
			rep.Installed = append(rep.Installed, sk.Name)
		case err != nil:
			return rep, err
		case cur.ContentHash == pkg.Hash:
			// Mesmo conteúdo já no catálogo (por exemplo, importado à mão do mesmo pacote): é o padrão
			// intocado. Passa a valer como padrão e a origem é normalizada, para receber as versões novas.
			if !cur.IsDefault {
				if err := markDefault(ctx, db, cur.ID); err != nil {
					return rep, err
				}
			}
			if cur.SourceKind != "package" || cur.SourceRef != ref {
				if _, err := db.Exec(ctx, `UPDATE skills SET source_kind = 'package', source_ref = $2 WHERE id = $1`, cur.ID, ref); err != nil {
					return rep, persistence.MapError(err)
				}
			}
			rep.Unchanged = append(rep.Unchanged, cur.Name)
		case cur.IsDefault && cur.SourceRef == ref:
			if _, _, err := Save(ctx, db, pkg, "package", ref, "", true); err != nil {
				return rep, err
			}
			rep.Updated = append(rep.Updated, cur.Name)
		default:
			rep.Kept = append(rep.Kept, cur.Name)
		}
	}
	return rep, nil
}

// markDefault marca a skill como padrão e a associa a todos os projetos, numa única transação.
func markDefault(ctx context.Context, db DB, skillID string) error {
	return persistence.WithTx(ctx, db, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `UPDATE skills SET is_default = true WHERE id = $1`, skillID); err != nil {
			return persistence.MapError(err)
		}
		_, err := tx.Exec(ctx, `INSERT INTO project_resources (project_id, kind, ref) SELECT id, 'skill', $1 FROM projects ON CONFLICT DO NOTHING`, skillID)
		return persistence.MapError(err)
	})
}

// readTree lê os arquivos de uma pasta de skill (caminhos relativos a ela, em ordem estável).
func readTree(fsys fs.FS, dir string) ([]File, error) {
	var out []File
	err := fs.WalkDir(fsys, dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := fs.ReadFile(fsys, p)
		if err != nil {
			return err
		}
		out = append(out, File{Path: strings.TrimPrefix(p, path.Clean(dir)+"/"), Content: b})
		return nil
	})
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out, err
}
