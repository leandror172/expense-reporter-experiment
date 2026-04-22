//go:build acceptance

package verify

import (
	"fmt"
	"os"
	"strconv"

	"github.com/stretchr/testify/assert"

	"expense-reporter/test/harness"
)

// CommandSucceeded asserts the command exited with code 0.
func CommandSucceeded() func(*harness.Context) {
	return func(ctx *harness.Context) {
		ctx.T.Helper()
		assert.Zero(ctx.T, ctx.ExitCode,
			"command should succeed (exit 0)\nstdout: %s\nstderr: %s", ctx.Stdout, ctx.Stderr)
	}
}

// CommandFailed asserts the command exited with a non-zero code.
func CommandFailed() func(*harness.Context) {
	return func(ctx *harness.Context) {
		ctx.T.Helper()
		assert.NotZero(ctx.T, ctx.ExitCode,
			"command should fail (non-zero exit)\nstdout: %s\nstderr: %s", ctx.Stdout, ctx.Stderr)
	}
}

// OutputFileExists asserts the artifact key maps to an existing file.
func OutputFileExists(artifactKey string) func(*harness.Context) {
	return func(ctx *harness.Context) {
		ctx.T.Helper()
		path, ok := ctx.Artifacts[artifactKey]
		if !assert.True(ctx.T, ok, "artifact %q not registered", artifactKey) {
			return
		}
		_, err := os.Stat(path)
		assert.NoError(ctx.T, err, "output file %q should exist at %s", artifactKey, path)
	}
}

// OutputFileHasRows asserts the CSV artifact has exactly n rows (including header).
func OutputFileHasRows(artifactKey string, n int) func(*harness.Context) {
	return func(ctx *harness.Context) {
		ctx.T.Helper()
		rows := readArtifact(ctx, artifactKey)
		if rows == nil {
			return
		}
		assert.Equal(ctx.T, n, len(rows),
			"%q should have %d rows (including header)", artifactKey, n)
	}
}

// OutputFileHasAtLeastRows asserts the CSV artifact has at least n rows.
func OutputFileHasAtLeastRows(artifactKey string, n int) func(*harness.Context) {
	return func(ctx *harness.Context) {
		ctx.T.Helper()
		rows := readArtifact(ctx, artifactKey)
		if rows == nil {
			return
		}
		assert.GreaterOrEqual(ctx.T, len(rows), n,
			"%q should have at least %d rows", artifactKey, n)
	}
}

// OutputFileHasColumns asserts every row in the CSV artifact has exactly n columns.
func OutputFileHasColumns(artifactKey string, n int) func(*harness.Context) {
	return func(ctx *harness.Context) {
		ctx.T.Helper()
		rows := readArtifact(ctx, artifactKey)
		if rows == nil {
			return
		}
		for i, row := range rows {
			assert.Len(ctx.T, row, n,
				"%q row %d should have %d columns", artifactKey, i, n)
		}
	}
}

// AllClassificationScoresValid asserts every data row in classified.csv has a
// confidence score in [0,1]. Skips the header row.
func AllClassificationScoresValid(artifactKey string) func(*harness.Context) {
	const confidenceCol = 5
	return func(ctx *harness.Context) {
		ctx.T.Helper()
		rows := readArtifact(ctx, artifactKey)
		if rows == nil {
			return
		}
		if len(rows) > 0 {
			rows = rows[1:] // skip header
		}
		for i, row := range rows {
			if !assert.Greater(ctx.T, len(row), confidenceCol,
				"%q row %d missing confidence column", artifactKey, i) {
				continue
			}
			v, err := strconv.ParseFloat(row[confidenceCol], 64)
			if !assert.NoError(ctx.T, err,
				"%q row %d: confidence %q is not a valid float", artifactKey, i, row[confidenceCol]) {
				continue
			}
			assert.True(ctx.T, harness.ConfidenceInRange(v),
				"%q row %d: confidence %.4f out of [0,1]", artifactKey, i, v)
		}
	}
}

// NoExpenseInBothFiles asserts no row appears in both artifact1 and artifact2.
func NoExpenseInBothFiles(artifact1, artifact2 string) func(*harness.Context) {
	return func(ctx *harness.Context) {
		ctx.T.Helper()
		rows1 := readArtifact(ctx, artifact1)
		rows2 := readArtifact(ctx, artifact2)
		if rows1 == nil || rows2 == nil {
			return
		}
		set := make(map[string]bool)
		for _, row := range rows1 {
			set[joinRow(row)] = true
		}
		for _, row := range rows2 {
			assert.False(ctx.T, set[joinRow(row)],
				"expense %v appears in both %q and %q", row, artifact1, artifact2)
		}
	}
}

// OutputContains asserts stdout+stderr contains substr.
func OutputContains(substr string, msgAndArgs ...interface{}) func(*harness.Context) {
	return func(ctx *harness.Context) {
		ctx.T.Helper()
		output := ctx.Stdout + ctx.Stderr
		msg := fmt.Sprintf("%q not found in command output\nstdout: %s\nstderr: %s",
			substr, ctx.Stdout, ctx.Stderr)
		if len(msgAndArgs) > 0 {
			msg = fmt.Sprintf("%s — %v\n%s", substr, msgAndArgs[0], msg)
		}
		assert.Contains(ctx.T, output, substr, msg)
	}
}

// OutputNotContains asserts stdout+stderr does NOT contain substr.
func OutputNotContains(substr string, msgAndArgs ...interface{}) func(*harness.Context) {
	return func(ctx *harness.Context) {
		ctx.T.Helper()
		output := ctx.Stdout + ctx.Stderr
		msg := fmt.Sprintf("%q unexpectedly found in command output\nstdout: %s\nstderr: %s",
			substr, ctx.Stdout, ctx.Stderr)
		if len(msgAndArgs) > 0 {
			msg = fmt.Sprintf("%s — %v\n%s", substr, msgAndArgs[0], msg)
		}
		assert.NotContains(ctx.T, output, substr, msg)
	}
}

// readArtifact resolves an artifact key to a file path and reads it as CSV.
func readArtifact(ctx *harness.Context, key string) [][]string {
	ctx.T.Helper()
	path, ok := ctx.Artifacts[key]
	if !assert.True(ctx.T, ok, "artifact %q not registered in ctx.Artifacts", key) {
		return nil
	}
	return harness.ReadCSVFile(ctx.T, path)
}

func joinRow(row []string) string {
	result := ""
	for i, f := range row {
		if i > 0 {
			result += ";"
		}
		result += f
	}
	return result
}
