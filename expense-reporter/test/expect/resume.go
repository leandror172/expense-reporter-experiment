//go:build acceptance

package expect

import (
	"strings"

	"github.com/stretchr/testify/assert"

	"github.com/leandror172/acceptance-harness/harness"
)

// skipMarker is the stable substring batch-auto prints once per row skipped by --resume
// (e.g. "[1/2] SKIP  Netflix (already logged)").
const skipMarker = "already logged"

// duplicateWarningMarker is the stable substring of the always-on duplicate notice. Since
// T-80 it is emitted once per row HELD BACK from auto-insert because its id already existed
// in the expense log — not, as before, once per entry appended over one. The substring was
// kept across that behavior change on purpose, so the assertion counting it went on meaning
// something instead of silently counting a string nothing emits any more.
const duplicateWarningMarker = "already in expense log"

// ResumeSkipCount asserts that exactly n rows were reported as already logged (skipped) by
// --resume. Counts the skip marker across stdout+stderr, so it is robust to which stream the
// per-row markers land on. This is the deterministic heart of the one-of-two-duplicates
// scenario: skip-count is decided by the ledger, not by the model.
func ResumeSkipCount(n int) func(*harness.Context) {
	return func(ctx *harness.Context) {
		ctx.T.Helper()
		got := strings.Count(ctx.Stdout+ctx.Stderr, skipMarker)
		assert.Equal(ctx.T, n, got,
			"expected %d row(s) skipped as already-logged, got %d\nstdout: %s\nstderr: %s",
			n, got, ctx.Stdout, ctx.Stderr)
	}
}

// DuplicateWarningCount asserts the always-on duplicate notice appears exactly n times on
// stderr — one per row routed to review because its id was already present in the log. It
// must fire for a pre-existing log duplicate but NOT for a second in-CSV legitimate
// duplicate: the ledger models what is IN THE LOG, not what this run wrote, so intra-batch
// duplicates are deliberately invisible to it (out of T-80 scope, filed separately).
func DuplicateWarningCount(n int) func(*harness.Context) {
	return func(ctx *harness.Context) {
		ctx.T.Helper()
		got := strings.Count(ctx.Stderr, duplicateWarningMarker)
		assert.Equal(ctx.T, n, got,
			"expected %d duplicate-append warning(s) on stderr, got %d\nstderr: %s",
			n, got, ctx.Stderr)
	}
}
