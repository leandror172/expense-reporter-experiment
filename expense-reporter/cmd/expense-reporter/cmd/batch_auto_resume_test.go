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

func TestWarnIfDuplicate(t *testing.T) {
	t.Run("duplicate detected and consumed", func(t *testing.T) {
		ledger := map[string]int{"X": 1}
		item := "test item"

		result := warnIfDuplicate(ledger, item, "X")
		assert.True(t, result)
		require.Equal(t, 0, ledger["X"])

		result = warnIfDuplicate(ledger, item, "X")
		assert.False(t, result)
		require.Equal(t, 0, ledger["X"])
	})

	t.Run("no duplicate in empty ledger", func(t *testing.T) {
		ledger := map[string]int{}
		item := "test item"

		result := warnIfDuplicate(ledger, item, "Y")
		assert.False(t, result)
		require.Empty(t, ledger)
	})
}
