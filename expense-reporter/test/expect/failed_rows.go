//go:build acceptance

package expect

import (
	"bufio"
	"os"
	"strings"

	"github.com/leandror172/acceptance-harness/harness"
	"github.com/stretchr/testify/assert"
)

// FailedRowsCarryTheirReason asserts that failed.csv lists exactly wantRawLines, in
// order, each still carrying the reason it was rejected.
//
// It checks "begins with" rather than equality because the reason is appended to the
// SAME line as the data — that is the property that makes the file repairable in place,
// so asserting equality would forbid the very thing under test.
func FailedRowsCarryTheirReason(artifactKey string, wantRawLines []string) func(*harness.Context) {
	return func(ctx *harness.Context) {
		ctx.T.Helper()
		path, ok := ctx.Artifacts[artifactKey]
		if !ok {
			ctx.T.Fatalf("FailedRowsCarryTheirReason: artifact %q not registered", artifactKey)
		}
		got := failedDataLines(ctx, path, artifactKey)
		if !assert.Len(ctx.T, got, len(wantRawLines),
			"%s should list one line per rejected row", artifactKey) {
			return
		}
		for i, want := range wantRawLines {
			assertRowKeepsItsReason(ctx, artifactKey, i, want, got[i])
		}
	}
}

// failedDataLines returns the lines a re-import would consume: the '#' header block and
// blank lines are dropped, mirroring what batch.CSVReader does with the same file.
func failedDataLines(ctx *harness.Context, path, artifactKey string) []string {
	file, err := os.Open(path)
	if err != nil {
		ctx.T.Fatalf("FailedRowsCarryTheirReason: opening %s: %v", artifactKey, err)
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		// RIGHT trim only. The raw line is written verbatim, and an input line may
		// legitimately carry leading whitespace (" Uber ;15/04;35,50" is a covered
		// case in TestParse3FieldLine). Left-trimming here would break the caller's
		// HasPrefix/TrimPrefix against the expected raw line, and TrimPrefix fails
		// SILENTLY — it returns the whole string — so the test would report a
		// missing '#' instead of the mismatch that actually occurred.
		lines = append(lines, strings.TrimRight(line, " \t"))
	}
	if err := scanner.Err(); err != nil {
		ctx.T.Fatalf("FailedRowsCarryTheirReason: reading %s: %v", artifactKey, err)
	}
	return lines
}

// assertRowKeepsItsReason pins both halves of one line: the data the human will repair,
// and a non-empty explanation of why it was rejected. A row with no reason is not
// repairable, so an empty comment fails rather than passing as "present".
func assertRowKeepsItsReason(ctx *harness.Context, artifactKey string, i int, wantRaw, got string) {
	if !assert.True(ctx.T, strings.HasPrefix(got, wantRaw),
		"%s line %d should begin with the original input %q, got %q", artifactKey, i, wantRaw, got) {
		return
	}
	comment := strings.TrimSpace(strings.TrimPrefix(got, wantRaw))
	assert.True(ctx.T, strings.HasPrefix(comment, "#"),
		"%s line %d should carry its reason as a trailing comment, got %q", artifactKey, i, got)
	assert.NotEmpty(ctx.T, strings.TrimSpace(strings.TrimPrefix(comment, "#")),
		"%s line %d has an empty reason, so the row cannot be repaired", artifactKey, i)
}
