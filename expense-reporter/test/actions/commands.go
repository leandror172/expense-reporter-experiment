//go:build acceptance

package actions

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"time"

	"expense-reporter/test/harness"
)

// RunClassify returns a When closure that runs the classify command.
// Passes --data-dir from ctx when set.
func RunClassify(args ...string) func(*harness.Context) {
	return func(ctx *harness.Context) {
		cmdArgs := []string{"classify"}
		if ctx.DataDir != "" {
			cmdArgs = append(cmdArgs, "--data-dir", ctx.DataDir)
		}
		runCommand(ctx, append(cmdArgs, args...)...)
	}
}

// RunAuto returns a When closure that runs the auto command.
// Passes --data-dir and --workbook from ctx when set.
func RunAuto(args ...string) func(*harness.Context) {
	return func(ctx *harness.Context) {
		cmdArgs := []string{"auto"}
		if ctx.DataDir != "" {
			cmdArgs = append(cmdArgs, "--data-dir", ctx.DataDir)
		}
		if ctx.WorkbookPath != "" {
			cmdArgs = append(cmdArgs, "--workbook", ctx.WorkbookPath)
		}
		runCommand(ctx, append(cmdArgs, args...)...)
	}
}

// RunAdd returns a When closure that runs the add command with the given expense string.
// Passes --data-dir and --workbook from ctx when set. Extra flags (e.g. prediction context) appended last.
func RunAdd(expenseString string, extraFlags ...string) func(*harness.Context) {
	return func(ctx *harness.Context) {
		args := []string{"add", expenseString}
		if ctx.DataDir != "" {
			args = append(args, "--data-dir", ctx.DataDir)
		}
		if ctx.WorkbookPath != "" {
			args = append(args, "--workbook", ctx.WorkbookPath)
		}
		args = append(args, extraFlags...)
		runCommand(ctx, args...)
	}
}

// RunCorrect returns a When closure that runs the correct command.
// Passes --data-dir from ctx when set. Does NOT pass --workbook (correct is feedback-only).
func RunCorrect(expenseString string) func(*harness.Context) {
	return func(ctx *harness.Context) {
		args := []string{"correct", expenseString}
		if ctx.DataDir != "" {
			args = append(args, "--data-dir", ctx.DataDir)
		}
		runCommand(ctx, args...)
	}
}

// RunAddDryRun returns a When closure that runs the add command with --dry-run.
// Passes --data-dir from ctx when set. Extra flags (e.g., "--json") are appended after the expense string.
func RunAddDryRun(expenseString string, extraFlags ...string) func(*harness.Context) {
	return func(ctx *harness.Context) {
		args := []string{"add", "--dry-run"}
		if ctx.DataDir != "" {
			args = append(args, "--data-dir", ctx.DataDir)
		}
		args = append(args, expenseString)
		args = append(args, extraFlags...)
		runCommand(ctx, args...)
	}
}

// RunBatchAuto returns a When closure that runs the batch-auto command.
func RunBatchAuto(args ...string) func(*harness.Context) {
	return func(ctx *harness.Context) {
		runCommand(ctx, append([]string{"batch-auto"}, args...)...)
	}
}

// RunBatchAutoWithFixture reads the fixture config.json, builds batch-auto args, runs the command,
// and registers classified.csv and review.csv in ctx.Artifacts.
func RunBatchAutoWithFixture(fixtureDir string) func(*harness.Context) {
	return func(ctx *harness.Context) {
		cfg, err := harness.LoadFixtureConfig(fixtureDir)
		if err != nil {
			ctx.T.Fatalf("RunBatchAutoWithFixture: load config: %v", err)
		}
		args := []string{
			"batch-auto",
			filepath.Join(ctx.WorkDir, "input.csv"),
			"--model", cfg.Model,
			"--threshold", fmt.Sprintf("%.2f", cfg.Threshold),
			"--top", fmt.Sprintf("%d", cfg.TopN),
			"--output-dir", ctx.WorkDir,
		}
		if ctx.DataDir != "" {
			args = append(args, "--data-dir", ctx.DataDir)
		}
		if ctx.WorkbookPath != "" {
			args = append(args, "--workbook", ctx.WorkbookPath)
		}
		args = append(args, cfg.ExtraArgs...)
		runCommand(ctx, args...)
		ctx.Artifacts["classified.csv"] = filepath.Join(ctx.WorkDir, "classified.csv")
		ctx.Artifacts["review.csv"] = filepath.Join(ctx.WorkDir, "review.csv")
		ctx.Artifacts["rollover.csv"] = filepath.Join(ctx.WorkDir, "rollover.csv")
	}
}

// RunBatchAutoWithInput reads config from fixtureDir but uses inputFile as the CSV input
// instead of the default "input.csv". Registers classified.csv and review.csv in ctx.Artifacts.
// Use when a fixture directory contains multiple input files for different test scenarios.
func RunBatchAutoWithInput(fixtureDir, inputFile string) func(*harness.Context) {
	return func(ctx *harness.Context) {
		cfg, err := harness.LoadFixtureConfig(fixtureDir)
		if err != nil {
			ctx.T.Fatalf("RunBatchAutoWithInput: load config: %v", err)
		}
		args := []string{
			"batch-auto",
			filepath.Join(ctx.WorkDir, inputFile),
			"--model", cfg.Model,
			"--threshold", fmt.Sprintf("%.2f", cfg.Threshold),
			"--top", fmt.Sprintf("%d", cfg.TopN),
			"--output-dir", ctx.WorkDir,
		}
		if ctx.DataDir != "" {
			args = append(args, "--data-dir", ctx.DataDir)
		}
		if ctx.WorkbookPath != "" {
			args = append(args, "--workbook", ctx.WorkbookPath)
		}
		args = append(args, cfg.ExtraArgs...)
		runCommand(ctx, args...)
		ctx.Artifacts["classified.csv"] = filepath.Join(ctx.WorkDir, "classified.csv")
		ctx.Artifacts["review.csv"] = filepath.Join(ctx.WorkDir, "review.csv")
		ctx.Artifacts["rollover.csv"] = filepath.Join(ctx.WorkDir, "rollover.csv")
	}
}

// RunBatchAutoIntoArtifactDir reads config from fixtureDir, then uses the artifact registered
// under outputDirKey as the --output-dir value. Registers classified.csv and review.csv in
// ctx.Artifacts pointing into that directory. Use when the output directory is set up in Given
// and its path is only known at runtime via ctx.Artifacts.
func RunBatchAutoIntoArtifactDir(fixtureDir, outputDirKey string) func(*harness.Context) {
	return func(ctx *harness.Context) {
		cfg, err := harness.LoadFixtureConfig(fixtureDir)
		if err != nil {
			ctx.T.Fatalf("RunBatchAutoIntoArtifactDir: load config: %v", err)
		}
		outDir, ok := ctx.Artifacts[outputDirKey]
		if !ok {
			ctx.T.Fatalf("RunBatchAutoIntoArtifactDir: artifact key %q not registered in Given", outputDirKey)
		}
		args := []string{
			"batch-auto",
			filepath.Join(ctx.WorkDir, "input.csv"),
			"--model", cfg.Model,
			"--threshold", fmt.Sprintf("%.2f", cfg.Threshold),
			"--top", fmt.Sprintf("%d", cfg.TopN),
			"--output-dir", outDir,
		}
		if ctx.DataDir != "" {
			args = append(args, "--data-dir", ctx.DataDir)
		}
		if ctx.WorkbookPath != "" {
			args = append(args, "--workbook", ctx.WorkbookPath)
		}
		args = append(args, cfg.ExtraArgs...)
		runCommand(ctx, args...)
		ctx.Artifacts["classified.csv"] = filepath.Join(outDir, "classified.csv")
		ctx.Artifacts["review.csv"] = filepath.Join(outDir, "review.csv")
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
