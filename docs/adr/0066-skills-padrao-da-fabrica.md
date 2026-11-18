# ADR-0066 — Skills padrão da fábrica, instaladas e associadas a todos os projetos

- **Estado:** ACEITA · **Relacionadas:** [ADR-0014](0014-catalogo-de-skills-e-importacao-segura.md), [ADR-0034](0034-perfil-tecnico-e-recursos-do-projeto.md)

## Contexto
As skills da fábrica (`escrever-prd-e-specs`, `lf-descobrir-ambiente-visual`, `lf-implementar-tarefa`, `lf-revisar-backend`, `lf-revisar-frontend`, `lf-tech-lead`) são o padrão de trabalho de todo projeto. Importá-las à mão, uma a uma, em cada instalação não serve: elas precisam já estar configuradas. O catálogo de skills é global, e cada projeto *associa* os recursos que usa (ADR-0034), de modo que um projeto novo nascia sem nenhuma.

## Decisão
1. **Os pacotes moram no repositório e vão embutidos no binário** (`internal/skills/defaults/<nome>/`, com `SKILL.md`, `references/`, `assets/`), com `go:embed`. A pasta de origem fora do repositório deixa de ser necessária. Para mudar uma skill padrão, edite o pacote aqui.
2. **A API instala ao subir** (`skills.InstallDefaults`), de forma idempotente. Falhar não impede a API de subir; a próxima subida tenta de novo. Migration 00069: `skills.is_default`.
3. **Regras de instalação:**
   - skill nova entra no catálogo, é marcada como padrão (`source_ref = padrao:<nome>`) e **associada a todos os projetos existentes**;
   - padrão ainda intocada (`source_ref` `padrao:…`) com conteúdo diferente do embutido é **atualizada** (nova versão);
   - qualquer outra versão com o mesmo nome (editada ou importada por alguém) **é mantida**: a fábrica nunca sobrescreve o que uma pessoa mudou, nem transforma uma skill própria em padrão;
   - se o mesmo conteúdo já estava no catálogo, importado à mão, ele passa a valer como padrão.
4. **Projetos novos já nascem com as skills padrão associadas**, no mesmo comando que cria o projeto (CTE em `projects.Create`).
5. **A associação acontece só na instalação.** Quem tirar uma skill padrão de um projeto não a vê voltar na próxima subida; uma skill padrão nova, numa versão futura, é associada a todos os projetos uma vez.
6. **Tela:** a lista de Skills mostra o selo **Padrão**; o painel do projeto mostra os recursos associados por tipo (antes era o texto fixo "Nenhum associado").

## Limites
- O conteúdo das skills é o da pasta de origem do responsável, sem alterações. As skills de `skills/` na raiz do repositório (para os agentes de desenvolvimento) são outras e seguem pelo `make sync-skills`.
- Marcar como padrão não impede editar, substituir ou desassociar. Apagar uma skill padrão pela tela faz a próxima subida reinstalá-la e associá-la de novo a todos os projetos.
- Skills associadas são só registradas (existência), como no ADR-0034: não há validação de uso na execução além do que já existia.
- Projetos criados direto no banco (por exemplo `seed-dev`) não recebem as associações; só os criados pela API e os existentes na instalação.
