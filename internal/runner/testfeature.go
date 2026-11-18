package runner

import "strings"

// A feature fictícia do botão "Testar o fluxo": pequena, igual em todo teste e independente da stack do projeto.
// Tem uma parte de BACKEND e uma de FRONTEND (as duas trilhas do fluxo padrão) e um verificador que os nós de
// checks rodam no modo teste. Nada dela vai para fora: no modo teste o push e a Draft PR são só simulados.
const (
	TestFeatureTitle = "Teste da fábrica: selo de saúde do sistema"

	TestBackendFile  = "fabrica-teste/backend/saude.md"
	TestFrontendFile = "fabrica-teste/frontend/selo-saude.html"
)

// TestFeatureTask é o pedido entregue aos agentes.
const TestFeatureTask = `FEATURE FICTÍCIA DE TESTE (serve só para testar a fábrica; não mexa em nenhum outro arquivo do projeto).

Objetivo: um "selo de saúde" que mostra se o sistema está saudável.

Trilha BACKEND:
- Crie o arquivo ` + TestBackendFile + ` descrevendo o endpoint fictício GET /saude, que responde {"status":"ok"} com HTTP 200.
- O arquivo deve conter o título "Endpoint de saúde", a rota "GET /saude" e a resposta {"status":"ok"}.

Trilha FRONTEND:
- Crie o arquivo ` + TestFrontendFile + ` com uma página HTML simples que mostra um selo verde com o texto "Sistema saudável".
- A página deve conter o texto "Sistema saudável" e mencionar a rota "/saude".

Cada trilha cria apenas o seu arquivo. Ao terminar, resuma o que foi criado.`

// TestChecks são os comandos que os nós de checks rodam no modo teste: conferem os dois arquivos da feature.
var TestChecks = []string{
	`test -f ` + TestBackendFile,
	`grep -q "GET /saude" ` + TestBackendFile,
	`grep -q '"status":"ok"' ` + TestBackendFile,
	`test -f ` + TestFrontendFile,
	`grep -q "Sistema saudável" ` + TestFrontendFile,
	`grep -q "/saude" ` + TestFrontendFile,
}

// TestChecksText é TestChecks, um comando por linha.
func TestChecksText() string { return strings.Join(TestChecks, "\n") }

// StartTestInput devolve a entrada padronizada do teste do fluxo.
func StartTestInput(projectID, workspace, by string) StartInput {
	return StartInput{ProjectID: projectID, Title: TestFeatureTitle, Task: TestFeatureTask, Workspace: workspace,
		CreatedBy: by, Kind: KindTest, PRDDone: true}
}
