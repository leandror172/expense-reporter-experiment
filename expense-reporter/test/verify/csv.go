//go:build acceptance

package verify

import (
	"os"
	"strconv"
	"strings"

	"expense-reporter/test/harness"
)

// StdoutContains returns a Then closure asserting ctx.Stdout contains substr.
func StdoutContains(substr string) func(*harness.Context) {
	return func(ctx *harness.Context) {
		ctx.T.Helper()
		if !strings.Contains(ctx.Stdout+ctx.Stderr, substr) {
			ctx.T.Errorf("StdoutContains(%q): not found in output\nstdout: %s\nstderr: %s",
				substr, ctx.Stdout, ctx.Stderr)
		}
	}
}

// StdoutNotContains returns a Then closure asserting ctx.Stdout does NOT contain substr.
func StdoutNotContains(substr string) func(*harness.Context) {
	return func(ctx *harness.Context) {
		ctx.T.Helper()
		if strings.Contains(ctx.Stdout+ctx.Stderr, substr) {
			ctx.T.Errorf("StdoutNotContains(%q): unexpectedly found in output\nstdout: %s\nstderr: %s",
				substr, ctx.Stdout, ctx.Stderr)
		}
	}
}

// ExitCodeZero returns a Then closure asserting ctx.ExitCode == 0.
func ExitCodeZero() func(*harness.Context) {
	return func(ctx *harness.Context) {
		ctx.T.Helper()
		if ctx.ExitCode != 0 {
			ctx.T.Errorf("ExitCodeZero: got exit code %d\nstdout: %s\nstderr: %s",
				ctx.ExitCode, ctx.Stdout, ctx.Stderr)
		}
	}
}

// FileExists returns a Then closure asserting the artifact key maps to an existing file.
func FileExists(artifactKey string) func(*harness.Context) {
	return func(ctx *harness.Context) {
		ctx.T.Helper()
		path, ok := ctx.Artifacts[artifactKey]
		if !ok {
			ctx.T.Errorf("FileExists: artifact key %q not registered", artifactKey)
			return
		}
		if _, err := os.Stat(path); os.IsNotExist(err) {
			ctx.T.Errorf("FileExists: file not found: %s", path)
		}
	}
}

// RowCount returns a Then closure asserting the CSV artifact has exactly n rows.
func RowCount(artifactKey string, n int) func(*harness.Context) {
	return func(ctx *harness.Context) {
		ctx.T.Helper()
		rows := readArtifact(ctx, artifactKey)
		if rows == nil {
			return
		}
		if len(rows) != n {
			ctx.T.Errorf("RowCount(%q): got %d rows, want %d", artifactKey, len(rows), n)
		}
	}
}

// RowCountAtLeast returns a Then closure asserting the CSV artifact has at least n rows.
func RowCountAtLeast(artifactKey string, n int) func(*harness.Context) {
	return func(ctx *harness.Context) {
		ctx.T.Helper()
		rows := readArtifact(ctx, artifactKey)
		if rows == nil {
			return
		}
		if len(rows) < n {
			ctx.T.Errorf("RowCountAtLeast(%q): got %d rows, want at least %d", artifactKey, len(rows), n)
		}
	}
}

// ColumnCount returns a Then closure asserting every row in the CSV artifact has exactly n columns.
func ColumnCount(artifactKey string, n int) func(*harness.Context) {
	return func(ctx *harness.Context) {
		ctx.T.Helper()
		rows := readArtifact(ctx, artifactKey)
		if rows == nil {
			return
		}
		for i, row := range rows {
			if len(row) != n {
				ctx.T.Errorf("ColumnCount(%q): row %d has %d columns, want %d", artifactKey, i, len(row), n)
			}
		}
	}
}

// AllConfidencesInRange returns a Then closure asserting every row's confidenceCol field is in [0,1].
func AllConfidencesInRange(artifactKey string, confidenceCol int) func(*harness.Context) {
	return func(ctx *harness.Context) {
		ctx.T.Helper()
		rows := readArtifact(ctx, artifactKey)
		if rows == nil {
			return
		}
		for i, row := range rows {
			if confidenceCol >= len(row) {
				ctx.T.Errorf("AllConfidencesInRange(%q): row %d has %d cols, want col %d",
					artifactKey, i, len(row), confidenceCol)
				continue
			}
			v, err := strconv.ParseFloat(row[confidenceCol], 64)
			if err != nil {
				ctx.T.Errorf("AllConfidencesInRange(%q): row %d col %d: parse %q: %v",
					artifactKey, i, confidenceCol, row[confidenceCol], err)
				continue
			}
			if !harness.ConfidenceInRange(v) {
				ctx.T.Errorf("AllConfidencesInRange(%q): row %d confidence %.4f out of [0,1]",
					artifactKey, i, v)
			}
		}
	}
}

// NoOverlap returns a Then closure asserting no row appears in both artifact1 and artifact2
// (compared by full row equality).
func NoOverlap(artifact1, artifact2 string) func(*harness.Context) {
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
			if set[joinRow(row)] {
				ctx.T.Errorf("NoOverlap: row %v appears in both %q and %q", row, artifact1, artifact2)
			}
		}
	}
}

// readArtifact resolves an artifact key to a file path and reads it as CSV.
// Returns nil (with t.Error) if key is missing or file unreadable.
func readArtifact(ctx *harness.Context, key string) [][]string {
	ctx.T.Helper()
	path, ok := ctx.Artifacts[key]
	if !ok {
		ctx.T.Errorf("readArtifact: key %q not registered in ctx.Artifacts", key)
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
