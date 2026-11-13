package mcp

import (
	"fmt"
	"math"
	"sort"
	"strings"
)

// ValidateArgs confere os argumentos contra o esquema de entrada anunciado pelo
// servidor (subconjunto de JSON Schema: object, properties, required, type,
// enum, tamanhos e limites). Propriedades fora do esquema são recusadas, salvo
// `additionalProperties: true`. Tudo falha ANTES de qualquer chamada remota.
func ValidateArgs(schema map[string]any, args map[string]any) error {
	if t, _ := schema["type"].(string); t != "" && t != "object" {
		return fmt.Errorf("esquema de entrada não é um objeto")
	}
	props, _ := schema["properties"].(map[string]any)
	allowExtra, _ := schema["additionalProperties"].(bool)
	names := make([]string, 0, len(args))
	for k := range args {
		names = append(names, k)
	}
	sort.Strings(names)
	for _, k := range names {
		if _, known := props[k]; !known && !allowExtra {
			return fmt.Errorf("parâmetro %q não existe no esquema da ferramenta", k)
		}
	}
	if req, ok := schema["required"].([]any); ok {
		for _, r := range req {
			name, _ := r.(string)
			if v, present := args[name]; !present || v == nil {
				return fmt.Errorf("parâmetro obrigatório %q ausente", name)
			}
		}
	}
	for _, k := range names {
		ps, _ := props[k].(map[string]any)
		if ps == nil {
			continue
		}
		if err := validateValue(k, ps, args[k]); err != nil {
			return err
		}
	}
	return nil
}

func validateValue(name string, ps map[string]any, v any) error {
	if enum, ok := ps["enum"].([]any); ok {
		found := false
		for _, e := range enum {
			found = found || fmt.Sprint(e) == fmt.Sprint(v)
		}
		if !found {
			return fmt.Errorf("parâmetro %q fora dos valores permitidos", name)
		}
	}
	t, _ := ps["type"].(string)
	switch t {
	case "string":
		s, ok := v.(string)
		if !ok {
			return fmt.Errorf("parâmetro %q deve ser texto", name)
		}
		if n, ok := ps["maxLength"].(float64); ok && float64(len(s)) > n {
			return fmt.Errorf("parâmetro %q acima do tamanho máximo", name)
		}
		if n, ok := ps["minLength"].(float64); ok && float64(len(s)) < n {
			return fmt.Errorf("parâmetro %q abaixo do tamanho mínimo", name)
		}
	case "integer", "number":
		f, ok := toFloat(v)
		if !ok || (t == "integer" && f != math.Trunc(f)) {
			return fmt.Errorf("parâmetro %q deve ser %s", name, map[string]string{"integer": "inteiro", "number": "numérico"}[t])
		}
		if n, ok := ps["minimum"].(float64); ok && f < n {
			return fmt.Errorf("parâmetro %q abaixo do mínimo", name)
		}
		if n, ok := ps["maximum"].(float64); ok && f > n {
			return fmt.Errorf("parâmetro %q acima do máximo", name)
		}
	case "boolean":
		if _, ok := v.(bool); !ok {
			return fmt.Errorf("parâmetro %q deve ser verdadeiro ou falso", name)
		}
	case "array":
		a, ok := v.([]any)
		if !ok {
			return fmt.Errorf("parâmetro %q deve ser uma lista", name)
		}
		if n, ok := ps["maxItems"].(float64); ok && float64(len(a)) > n {
			return fmt.Errorf("parâmetro %q com itens demais", name)
		}
	case "object":
		if _, ok := v.(map[string]any); !ok {
			return fmt.Errorf("parâmetro %q deve ser um objeto", name)
		}
	}
	return nil
}

func toFloat(v any) (float64, bool) {
	switch x := v.(type) {
	case float64:
		return x, true
	case int:
		return float64(x), true
	case int64:
		return float64(x), true
	}
	return 0, false
}

// SchemaProblems devolve, para exibição, o que o esquema declara de obrigatório.
func SchemaProblems(schema map[string]any) string {
	var req []string
	if r, ok := schema["required"].([]any); ok {
		for _, x := range r {
			if s, ok := x.(string); ok {
				req = append(req, s)
			}
		}
	}
	return strings.Join(req, ", ")
}
