//go:build acceptance

package acceptance_test

// The writing half of telegram-import. These pin plan D7 (a line lands in the month of
// its RESOLVED expense date, not the day it was sent), D8 (an existing target file is
// never overwritten without --force, and a refusal writes nothing at all) and the T-63
// shape of the rejects file. The last scenario is the step-3 gate itself — the file the
// converter wrote is read by the real batch-auto — and needs Ollama because batch-auto
// classifies, so it runs in the -full group only.

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"expense-reporter/test/actions"
	"expense-reporter/test/expect"
	"expense-reporter/test/extern"

	"github.com/leandror172/acceptance-harness/harness"
	"github.com/leandror172/acceptance-harness/verify"
)

// sentinelRow is what an earlier run left in the May file; a refusal must leave it intact.
const sentinelRow = "sentinel;01/05/2025;1,00"

func TestTelegramImport_WritesOneFilePerResolvedMonth(t *testing.T) {
	fixDir := filepath.Join(fixturesDir(), "telegram-import-dry-run")

	harness.Run(t, harness.Scenario{
		Name:    "telegram-import writes one batch-auto input per month of the resolved expense date, plus the rejects",
		Fixture: fixDir,
		When:    actions.RunTelegramImportWriting("result.json"),
		Then: slices.Concat(
			commandSucceeded(),
			monthFilesSplitByResolvedExpenseDate(),
			repairedLinesCarryTheirOrigin(),
			rejectsListEveryAttemptedExpenseWithItsReason(),
			conversationReachesNoFile(),
		),
	})
}

func TestTelegramImport_RefusesToOverwriteAnExistingMonthFile(t *testing.T) {
	fixDir := filepath.Join(fixturesDir(), "telegram-import-dry-run")

	harness.Run(t, harness.Scenario{
		Name:    "telegram-import refuses to overwrite a month file that already exists, and writes nothing",
		Fixture: fixDir,
		Given:   monthFileAlreadyExported(),
		When:    actions.RunTelegramImportWriting("result.json"),
		Then: slices.Concat(
			commandFailed(),
			blockingFileNamedAndForceOffered(),
			nothingWrittenWhenRefusing(),
		),
	})
}

func TestTelegramImport_ForceReplacesTheBlockingFile(t *testing.T) {
	fixDir := filepath.Join(fixturesDir(), "telegram-import-dry-run")

	harness.Run(t, harness.Scenario{
		Name:    "telegram-import --force replaces the blocking month file",
		Fixture: fixDir,
		Given:   monthFileAlreadyExported(),
		When:    actions.RunTelegramImportWriting("result.json", "--force"),
		Then: slices.Concat(
			commandSucceeded(),
			blockingFileReplaced(),
		),
	})
}

func TestTelegramImport_NoRejectsFileWhenEveryMessageConverts(t *testing.T) {
	fixDir := filepath.Join(fixturesDir(), "telegram-import-clean")

	harness.Run(t, harness.Scenario{
		Name:    "telegram-import leaves no rejects file when every message converts",
		Fixture: fixDir,
		When:    actions.RunTelegramImportWriting("result.json"),
		Then: slices.Concat(
			commandSucceeded(),
			cleanExportLeavesNoRejectsFile(),
		),
	})
}

// The step-3 gate: the month file the converter wrote is read by the real batch-auto.
// Ollama-gated because batch-auto classifies every parsed row; the deterministic half of
// the same claim is the round-trip unit test in cmd/.../telegram_import_test.go.
func TestTelegramImport_MonthFileIsAcceptedByBatchAuto(t *testing.T) {
	extern.RequireOllama(t, "")
	fixDir := filepath.Join(fixturesDir(), "telegram-import-to-batch-auto")

	harness.Run(t, harness.Scenario{
		Name:    "a month file written by telegram-import is accepted line for line by batch-auto",
		Fixture: fixDir,
		Given:   taxonomyAuthoredWithTrainingData(),
		When:    monthFileFedToBatchAuto(),
		Then: slices.Concat(
			commandSucceeded(),
			everyConvertedLineParsedByBatchAuto(),
		),
	})
}

// --- Given helpers ---

// monthFileAlreadyExported: an earlier run — or a hand repair — already put a May file
// in the output directory. Plan D8 says the converter must not silently destroy it.
func monthFileAlreadyExported() func(*harness.Context) {
	return func(ctx *harness.Context) {
		path := filepath.Join(ctx.WorkDir, "expenses-2025-05.csv")
		if err := os.WriteFile(path, []byte(sentinelRow+"\n"), 0o644); err != nil {
			ctx.T.Fatalf("writing the pre-existing month file: %v", err)
		}
	}
}

// --- When helpers ---

// monthFileFedToBatchAuto runs two commands in one When because the claim is about the
// hand-off between them; Then sees the second run's output and artifacts.
func monthFileFedToBatchAuto() func(*harness.Context) {
	return func(ctx *harness.Context) {
		actions.RunTelegramImportWriting("result.json")(ctx)
		actions.RunBatchAutoWithInput("expenses-2025-05.csv")(ctx)
	}
}

// --- Then helpers (composable) ---

// monthFilesSplitByResolvedExpenseDate: the January line was SENT in July and dated
// 09/01, so it belongs to January (plan D7); the April line was sent on 04/05.
func monthFilesSplitByResolvedExpenseDate() []func(*harness.Context) {
	return []func(*harness.Context){
		verify.WorkFileExists("expenses-2025-01.csv"),
		verify.FileContains("expenses-2025-01.csv", "Cartao Teste ADM;09/01/2025;405,25 x4"),
		verify.WorkFileExists("expenses-2025-04.csv"),
		verify.FileContains("expenses-2025-04.csv", "Mercado;30/04/2025;1.234,56"),
		verify.WorkFileExists("expenses-2025-05.csv"),
		verify.FileContains("expenses-2025-05.csv", "Padaria;03/05/2025;12,50"),
		verify.WorkFileExists("expenses-2025-07.csv"),
		verify.FileContains("expenses-2025-07.csv", "Cinema;08/07/2025;45,00"),
		verify.FileContains("expenses-2025-07.csv", "Tela celular;25/07/2025;90,00"),
	}
}

// repairedLinesCarryTheirOrigin: the audit trail rides inside the file batch-auto reads.
// The reader strips a whitespace-preceded '#' after three complete fields (T-63).
func repairedLinesCarryTheirOrigin() []func(*harness.Context) {
	return []func(*harness.Context){
		verify.FileContains("expenses-2025-05.csv", "Almoço teste;06/05/2025;50,00   # repaired from: Almoço teste, 06/05; 50,00"),
	}
}

// rejectsListEveryAttemptedExpenseWithItsReason: stream order, and exactly these four —
// a receipt or a chat line leaking into the rejects file fails the length check.
func rejectsListEveryAttemptedExpenseWithItsReason() []func(*harness.Context) {
	return []func(*harness.Context){
		expect.FailedRowsCarryTheirReason("telegram-rejects.csv", []string{
			"Uber; 05/07; 16:20",
			"Café; 31/02; 8,00",
			"É 450,00 três meses, tá?",
			"Bar; 15/05/ 2025,50",
		}),
	}
}

func conversationReachesNoFile() []func(*harness.Context) {
	return []func(*harness.Context){
		verify.FileNotContains("telegram-rejects.csv", "Não quer mais?"),
	}
}

func blockingFileNamedAndForceOffered() []func(*harness.Context) {
	return []func(*harness.Context){
		verify.OutputContains("expenses-2025-05.csv"),
		verify.OutputContains("--force"),
	}
}

// nothingWrittenWhenRefusing: all-or-nothing. A refusal that had already written three
// of four files would leave a half-exported month that looks complete.
func nothingWrittenWhenRefusing() []func(*harness.Context) {
	return []func(*harness.Context){
		verify.FileContains("expenses-2025-05.csv", sentinelRow),
		verify.FileAbsent("expenses-2025-01.csv"),
		verify.FileAbsent("expenses-2025-04.csv"),
		verify.FileAbsent("expenses-2025-07.csv"),
		verify.FileAbsent("telegram-rejects.csv"),
	}
}

func blockingFileReplaced() []func(*harness.Context) {
	return []func(*harness.Context){
		verify.FileContains("expenses-2025-05.csv", "Padaria;03/05/2025;12,50"),
		verify.FileNotContains("expenses-2025-05.csv", sentinelRow),
	}
}

// cleanExportLeavesNoRejectsFile: existence is the signal (T-63), so a clean run must
// leave nothing behind.
func cleanExportLeavesNoRejectsFile() []func(*harness.Context) {
	return []func(*harness.Context){
		verify.WorkFileExists("expenses-2025-05.csv"),
		verify.FileAbsent("telegram-rejects.csv"),
	}
}

// everyConvertedLineParsedByBatchAuto: failed.csv exists exactly when a row was
// rejected, so its absence means every converted line — the repaired one included —
// went through the real reader.
func everyConvertedLineParsedByBatchAuto() []func(*harness.Context) {
	return []func(*harness.Context){
		verify.FileAbsent("failed.csv"),
	}
}
