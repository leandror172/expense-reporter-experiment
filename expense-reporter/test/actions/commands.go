//go:build acceptance

package actions

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"

	"expense-reporter/test/harness"
)

// RunClassify returns a When closure that runs the classify command.
func RunClassify(args ...string) func(*harness.Context) {
	return func(ctx *harness.Context) {
		runCommand(ctx, append([]string{"classify"}, args...)...)
	}
}

// RunAuto returns a When closure that runs the auto command.
func RunAuto(args ...string) func(*harness.Context) {
	return func(ctx *harness.Context) {
		runCommand(ctx, append([]string{"auto"}, args...)...)
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
			"--dry-run",
			"--output-dir", ctx.WorkDir,
		}
		args = append(args, cfg.ExtraArgs...)
		runCommand(ctx, args...)
		ctx.Artifacts["classified.csv"] = filepath.Join(ctx.WorkDir, "classified.csv")
		ctx.Artifacts["review.csv"] = filepath.Join(ctx.WorkDir, "review.csv")
	}
}

// runCommand executes binary with args, capturing stdout/stderr/exitcode into ctx.
func runCommand(ctx *harness.Context, args ...string) {
	if len(args) == 0 {
		ctx.T.Fatal("runCommand: no args provided")
	}
	cmd := exec.Command(ctx.BinaryPath, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	ctx.Stdout = stdout.String()
	ctx.Stderr = stderr.String()
	if err == nil {
		ctx.ExitCode = 0
		return
	}
	if e, ok := err.(*exec.ExitError); ok {
		ctx.ExitCode = e.ExitCode()
	} else {
		ctx.T.Fatalf("runCommand %v: unexpected error: %v", args, err)
	}
}
