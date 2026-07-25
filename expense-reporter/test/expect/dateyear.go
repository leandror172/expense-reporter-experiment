//go:build acceptance

package expect

import (
	"github.com/stretchr/testify/assert"

	"github.com/leandror172/acceptance-harness/harness"
)

// staleConfiguredYearMarker is the stable substring of the warning emitted when a
// configured date_year BELOW the current year is what actually dated an entry.
const staleConfiguredYearMarker = "is before the current year"

// StaleConfiguredYearWarned asserts the stale-date_year warning reached stderr.
// The stream matters, not just the text: --json writes its payload to stdout, so a
// warning on the wrong stream would corrupt machine-readable output rather than
// merely annoy. Asserting stdout+stderr combined would let that regression pass.
func StaleConfiguredYearWarned() func(*harness.Context) {
	return func(ctx *harness.Context) {
		ctx.T.Helper()
		assert.Contains(ctx.T, ctx.Stderr, staleConfiguredYearMarker,
			"expected the stale date_year warning on stderr\nstdout: %s\nstderr: %s",
			ctx.Stdout, ctx.Stderr)
	}
}

// NoStaleConfiguredYearWarning asserts the warning appears on NEITHER stream.
// Absence is checked across both on purpose: the claim is that nothing was said at
// all, which a stderr-only check would not establish.
func NoStaleConfiguredYearWarning() func(*harness.Context) {
	return func(ctx *harness.Context) {
		ctx.T.Helper()
		assert.NotContains(ctx.T, ctx.Stdout+ctx.Stderr, staleConfiguredYearMarker,
			"expected no stale date_year warning\nstdout: %s\nstderr: %s",
			ctx.Stdout, ctx.Stderr)
	}
}
