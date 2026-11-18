package factory

import "sort"

// claudeTools traduz as ferramentas nativas do catálogo (read_file, run_command…) nas permissões do Claude Code.
// Leitura, busca e edição já fazem parte do modo de escrita do terminal; aqui entram os extras: executar comandos
// (Bash) e consultar o git. Só vale para o Claude (o Codex não tem lista de ferramentas). Um nome desconhecido é
// ignorado: o catálogo, não o nó, é quem decide o que existe.
func claudeTools(catalog []string) []string {
	m := map[string][]string{
		"run_command": {"Bash"},
		"git_status":  {"Bash(git status:*)"},
		"git_diff":    {"Bash(git diff:*)"},
		"git_log":     {"Bash(git log:*)"},
	}
	seen := map[string]bool{}
	var out []string
	for _, t := range catalog {
		for _, c := range m[t] {
			if !seen[c] {
				seen[c] = true
				out = append(out, c)
			}
		}
	}
	// "Bash" liberado por inteiro já cobre os recortes de git: não repete.
	if seen["Bash"] {
		out = []string{"Bash"}
	}
	sort.Strings(out)
	return out
}
