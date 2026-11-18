package factory

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/ViniciusBernucci/le-fabrique/internal/agents"
	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
	"github.com/ViniciusBernucci/le-fabrique/internal/skills"
)

// Prompt é o comando que a fábrica entrega ao CLI do agente: quem ele é, qual skill usar, o que a fábrica
// quer e as regras. A pessoa não escreve isto: sai do cadastro do agente, do nó e da tarefa da execução.
func Prompt(eff agents.Effective, project, title, task, skillDir string, test bool, extra string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Você é o agente %q da La fabrique, trabalhando no projeto %q. Responda em português.\n\n", eff.AgentName, project)
	if len(eff.Skills) > 0 {
		b.WriteString("## Skills\n")
		for _, s := range eff.Skills {
			fmt.Fprintf(&b, "- Use a skill `%s` (instruções em %s/%s/SKILL.md; leia também a pasta references/ dela quando precisar) para conduzir esta etapa.\n", s, skillDir, s)
		}
		b.WriteString("\n")
	}
	if strings.TrimSpace(eff.Instructions) != "" {
		fmt.Fprintf(&b, "## Instruções do agente\n%s\n\n", strings.TrimSpace(eff.Instructions))
	}
	fmt.Fprintf(&b, "## Tarefa da fábrica\nTítulo: %s\n", title)
	if strings.TrimSpace(task) != "" {
		fmt.Fprintf(&b, "%s\n", strings.TrimSpace(task))
	}
	b.WriteString("\n")
	if strings.TrimSpace(extra) != "" {
		b.WriteString(strings.TrimSpace(extra))
		b.WriteString("\n\n")
	}
	if strings.TrimSpace(eff.Prompt) != "" {
		fmt.Fprintf(&b, "## O que fazer nesta etapa\n%s\n\n", strings.TrimSpace(eff.Prompt))
	}
	if test {
		b.WriteString(`## MODO TESTE DA FÁBRICA
Esta execução é um TESTE com uma feature fictícia, para verificar o funcionamento do fluxo. Faça somente o que o papel desta etapa pede, aplicado a essa feature:
- se sua etapa é de implementação, crie apenas os arquivos descritos para a sua trilha (backend ou frontend);
- se é de planejamento, escreva o plano em texto na sua resposta (não crie arquivos);
- se é de revisão, leia os arquivos existentes e responda com os achados (não altere nada, a menos que a etapa peça correção);
- se é de outro tipo, descreva em poucas linhas o que faria.
Não toque em nenhum outro arquivo do projeto.

`)
	}
	b.WriteString(`## Regras
- Trabalhe só dentro do repositório atual. Edite os arquivos necessários; a fábrica recolhe as alterações.
- Não faça commit, push, merge nem abra Pull Request: isso é feito pela fábrica.
- Nunca grave segredos em arquivos, logs ou mensagens.
- Ao terminar, responda com um resumo curto do que fez, do que não conseguiu fazer e dos próximos passos.
`)
	return b.String()
}

// skillTar monta o tar com as skills do agente, prontas para extrair em <dir>/ (cada uma em <dir>/<nome>/).
// Devolve também os caminhos a ignorar no git (para a skill não entrar no patch).
func skillTar(ctx context.Context, db persistence.DBTX, names []string, dir string) ([]byte, error) {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	seen := map[string]bool{}
	add := func(name string, data []byte) error {
		if seen[name] {
			return nil
		}
		seen[name] = true
		if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0o644, Size: int64(len(data)), Typeflag: tar.TypeReg}); err != nil {
			return err
		}
		_, err := tw.Write(data)
		return err
	}
	sort.Strings(names)
	for _, n := range names {
		sk, err := skills.GetByName(ctx, db, n)
		if err != nil {
			return nil, fmt.Errorf("a skill %q do agente não está no catálogo: %w", n, err)
		}
		rows, err := db.Query(ctx, `SELECT path, content FROM skill_files WHERE skill_id = $1 ORDER BY path`, sk.ID)
		if err != nil {
			return nil, persistence.MapError(err)
		}
		hasMain := false
		for rows.Next() {
			var path string
			var content []byte
			if err := rows.Scan(&path, &content); err != nil {
				rows.Close()
				return nil, err
			}
			if strings.Contains(path, "..") || strings.HasPrefix(path, "/") {
				continue
			}
			if path == "SKILL.md" {
				hasMain = true
			}
			if err := add(dir+"/"+n+"/"+path, content); err != nil {
				rows.Close()
				return nil, err
			}
		}
		rows.Close()
		if !hasMain {
			if err := add(dir+"/"+n+"/SKILL.md", []byte(sk.Body)); err != nil {
				return nil, err
			}
		}
	}
	if err := tw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
