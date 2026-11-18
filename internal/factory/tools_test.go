package factory

import (
	"reflect"
	"testing"
)

func TestClaudeToolsMapsCatalogToolsAndIgnoresTheRest(t *testing.T) {
	cases := []struct {
		in   []string
		want []string
	}{
		{nil, nil},
		{[]string{"read_file", "write_file", "edit_file", "search", "list_dir"}, nil}, // já fazem parte do modo de escrita
		{[]string{"git_status", "git_diff"}, []string{"Bash(git diff:*)", "Bash(git status:*)"}},
		{[]string{"run_command", "git_status"}, []string{"Bash"}},
		{[]string{"desconhecida", "run_command"}, []string{"Bash"}},
	}
	for _, c := range cases {
		if got := claudeTools(c.in); !reflect.DeepEqual(got, c.want) {
			t.Errorf("claudeTools(%v) = %v, esperado %v", c.in, got, c.want)
		}
	}
}
