// Package fixtures contém dados reproduzíveis para desenvolvimento e testes.
// Os projetos abaixo são fictícios e existem apenas para exercitar o painel.
package fixtures

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

//go:embed projects.json
var projectsJSON []byte

// Project é um projeto de exemplo.
type Project struct {
	Slug     string `json:"slug"`
	Name     string `json:"name"`
	State    string `json:"state"`
	Priority int    `json:"priority"`
}

// Projects devolve os projetos de exemplo.
func Projects() ([]Project, error) {
	var out []Project
	if err := json.Unmarshal(projectsJSON, &out); err != nil {
		return nil, fmt.Errorf("fixtures/projects.json inválido: %w", err)
	}
	return out, nil
}
