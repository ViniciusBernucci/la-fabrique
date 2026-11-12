package projectres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
	"github.com/ViniciusBernucci/le-fabrique/internal/security"
)

// FieldError é um erro de entrada.
type FieldError struct{ Field, Message string }

func (e *FieldError) Error() string { return e.Field + ": " + e.Message }
func (e *FieldError) Unwrap() error { return persistence.ErrInvalidInput }

// DB é o que o pacote precisa do PostgreSQL.
type DB interface {
	persistence.DBTX
	persistence.TxStarter
}

// Fontes de um comando.
const SourceManual = "manual"

// Comandos reconhecidos do perfil.
var commandNames = map[string]bool{"install": true, "build": true, "test": true, "lint": true, "start": true}

// Command é um comando do perfil com a origem do valor.
type Command struct {
	Value  string `json:"value"`
	Source string `json:"source"` // manifesto (ex.: package.json#scripts.test) ou manual
}

// Profile é o perfil técnico do projeto.
type Profile struct {
	ProjectID    string             `json:"project_id"`
	Commands     map[string]Command `json:"commands"`
	Services     []string           `json:"services"`
	SecretRefs   map[string]string  `json:"secret_refs"` // NOME → connection:<id>
	Discovery    *Discovery         `json:"discovery"`
	DiscoveredAt *time.Time         `json:"discovered_at"`
	Version      int                `json:"version"`
}

// ProfileInput é o que o responsável escolhe.
type ProfileInput struct {
	Commands   map[string]string `json:"commands"`
	Services   []string          `json:"services"`
	SecretRefs map[string]string `json:"secret_refs"`
}

var (
	refPattern  = regexp.MustCompile(`^connection:([0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})$`)
	namePattern = regexp.MustCompile(`^[A-Z][A-Z0-9_]{1,63}$`)
)

const maxCommandLen = 500

func emptyProfile(projectID string) Profile {
	return Profile{ProjectID: projectID, Commands: map[string]Command{}, Services: []string{}, SecretRefs: map[string]string{}, Version: 0}
}

// GetProfile devolve o perfil (vazio se nunca foi configurado).
func GetProfile(ctx context.Context, db persistence.DBTX, projectID string) (Profile, error) {
	var p Profile
	var cmds, svcs, refs, disc []byte
	err := db.QueryRow(ctx, `SELECT project_id, commands, services, secret_refs, discovery, discovered_at, version FROM project_profiles WHERE project_id = $1`, projectID).
		Scan(&p.ProjectID, &cmds, &svcs, &refs, &disc, &p.DiscoveredAt, &p.Version)
	if errors.Is(persistence.MapError(err), persistence.ErrNotFound) {
		var one int
		if e := db.QueryRow(ctx, `SELECT 1 FROM projects WHERE id = $1`, projectID).Scan(&one); e != nil {
			return Profile{}, persistence.MapError(e)
		}
		return emptyProfile(projectID), nil
	}
	if err != nil {
		return Profile{}, persistence.MapError(err)
	}
	p.Commands, p.Services, p.SecretRefs = map[string]Command{}, []string{}, map[string]string{}
	_ = json.Unmarshal(cmds, &p.Commands)
	_ = json.Unmarshal(svcs, &p.Services)
	_ = json.Unmarshal(refs, &p.SecretRefs)
	if len(disc) > 0 && string(disc) != "null" {
		var d Discovery
		if json.Unmarshal(disc, &d) == nil {
			p.Discovery = &d
		}
	}
	return p, nil
}

// SaveDiscovery grava o resultado da inspeção (só sugestões; não altera o perfil).
func SaveDiscovery(ctx context.Context, db persistence.DBTX, projectID string, d Discovery, now time.Time) error {
	b, _ := json.Marshal(d)
	_, err := db.Exec(ctx, `INSERT INTO project_profiles (project_id, discovery, discovered_at, updated_at) VALUES ($1, $2::jsonb, $3, $3)
		ON CONFLICT (project_id) DO UPDATE SET discovery = $2::jsonb, discovered_at = $3, updated_at = $3`, projectID, string(b), now)
	return persistence.MapError(err)
}

// SetProfile grava o perfil escolhido. Comandos que coincidem com uma sugestão
// da descoberta guardam a origem dela; os demais são "manual". Segredo só por
// referência de uma conexão ativa; qualquer outra coisa (inclusive um valor) é recusada.
func SetProfile(ctx context.Context, db DB, projectID string, expectedVersion int, in ProfileInput, red *security.Redactor, now time.Time) (Profile, error) {
	cmds := map[string]Command{}
	disc, derr := GetProfile(ctx, db, projectID)
	if derr != nil {
		return Profile{}, derr
	}
	if disc.Version != expectedVersion {
		return Profile{}, persistence.ErrVersionConflict
	}
	suggested := map[string]string{}
	if disc.Discovery != nil {
		for _, s := range disc.Discovery.Commands {
			suggested[s.Name+"\x00"+s.Value] = s.Source
		}
	}
	for name, val := range in.Commands {
		val = strings.TrimSpace(val)
		if !commandNames[name] {
			return Profile{}, &FieldError{"commands", fmt.Sprintf("comando desconhecido: %q (use install, build, test, lint ou start)", name)}
		}
		if val == "" {
			continue
		}
		if len(val) > maxCommandLen || strings.ContainsAny(val, "\x00\r\n") {
			return Profile{}, &FieldError{"commands." + name, "comando inválido (uma linha, até 500 caracteres)"}
		}
		if red.Redact(val) != val {
			return Profile{}, &FieldError{"commands." + name, "o comando contém um segredo; use uma referência de segredo"}
		}
		src := SourceManual
		if s, ok := suggested[name+"\x00"+val]; ok {
			src = s
		}
		cmds[name] = Command{Value: val, Source: src}
	}
	services := []string{}
	seen := map[string]bool{}
	for _, s := range in.Services {
		s = strings.TrimSpace(s)
		if s == "" || seen[s] {
			continue
		}
		if !svcName.MatchString(s) {
			return Profile{}, &FieldError{"services", fmt.Sprintf("nome de serviço inválido: %q", s)}
		}
		seen[s] = true
		services = append(services, s)
	}
	sort.Strings(services)
	refs := map[string]string{}
	for name, ref := range in.SecretRefs {
		if !namePattern.MatchString(name) {
			return Profile{}, &FieldError{"secret_refs", fmt.Sprintf("nome de variável inválido: %q (use MAIÚSCULAS_COM_SUBLINHADO)", name)}
		}
		m := refPattern.FindStringSubmatch(strings.TrimSpace(ref))
		if m == nil {
			return Profile{}, &FieldError{"secret_refs." + name, "informe a referência da conexão (connection:<id>), nunca o valor do segredo"}
		}
		var state string
		if err := db.QueryRow(ctx, `SELECT state FROM connections WHERE id = $1`, m[1]).Scan(&state); err != nil {
			if errors.Is(persistence.MapError(err), persistence.ErrNotFound) {
				return Profile{}, &FieldError{"secret_refs." + name, "a conexão não existe"}
			}
			return Profile{}, persistence.MapError(err)
		}
		if state != "active" {
			return Profile{}, &FieldError{"secret_refs." + name, "a conexão está revogada"}
		}
		refs[name] = "connection:" + m[1]
	}
	cj, _ := json.Marshal(cmds)
	sj, _ := json.Marshal(services)
	rj, _ := json.Marshal(refs)
	err := persistence.WithTx(ctx, db, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `
			INSERT INTO project_profiles (project_id, commands, services, secret_refs, version, updated_at)
			VALUES ($1, $2::jsonb, $3::jsonb, $4::jsonb, 1, $5)
			ON CONFLICT (project_id) DO UPDATE SET commands = $2::jsonb, services = $3::jsonb, secret_refs = $4::jsonb,
				version = project_profiles.version + 1, updated_at = $5
			WHERE project_profiles.version = $6`, projectID, string(cj), string(sj), string(rj), now, expectedVersion)
		if err != nil {
			return persistence.MapError(err)
		}
		if tag.RowsAffected() == 0 {
			return persistence.ErrVersionConflict
		}
		return nil
	})
	if err != nil {
		return Profile{}, err
	}
	return GetProfile(ctx, db, projectID)
}
