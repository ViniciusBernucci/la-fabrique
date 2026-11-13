package acceptance

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// Test identifica um teste automatizado pelo pacote e pelo nome.
type Test struct{ Pkg, Name string }

// Scenario é um cenário de ensaio de falha, segurança ou integração (T17-03): o esperado, a área
// e os testes que o exercitam. O observado vem da execução (`make ensaios`), nunca do catálogo.
type Scenario struct {
	ID, Area, Name, Expected string
	Tests                    []Test
}

const (
	pkgIntegration = "./tests/integration"
	pkgConnectors  = "./internal/integrations"
)

func it(names ...string) []Test {
	out := make([]Test, len(names))
	for i, n := range names {
		out[i] = Test{pkgIntegration, n}
	}
	return out
}

// RequiredAreas são as áreas que o ticket T17-03 exige cobrir.
var RequiredAreas = []string{"recuperação", "idempotência", "conflitos", "isolamento", "concorrência", "fallback", "orçamento", "aprovações", "conectores"}

// Catalog é o conjunto de cenários do ensaio.
func Catalog() []Scenario {
	return []Scenario{
		{"R01", "recuperação", "Queda e retomada seguras", "Interrupção segura é recuperada sozinha; incerteza abre decisão sem parar as outras tarefas; checkpoint corrompido exige decisão; limites por classe de falha.",
			it("TestACrashIsRecoveredAutomaticallyWhenItIsSafe", "TestUncertaintyOpensADecisionAndOtherEligibleTasksKeepGoing", "TestCorruptCheckpointAndVerifierProblemsRequireADecision", "TestRecoveryLimitsAreSeparatedPerFailureClass")},
		{"R02", "recuperação", "Checkpoint íntegro e retomada seletiva", "Checkpoint restaura com hashes iguais, recusa captura com processo vivo, detecta corrupção; retomada reexecuta só o conjunto afetado.",
			it("TestCheckpointRestoresInANewContainerAndHashesMatch", "TestInterruptedSnapshotWriteKeepsThePreviousCheckpointConsistent", "TestCheckpointCorruptionIsDetectedOnReadAndRestore", "TestResumingANodeRerunsOnlyTheAffectedSetOnTheOriginalWorkflow")},
		{"I01", "idempotência", "Efeito externo sem duplicar", "Timeout após o efeito não duplica; ausência provada permite repetir; estado inconclusivo bloqueia e abre decisão; execuções concorrentes enviam uma vez.",
			it("TestTimeoutAfterTheEffectLandedDoesNotDuplicate", "TestProvenAbsenceAllowsASafeRetry", "TestInconclusiveStateBlocksAndOpensADecisionWithEvidence", "TestConcurrentRunsOfTheSameOperationSendOnce")},
		{"I02", "idempotência", "Escrita na origem, push e PR sem duplicar", "Publicar o plano repetido não duplica itens; push com resposta perdida reconcilia pela prova; o mesmo comentário de revisão não duplica trabalho.",
			it("TestPublishingThePlanToTheOriginNeverDuplicatesOnRetry", "TestPushTimeoutIsSettledByRefProofWithoutDuplicates", "TestTheSameExternalCommentNeverDuplicatesWork", "TestAnObservedHumanMergeReleasesDependentsAndAStatusFailureIsVisibleAndReconcilable")},
		{"C01", "conflitos", "Mudança de requisitos e troca de origem", "Mudança compatível mantém histórico; conflito com trabalho em andamento preserva tudo e abre decisão; troca de plataforma mantém vínculos e uma única origem ativa.",
			it("TestACompatibleChangeKeepsTheHistory", "TestAConflictWithWorkInProgressPreservesEverythingAndOpensADecision", "TestSwitchingThePlatformKeepsHistoricalLinksAndASingleActiveOrigin", "TestConflictingInputChangesOpenADecisionAndChangeNothing")},
		{"C02", "conflitos", "PR fechada e remoto divergente", "PR fechada conserva artefatos e respeita a decisão; remoto divergente bloqueia com evidência e preserva o commit alheio.",
			it("TestAClosedPRKeepsArtifactsAndOffersThreeControlledActions", "TestDivergedRemoteBlocksWithEvidenceAndKeepsTheForeignCommit")},
		{"S01", "isolamento", "Projeto A não vê projeto B", "Memória, checkpoints e decisões são escopados por projeto; o que é de outro projeto é 'não encontrado'.",
			it("TestProjectAnNeverConsumesProjectBMemory", "TestCheckpointsAreScopedByProjectAndPrunedSafely", "TestResolvingChangesOnlyTheAffectedExecution")},
		{"S02", "isolamento", "Segredos e sessão", "O segredo sentinela não aparece em lugar nenhum; sessão revogada perde acesso; CSRF e origem são exigidos; senha sozinha não abre sessão.",
			it("TestSentinelSecretNeverAppearsAnywhere", "TestRevokedSessionsLoseAccess", "TestCSRFAndOriginAreEnforced", "TestPasswordAloneNeverOpensSession")},
		{"K01", "concorrência", "Fila, vaga única e limite global", "Uma só tarefa por projeto, paralelismo entre projetos, limite global respeitado e escritor obsoleto não escreve.",
			it("TestConcurrentWorkersNeverStartTwoTasksInOneProject", "TestConcurrentClaimsRespectOnePerProjectAndTheGlobalLimit", "TestAStaleWriterCannotWrite", "TestConcurrentResolutionsProduceExactlyOneResumeSignal")},
		{"F01", "fallback", "Troca de conta e esgotamento", "Cota na principal leva só à alternativa autorizada; troca exige checkpoint íntegro e sandbox quieto; esgotamento preserva a tarefa enquanto outra avança.",
			it("TestQuotaOnThePrimaryLeadsOnlyToTheAuthorizedAlternative", "TestSwitchingRequiresAnIntactCheckpointAndAQuietSandbox", "TestExhaustionPreservesTheTaskWhileAnotherAdvances", "TestFallbackSkipsIncompatibleAndOverBudgetAlternatives")},
		{"O01", "orçamento", "Teto de gasto", "Reservas concorrentes nunca passam do teto; o orçamento falha fechado; o laço do gateway confere o orçamento antes de cada chamada.",
			it("TestConcurrentReservationsNeverExceedTheAuthorizedCeiling", "TestBudgetFailsClosedAndWindowsRoll", "TestGatewayLoopEnforcesTheBudgetBeforeEveryCall", "TestSubscriptionQuotaIsRecordedSeparatelyFromAPISpend")},
		{"A01", "aprovações", "Aprovação, recusa e revogação", "Última autorização só vale uma vez; ação divergente é recusada sem gastar uso; revogar durante uso dispara parada segura; painel produz evento e estado verificáveis.",
			it("TestRaceForTheLastUseAuthorizesExactlyOne", "TestDivergentActionIsRefusedWithoutSpendingAUse", "TestRevokingWithAUseInFlightTriggersSafeStopAndCheckpoint", "TestApprovingRefusingAndRevokingFromThePanelLeaveVerifiableEventsAndState")},
		{"A02", "aprovações", "Compensação e operações sensíveis", "Compensação sensível exige autorização que combine; efeito irreversível é preservado e a recusa auditada; falha de compensação bloqueia agravantes.",
			it("TestSensitiveCompensationNeedsAMatchingAuthorization", "TestIrreversibleEffectsArePreservedAndTheRefusalIsAudited", "TestFailedCompensationKeepsTheWorkAndBlocksAggravatingOperations")},
		{"N01", "conectores", "Jira e ClickUp (fixtures)", "Os dois conectores passam o mesmo contrato, não seguem redirecionamentos e usam a autenticação correta.",
			[]Test{{pkgConnectors, "TestJiraContract"}, {pkgConnectors, "TestClickUpContract"}, {pkgConnectors, "TestJiraDoesNotFollowRedirects"}, {pkgConnectors, "TestClickUpSendsTokenWithoutBearerAndDoesNotFollowRedirects"}}},
		{"P01", "integração", "Feature da Draft PR ao pós-merge", "PR, commits, checks, origem, dependência e memória ficam ligados à mesma revisão integrada; repetir o pós-merge não muda nada.",
			it("TestFeatureRehearsalFromDraftPRToPostMergeCycleLinkedToTheIntegratedRevision", "TestOnlyAMergeConfirmedByTheOriginPromotesValidRecords")},
	}
}

var funcRe = regexp.MustCompile(`(?m)^func (Test[A-Za-z0-9_]+)\(t \*testing\.T\)`)

// CheckCatalog confere o catálogo contra o código: identificadores únicos, esperado e testes
// preenchidos, todas as áreas obrigatórias cobertas e cada teste existente no pacote.
func CheckCatalog(root string) []string {
	var out []string
	ids := map[string]bool{}
	areas := map[string]bool{}
	found := map[string]map[string]bool{}
	names := func(pkg string) map[string]bool {
		if m, ok := found[pkg]; ok {
			return m
		}
		m := map[string]bool{}
		files, _ := filepath.Glob(filepath.Join(root, pkg, "*_test.go"))
		for _, f := range files {
			b, err := os.ReadFile(f)
			if err != nil {
				continue
			}
			for _, mm := range funcRe.FindAllStringSubmatch(string(b), -1) {
				m[mm[1]] = true
			}
			// Testes que delegam a um contrato (func TestX(t *testing.T) { ... } na mesma linha).
			for _, mm := range regexp.MustCompile(`(?m)^func (Test[A-Za-z0-9_]+)\(t \*testing\.T\)?\s`).FindAllStringSubmatch(string(b), -1) {
				m[mm[1]] = true
			}
		}
		found[pkg] = m
		return m
	}
	for _, s := range Catalog() {
		if ids[s.ID] {
			out = append(out, "cenário repetido: "+s.ID)
		}
		ids[s.ID] = true
		areas[s.Area] = true
		if strings.TrimSpace(s.Expected) == "" || len(s.Tests) == 0 {
			out = append(out, s.ID+": falta o esperado ou os testes")
		}
		for _, t := range s.Tests {
			if !names(t.Pkg)[t.Name] {
				out = append(out, fmt.Sprintf("%s: o teste %s não existe em %s", s.ID, t.Name, t.Pkg))
			}
		}
	}
	for _, a := range RequiredAreas {
		if !areas[a] {
			out = append(out, "área obrigatória sem cenário: "+a)
		}
	}
	sort.Strings(out)
	return out
}

// Result é o observado de um cenário.
type Result struct {
	Scenario
	Status   string // PASS | FAIL | NOT_RUN
	Detail   string
	Duration time.Duration
}

// RenderRehearsal monta o relatório dos ensaios (esperado, observado, evidência e revisão).
func RenderRehearsal(results []Result, when time.Time, revision string) string {
	var b strings.Builder
	b.WriteString("# Ensaios de falha, segurança e integração\n\n")
	b.WriteString("> **Gerado** por `make ensaios` (`cmd/ensaios`): cada cenário roda os testes automatizados indicados contra PostgreSQL real e o relatório registra o **observado**. O catálogo é conferido contra o código (`internal/acceptance`). Não edite à mão.\n\n")
	fmt.Fprintf(&b, "Execução: %s · revisão do código: `%s`.\n\n", when.UTC().Format("2006-01-02 15:04 UTC"), revision)
	pass, fail, notRun := 0, 0, 0
	for _, r := range results {
		switch r.Status {
		case "PASS":
			pass++
		case "FAIL":
			fail++
		default:
			notRun++
		}
	}
	fmt.Fprintf(&b, "**Resultado:** %d PASS · %d FAIL · %d NOT_RUN, de %d cenários. Falha obrigatória volta ao ticket de origem e é retestada com `make ensaios`.\n\n", pass, fail, notRun, len(results))
	b.WriteString("| ID | Área | Cenário | Esperado | Observado | Evidência (testes) | Revisão |\n|---|---|---|---|---|---|---|\n")
	for _, r := range results {
		var ts []string
		for _, t := range r.Tests {
			ts = append(ts, "`"+t.Name+"`")
		}
		obs := fmt.Sprintf("**%s** (%.1fs)", r.Status, r.Duration.Seconds())
		if r.Detail != "" {
			obs += "<br>" + strings.ReplaceAll(r.Detail, "|", "/")
		}
		fmt.Fprintf(&b, "| %s | %s | %s | %s | %s | %s | NOT_RUN |\n", r.ID, r.Area, r.Name, r.Expected, obs, strings.Join(ts, "<br>"))
	}
	b.WriteString("\n## O que este relatório não prova\n\n")
	b.WriteString("- **Ambiente real:** Jira, ClickUp, GitHub, provedores de IA (CLI e API), VPS e projeto real não foram usados; os conectores rodam contra fixtures e o GitHub contra um servidor HTTP de mentira. Esses cenários seguem **NOT_RUN** até o responsável fornecer acessos (F01–F12 da [ficha](01-ficha-de-preparacao.md)).\n")
	b.WriteString("- **Revisão independente:** a coluna Revisão é NOT_RUN: o aceite do ticket exige revisão do cenário em sessão separada (skill `revisor-de-codigo`).\n")
	b.WriteString("- **Cobertura:** cada cenário reúne os testes que o exercitam; testes de outros tickets continuam na suíte completa (`make test-integration`).\n")
	return b.String()
}
