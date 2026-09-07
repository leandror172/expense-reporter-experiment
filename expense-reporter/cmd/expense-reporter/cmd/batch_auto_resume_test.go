package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEvaluateResumeSkip(t *testing.T) {
	t.Run("single id present", func(t *testing.T) {
		ledger := map[string]int{"X": 1}
		predictedIDs := []string{"X"}

		skip, partial := evaluateResumeSkip(ledger, predictedIDs)

		assert.True(t, skip)
		assert.False(t, partial)
		require.Equal(t, 0, ledger["X"])
	})

	t.Run("single id absent", func(t *testing.T) {
		ledger := map[string]int{}
		predictedIDs := []string{"X"}

		skip, partial := evaluateResumeSkip(ledger, predictedIDs)

		assert.False(t, skip)
		assert.False(t, partial)
		require.Empty(t, ledger)
	})

	t.Run("all ids present", func(t *testing.T) {
		ledger := map[string]int{"A": 1, "B": 1, "C": 1}
		predictedIDs := []string{"A", "B", "C"}

		skip, partial := evaluateResumeSkip(ledger, predictedIDs)

		assert.True(t, skip)
		assert.False(t, partial)
		require.Equal(t, 0, ledger["A"])
		require.Equal(t, 0, ledger["B"])
		require.Equal(t, 0, ledger["C"])
	})

	t.Run("partial ids present", func(t *testing.T) {
		ledger := map[string]int{"A": 1, "B": 1}
		predictedIDs := []string{"A", "B", "C"}

		skip, partial := evaluateResumeSkip(ledger, predictedIDs)

		assert.False(t, skip)
		assert.True(t, partial)
		require.Equal(t, 1, ledger["A"])
		require.Equal(t, 1, ledger["B"])
	})

	t.Run("count consumption and duplicate handling", func(t *testing.T) {
		ledger := map[string]int{"X": 1}
		predictedIDs := []string{"X"}

		skip, partial := evaluateResumeSkip(ledger, predictedIDs)
		assert.True(t, skip)
		assert.False(t, partial)
		require.Equal(t, 0, ledger["X"])

		skip, partial = evaluateResumeSkip(ledger, predictedIDs)
		assert.False(t, skip)
		assert.False(t, partial)
	})
}

// TestRefuseDuplicateAppend pins the append-phase guard T-80 turned from a warning into an
// error. The behavior change worth pinning is that it does NOT consume the ledger: the
// classify-phase check owns consumption now, so a count still standing here means that
// check missed, and spending it would hide the miss instead of reporting it.
func TestRefuseDuplicateAppend(t *testing.T) {
	t.Run("a standing ledger count refuses the append without consuming it", func(t *testing.T) {
		ledger := map[string]int{"X": 1}

		err := refuseDuplicateAppend(ledger, "test item", []string{"X"})

		require.Error(t, err)
		assert.ErrorIs(t, err, errDuplicateEntry)
		assert.Contains(t, err.Error(), "X", "the offending id names the log line to look at")
		assert.Equal(t, 1, ledger["X"], "the guard reports; it must not spend the count it reported on")
	})

	t.Run("an id the ledger does not hold appends normally", func(t *testing.T) {
		ledger := map[string]int{}

		err := refuseDuplicateAppend(ledger, "test item", []string{"Y"})

		require.NoError(t, err)
		assert.Empty(t, ledger)
	})

	t.Run("one duplicate among a row's installment ids is enough to refuse", func(t *testing.T) {
		ledger := map[string]int{"i2": 1}

		err := refuseDuplicateAppend(ledger, "installment purchase", []string{"i1", "i2", "i3"})

		assert.ErrorIs(t, err, errDuplicateEntry,
			"a partially-logged series must not complete itself by appending the rest")
	})
}
