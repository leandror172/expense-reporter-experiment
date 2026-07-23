//go:build acceptance

package actions

// When closures: each runs the binary once and captures everything observable into
// the context.
//
// What belongs in an action's signature: the command's SUBJECT — the expense string,
// the flags, which input file. What does not: where the fixture lives. The scenario
// already declares that as Scenario.Fixture, so actions read ctx.FixtureDir rather
// than taking the same path a second time.

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"time"

	"expense-reporter/test/domain"

	"github.com/leandror172/acceptance-harness/harness"
)

// --- Context-carried flags ------------------------------------------------------
// The scenario context decides these, not the caller. Kept as two helpers rather
// than one with a boolean so that a command omitting the workbook reads as a
// deliberate choice at its call site instead of a false argument.

// dataDirFlag passes the classification data directory when the scenario set one.
func dataDirFlag(ctx *harness.Context) []string {
	if domain.DataDir(ctx) == "" {
		return nil
	}
	return []string{"--data-dir", domain.DataDir(ctx)}
}

// workbookFlag passes the workbook when the scenario set one.
func workbookFlag(ctx *harness.Context) []string {
	if domain.WorkbookPath(ctx) == "" {
		return nil
	}
	return []string{"--workbook", domain.WorkbookPath(ctx)}
}

// --- Single-expense commands ----------------------------------------------------

// RunClassify returns a When closure that runs the classify command.
func RunClassify(args ...string) func(*harness.Context) {
	return func(ctx *harness.Context) {
		cmdArgs := append([]string{"classify"}, dataDirFlag(ctx)...)
		runCommand(ctx, append(cmdArgs, args...)...)
	}
}

// RunAuto returns a When closure that runs the auto command.
func RunAuto(args ...string) func(*harness.Context) {
	return func(ctx *harness.Context) {
		cmdArgs := append([]string{"auto"}, dataDirFlag(ctx)...)
		cmdArgs = append(cmdArgs, workbookFlag(ctx)...)
		runCommand(ctx, append(cmdArgs, args...)...)
	}
}

// RunAdd returns a When closure that runs the add command with the given expense
// string. Extra flags (e.g. prediction context) are appended last.
func RunAdd(expenseString string, extraFlags ...string) func(*harness.Context) {
	return func(ctx *harness.Context) {
		args := []string{"add", expenseString}
		args = append(args, dataDirFlag(ctx)...)
		args = append(args, workbookFlag(ctx)...)
		runCommand(ctx, append(args, extraFlags...)...)
	}
}

// RunCorrect returns a When closure that runs the correct command.
// No --workbook on purpose: correct is feedback-only and never touches one.
func RunCorrect(expenseString string, extraFlags ...string) func(*harness.Context) {
	return func(ctx *harness.Context) {
		args := []string{"correct", expenseString}
		args = append(args, dataDirFlag(ctx)...)
		runCommand(ctx, append(args, extraFlags...)...)
	}
}

// RunAddDryRun returns a When closure that runs the add command with --dry-run.
// No --workbook: a dry run must not resolve one. Extra flags (e.g. "--json") are
// appended after the expense string.
func RunAddDryRun(expenseString string, extraFlags ...string) func(*harness.Context) {
	return func(ctx *harness.Context) {
		args := append([]string{"add", "--dry-run"}, dataDirFlag(ctx)...)
		args = append(args, expenseString)
		runCommand(ctx, append(args, extraFlags...)...)
	}
}

// --- batch-auto -----------------------------------------------------------------

// batchAutoRun is the variation between the batch-auto When closures: which input
// file the run reads, and where its output CSVs land. Everything else — fixture
// config, context flags, artifact registration — is identical across them, so it
// lives once in runBatchAuto.
type batchAutoRun struct {
	inputFile    string // "" → the fixture's default input.csv
	outputDirKey string // "" → the WorkDir; otherwise an artifact registered in Given
}

// runBatchAuto reads the scenario fixture's config, runs batch-auto, and registers
// the two output CSVs as artifacts.
func runBatchAuto(run batchAutoRun) func(*harness.Context) {
	return func(ctx *harness.Context) {
		cfg, err := domain.LoadExpenseFixtureConfig(ctx.FixtureDir)
		if err != nil {
			ctx.T.Fatalf("runBatchAuto: load fixture config: %v", err)
		}

		inputFile := run.inputFile
		if inputFile == "" {
			inputFile = "input.csv"
		}
		outputDir := ctx.WorkDir
		if run.outputDirKey != "" {
			registered, ok := ctx.Artifacts[run.outputDirKey]
			if !ok {
				ctx.T.Fatalf("runBatchAuto: artifact key %q not registered in Given", run.outputDirKey)
			}
			outputDir = registered
		}

		// No --threshold: T-32 replaced the confidence band with the keyword
		// agreement gate, and the flag is ignored.
		args := []string{
			"batch-auto",
			filepath.Join(ctx.WorkDir, inputFile),
			"--model", cfg.Model,
			"--top", fmt.Sprintf("%d", cfg.TopN),
			"--output-dir", outputDir,
		}
		args = append(args, dataDirFlag(ctx)...)
		args = append(args, workbookFlag(ctx)...)
		args = append(args, cfg.ExtraArgs...)
		runCommand(ctx, args...)

		ctx.Artifacts["classified.csv"] = filepath.Join(outputDir, "classified.csv")
		ctx.Artifacts["review.csv"] = filepath.Join(outputDir, "review.csv")
	}
}

// RunBatchAutoWithFixture runs batch-auto over the scenario fixture's input.csv.
func RunBatchAutoWithFixture() func(*harness.Context) {
	return runBatchAuto(batchAutoRun{})
}

// RunBatchAutoWithInput runs batch-auto over a named input file instead of the
// default input.csv. Use when one fixture holds several inputs.
func RunBatchAutoWithInput(inputFile string) func(*harness.Context) {
	return runBatchAuto(batchAutoRun{inputFile: inputFile})
}

// RunBatchAutoIntoArtifactDir writes the output CSVs into the directory registered
// in Given under outputDirKey, rather than beside the input.
func RunBatchAutoIntoArtifactDir(outputDirKey string) func(*harness.Context) {
	return runBatchAuto(batchAutoRun{outputDirKey: outputDirKey})
}

// RunBatchAuto returns a When closure that runs batch-auto with raw args — no
// fixture config, no context flags. For scenarios asserting on argument handling.
func RunBatchAuto(args ...string) func(*harness.Context) {
	return func(ctx *harness.Context) {
		runCommand(ctx, append([]string{"batch-auto"}, args...)...)
	}
}

// --- Review pipeline ------------------------------------------------------------

// RunReview renders the review page for csvPath and registers review.html.
func RunReview(csvPath string) func(*harness.Context) {
	return func(ctx *harness.Context) {
		outputPath := filepath.Join(ctx.WorkDir, "review.html")
		args := append([]string{"review", csvPath}, workbookFlag(ctx)...)
		args = append(args, "--output", outputPath)
		runCommand(ctx, args...)
		ctx.Artifacts["review.html"] = outputPath
	}
}

// RunApply applies a reviewed export.
func RunApply(reviewedPath string) func(*harness.Context) {
	return runApply(reviewedPath)
}

// RunApplyDryRun applies a reviewed export with --dry-run.
func RunApplyDryRun(reviewedPath string) func(*harness.Context) {
	return runApply(reviewedPath, "--dry-run")
}

// runApply is the shared apply invocation. The year is pinned so the scenarios stay
// deterministic across calendar years.
func runApply(reviewedPath string, extraFlags ...string) func(*harness.Context) {
	return func(ctx *harness.Context) {
		args := []string{"apply", reviewedPath, "--year", "2026"}
		args = append(args, workbookFlag(ctx)...)
		runCommand(ctx, append(args, extraFlags...)...)
	}
}

// RunGenerateWorkbook generates a workbook from the scenario fixture's taxonomy.json
// and the named entries file (fixture-relative; "" generates a skeleton). The output
// lands in the WorkDir as the "generated-workbook" artifact.
func RunGenerateWorkbook(entriesFile string, extraFlags ...string) func(*harness.Context) {
	return func(ctx *harness.Context) {
		outputPath := filepath.Join(ctx.WorkDir, "generated.xlsx")
		args := []string{
			"generate-workbook",
			"-o", outputPath,
			"--taxonomy", filepath.Join(ctx.FixtureDir, "taxonomy.json"),
		}
		if entriesFile != "" {
			args = append(args, "--entries", filepath.Join(ctx.FixtureDir, entriesFile))
		}
		runCommand(ctx, append(args, extraFlags...)...)
		ctx.Artifacts["generated-workbook"] = outputPath
	}
}

// runCommand executes binary with args, capturing stdout/stderr/exitcode into ctx.
func runCommand(ctx *harness.Context, args ...string) {
	if len(args) == 0 {
		ctx.T.Fatal("runCommand: no args provided")
	}
	label := args[0]
	if len(args) > 1 {
		label += " " + args[1]
	}
	ctx.T.Logf("  $ expense-reporter %s", label)
	start := time.Now()
	cmd := exec.Command(ctx.BinaryPath, args...)
	cmd.Env = domain.CommandEnv(ctx)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	ctx.Stdout = stdout.String()
	ctx.Stderr = stderr.String()
	elapsed := time.Since(start).Round(time.Second)
	if err == nil {
		ctx.ExitCode = 0
		ctx.T.Logf("  ✓ completed in %s", elapsed)
		return
	}
	if e, ok := err.(*exec.ExitError); ok {
		ctx.ExitCode = e.ExitCode()
		ctx.T.Logf("  ✗ completed in %s (exit %d)", elapsed, ctx.ExitCode)
	} else {
		ctx.T.Fatalf("runCommand %v: unexpected error: %v", args, err)
	}
}
