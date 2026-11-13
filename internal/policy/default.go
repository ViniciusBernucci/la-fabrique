package policy

// Options personaliza a política padrão por instalação/projeto.
type Options struct {
	// AllowedAPIHosts: hosts externos que aceitam GET/HEAD automaticamente.
	AllowedAPIHosts []string
	// AllowedMCPTools: nomes de ferramentas MCP liberadas explicitamente (a
	// allowlist por agente é do T08-04; anotações da própria ferramenta NUNCA liberam).
	AllowedMCPTools []string
}

func re(key, regex string) ParamMatch { return ParamMatch{Key: key, Regex: regex} }

// DefaultPolicy devolve a política inicial, já compilada. Falha fechada: o que
// não casa com nenhuma regra é negado.
func DefaultPolicy(o Options) *Policy {
	cmd := func(regex string) []ParamMatch { return []ParamMatch{re("command", regex)} }
	p := &Policy{Default: Deny, Rules: []Rule{
		// --- Scripts: proibido ---
		{ID: "script.network_tools", Effect: Deny, Tool: ToolScript, Params: cmd(`(^|\s)(curl|wget|ssh|scp|sftp|nc|ncat|netcat|telnet|ftp|rsync)(\s|$)`),
			Reason: "Ferramentas de rede não rodam em scripts: o acesso externo passa por adaptadores com política."},
		{ID: "script.privilege", Effect: Deny, Tool: ToolScript, Params: cmd(`(^|\s)(sudo|su|doas|chroot|mount|umount|nsenter|setcap)(\s|$)`),
			Reason: "Elevação de privilégio e montagens são proibidas."},
		{ID: "script.container_runtime", Effect: Deny, Tool: ToolScript, Params: cmd(`(^|\s)(docker|podman|kubectl|nerdctl|ctr)(\s|$)|docker\.sock`),
			Reason: "O código do projeto não acessa o runtime de containers."},
		{ID: "script.git_remote", Effect: Deny, Tool: ToolScript, Params: cmd(`^git (push|fetch|pull|clone|remote)( .*)?$`),
			Reason: "Git remoto só pelo broker da fábrica, não por script."},
		{ID: "script.rm_root", Effect: Deny, Tool: ToolScript, Params: cmd(`^rm .*-[a-zA-Z]*(rf|fr)[a-zA-Z]*\s+(/|~|\*|\.\.)(\s|$)`),
			Reason: "Remoção recursiva da raiz, do home ou de diretórios acima é proibida."},
		// --- Scripts: sensível ---
		{ID: "script.dependency_change", Effect: Approve, Tool: ToolScript, Params: cmd(`^(npm (install|i|add|update|uninstall)|pip3? install|go get|yarn add|pnpm add|cargo add)( .*)?$`),
			Reason: "Alterar dependências muda a cadeia de suprimentos e exige aprovação."},
		{ID: "script.shell_wrapper", Effect: Approve, Tool: ToolScript, Params: cmd(`^(sh|bash|zsh)( .*)?$`),
			Reason: "Um shell aninhado não pode ser analisado pela política."},
		{ID: "script.destructive", Effect: Approve, Tool: ToolScript, Params: cmd(`^(rm|mv)( .*)?$|^find .*(-exec|-delete)`),
			Reason: "Apagar ou mover arquivos exige aprovação."},
		{ID: "script.git_config", Effect: Approve, Tool: ToolScript, Params: cmd(`^git config( .*)?$`),
			Reason: "Configuração do Git exige aprovação."},
		// --- Scripts: baixo risco ---
		{ID: "script.build_test", Effect: Allow, Tool: ToolScript, Params: cmd(`^(make (test|lint|build|fmt|check)( .*)?|go (test|vet|build|fmt)( .*)?|npm (ci|test)( .*)?|npm run (build|lint|test)( .*)?|npx (biome|tsc|vitest)( .*)?)$`),
			Reason: "Build, testes e lint do projeto."},
		{ID: "script.read", Effect: Allow, Tool: ToolScript, Params: cmd(`^(ls|cat|head|tail|wc|pwd|echo|grep|find|stat|sort|uniq|diff|true|false)( .*)?$`),
			Reason: "Leitura e inspeção de arquivos."},
		{ID: "script.git_local", Effect: Allow, Tool: ToolScript, Params: cmd(`^git (status|diff|log|show|add|commit|branch|checkout|switch|restore|stash)( .*)?$`),
			Reason: "Git local, sem rede."},

		// --- Ferramentas nativas de arquivo (o caminho já é confinado ao workspace) ---
		{ID: "fs.read", Effect: Allow, Tool: ToolFS, Operation: "read", Reason: "Leitura no workspace."},
		{ID: "fs.list", Effect: Allow, Tool: ToolFS, Operation: "list", Reason: "Listagem no workspace."},
		{ID: "fs.search", Effect: Allow, Tool: ToolFS, Operation: "search", Reason: "Busca no workspace."},
		{ID: "fs.write", Effect: Allow, Tool: ToolFS, Operation: "write", Reason: "Escrita no workspace da feature."},

		// --- Git (broker da fábrica) ---
		{ID: "git.force", Effect: Deny, Tool: ToolGit, Params: []ParamMatch{{Key: "force", Equals: true}}, Reason: "Push forçado é proibido."},
		{ID: "git.push_protected", Effect: Deny, Tool: ToolGit, Operation: "push", Params: []ParamMatch{{Key: "branch", OneOf: []string{"main", "master", "production", "release"}}},
			Reason: "Nunca alterar a base protegida diretamente."},
		{ID: "git.merge", Effect: Deny, Tool: ToolGit, Operation: "merge", Reason: "Merge direto é proibido: a integração passa por PR revisada."},
		{ID: "git.local", Effect: Allow, Tool: ToolGit, Operation: "status", Reason: "Leitura do repositório."},
		{ID: "git.local_diff", Effect: Allow, Tool: ToolGit, Operation: "diff", Reason: "Leitura do repositório."},
		{ID: "git.commit", Effect: Allow, Tool: ToolGit, Operation: "commit", Reason: "Commit local na branch da feature."},
		{ID: "git.branch", Effect: Allow, Tool: ToolGit, Operation: "branch", Reason: "Branch da feature."},
		{ID: "git.push", Effect: Allow, Tool: ToolGit, Operation: "push", Params: []ParamMatch{re("branch", `^[A-Za-z0-9._/-]+$`)}, Reason: "Push restrito à branch da feature."},
		{ID: "git.draft_pr", Effect: Allow, Tool: ToolGit, Operation: "draft_pr", Reason: "Draft PR aguardando revisão."},

		// --- API ---
		{ID: "api.write", Effect: Approve, Tool: ToolAPI, Operation: "POST", Reason: "Escrita em API externa exige aprovação."},
		{ID: "api.write_put", Effect: Approve, Tool: ToolAPI, Operation: "PUT", Reason: "Escrita em API externa exige aprovação."},
		{ID: "api.write_patch", Effect: Approve, Tool: ToolAPI, Operation: "PATCH", Reason: "Escrita em API externa exige aprovação."},
		{ID: "api.delete", Effect: Approve, Tool: ToolAPI, Operation: "DELETE", Reason: "Remoção em API externa exige aprovação."},

		// --- MCP: anotações da ferramenta não liberam nada ---
		{ID: "mcp.list", Effect: Allow, Tool: ToolMCP, Operation: "tools/list", Reason: "Descoberta de ferramentas."},
		{ID: "mcp.sensitive_by_name", Effect: Approve, Tool: ToolMCP, Operation: "tools/call",
			Params: []ParamMatch{re("tool", `^(delete|remove|drop|write|create|update|send|post|exec|run|deploy|publish)`)},
			Reason: "Ferramenta MCP que altera estado exige aprovação."},

		// --- Conectores (Jira, ClickUp, GitHub) ---
		{ID: "connector.read", Effect: Allow, Tool: ToolConnector, Operation: "list_*", Reason: "Leitura em conector."},
		{ID: "connector.read_get", Effect: Allow, Tool: ToolConnector, Operation: "get_*", Reason: "Leitura em conector."},
		{ID: "connector.lifecycle", Effect: Allow, Tool: ToolConnector, Operation: "create_subtask", Reason: "Escrita do ciclo de vida da tarefa."},
		{ID: "connector.lifecycle_link", Effect: Allow, Tool: ToolConnector, Operation: "add_link", Reason: "Escrita do ciclo de vida da tarefa."},
		{ID: "connector.lifecycle_dep", Effect: Allow, Tool: ToolConnector, Operation: "add_dependency", Reason: "Escrita do ciclo de vida da tarefa."},
		{ID: "connector.lifecycle_transition", Effect: Allow, Tool: ToolConnector, Operation: "transition", Reason: "Escrita do ciclo de vida da tarefa."},
		{ID: "connector.delete", Effect: Deny, Tool: ToolConnector, Operation: "delete_*", Reason: "Conectores não apagam itens da origem."},
		{ID: "connector.archive", Effect: Approve, Tool: ToolConnector, Operation: "archive_*", Reason: "Arquivar na origem exige aprovação."},

		// --- Clientes CLI de IA (os modos já vêm restritos pelo adaptador) ---
		{ID: "cli.ai_client", Effect: Allow, Tool: ToolCLI, Operation: "ai_client.run",
			Params: []ParamMatch{{Key: "mode", OneOf: []string{"read_only", "workspace_write"}}}, Reason: "Cliente de IA em modo restrito, dentro do sandbox."},
	}}
	for _, h := range o.AllowedAPIHosts {
		for _, m := range []string{"GET", "HEAD"} {
			p.Rules = append(p.Rules, Rule{ID: "api.read." + m + "." + h, Effect: Allow, Tool: ToolAPI, Operation: m, Target: h, Reason: "Leitura em host da lista permitida."})
		}
	}
	for _, t := range o.AllowedMCPTools {
		p.Rules = append(p.Rules, Rule{ID: "mcp.allow." + t, Effect: Allow, Tool: ToolMCP, Operation: "tools/call", Params: []ParamMatch{{Key: "tool", Equals: t}}, Reason: "Ferramenta MCP liberada explicitamente."})
	}
	if err := p.Compile(); err != nil {
		panic("política padrão inválida: " + err.Error()) // erro de programação, pego pelos testes
	}
	return p
}
