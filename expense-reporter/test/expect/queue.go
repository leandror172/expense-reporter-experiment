package expect

import (
	"strings"

	"github.com/leandror172/acceptance-harness/harness"
	"github.com/stretchr/testify/assert"
)

// unreviewableMarker prefixes the one stderr line `review` emits per row it left out of
// the queue. Counting the PER-ROW lines rather than the summary header is deliberate: the
// header carries the count as text, so asserting on it would pass even if the rows it
// claims were never actually named.
const unreviewableMarker = "unreviewable:"

// UnreviewableRowsReported asserts that `review` named exactly n rows as unreviewable on
// stderr — rows batch-auto could not parse, which it records in classified.csv with empty
// date/value cells.
//
// stderr specifically: --json owns stdout, and the review summary goes there too, so a
// warning on stdout would corrupt a machine-readable run.
func UnreviewableRowsReported(n int) func(*harness.Context) {
	return func(ctx *harness.Context) {
		ctx.T.Helper()
		got := strings.Count(ctx.Stderr, unreviewableMarker)
		assert.Equal(ctx.T, n, got,
			"expected review to name %d unreviewable row(s) on stderr, got %d\nstderr: %s",
			n, got, ctx.Stderr)
	}
}

// NoUnreviewableRowsReported asserts review stayed silent about unreviewable rows — the
// correct behavior when every row parsed, so a clean run is not noisy.
func NoUnreviewableRowsReported() func(*harness.Context) {
	return UnreviewableRowsReported(0)
}
