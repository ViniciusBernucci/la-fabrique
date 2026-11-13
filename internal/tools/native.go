package tools

// Natives devolve as ferramentas nativas: leitura, edição, comando e consulta Git.
func Natives() []Tool {
	lim := Limits{TimeoutSeconds: 60, MaxOutputBytes: 1 << 20}
	return []Tool{
		{Name: "read_file", Kind: KindNative, Description: "Lê um arquivo do workspace.", Permissions: []string{PermRead}, Limits: lim,
			Input:  []Field{{Name: "path", Type: TypePath, Required: true}, {Name: "max_bytes", Type: TypeInt, Min: 1, Max: 1 << 20}},
			Output: []Field{{Name: "content", Type: TypeString, Required: true}, {Name: "truncated", Type: TypeBool, Required: true}, {Name: "bytes", Type: TypeInt, Required: true}}},
		{Name: "write_file", Kind: KindNative, Description: "Cria ou substitui um arquivo do workspace.", Permissions: []string{PermWrite}, Limits: lim,
			Input:  []Field{{Name: "path", Type: TypePath, Required: true}, {Name: "content", Type: TypeString, Required: true, Max: 1 << 20}},
			Output: []Field{{Name: "path", Type: TypePath, Required: true}, {Name: "bytes", Type: TypeInt, Required: true}}},
		{Name: "edit_file", Kind: KindNative, Description: "Troca exatamente uma ocorrência de um trecho num arquivo do workspace.", Permissions: []string{PermRead, PermWrite}, Limits: lim,
			Input: []Field{{Name: "path", Type: TypePath, Required: true}, {Name: "old", Type: TypeString, Required: true, Max: 100000},
				{Name: "new", Type: TypeString, Required: true, Max: 100000}},
			Output: []Field{{Name: "path", Type: TypePath, Required: true}, {Name: "replacements", Type: TypeInt, Required: true}}},
		{Name: "list_dir", Kind: KindNative, Description: "Lista um diretório do workspace (diretórios terminam em /).", Permissions: []string{PermRead}, Limits: lim,
			Input:  []Field{{Name: "path", Type: TypePath, Required: true}},
			Output: []Field{{Name: "entries", Type: TypeList, Required: true}}},
		{Name: "search", Kind: KindNative, Description: "Busca um texto nos arquivos do workspace.", Permissions: []string{PermRead}, Limits: lim,
			Input:  []Field{{Name: "pattern", Type: TypeString, Required: true, Max: 200}, {Name: "path", Type: TypePath}},
			Output: []Field{{Name: "matches", Type: TypeList, Required: true}, {Name: "truncated", Type: TypeBool, Required: true}}},
		{Name: "run_command", Kind: KindNative, Description: "Roda um comando no sandbox, sob a política de risco.", Permissions: []string{PermExec}, Limits: Limits{TimeoutSeconds: 600, MaxOutputBytes: 1 << 20},
			Input:  []Field{{Name: "command", Type: TypeString, Required: true, Max: 2000}},
			Output: []Field{{Name: "exit_code", Type: TypeInt, Required: true}, {Name: "stdout", Type: TypeString, Required: true}, {Name: "stderr", Type: TypeString, Required: true}}},
		{Name: "git_status", Kind: KindNative, Description: "Estado do repositório (git status --porcelain).", Permissions: []string{PermGit}, Limits: lim, Requires: []string{"git"},
			Output: []Field{{Name: "output", Type: TypeString, Required: true}}},
		{Name: "git_diff", Kind: KindNative, Description: "Diferenças do repositório (git diff).", Permissions: []string{PermGit}, Limits: lim, Requires: []string{"git"},
			Input:  []Field{{Name: "path", Type: TypePath}},
			Output: []Field{{Name: "diff", Type: TypeString, Required: true}}},
	}
}
