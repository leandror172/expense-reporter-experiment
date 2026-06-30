package cmd

import (
	"strings"
	"testing"
)

// Note: the former TestResolveExpenseType was removed in T-13. The command-layer
// resolveExpenseType wrapper is gone — the expense type now comes from the
// classifier's predicted full path. The underlying (category,subcategory)→type
// index (BuildTypeIndex/LookupType) is still covered by
// internal/taxonomy/lookup_test.go.

// --- confirmInsert (interactive prompt with reader) ---

func TestConfirmInsert_Yes(t *testing.T) {
	for _, input := range []string{"y\n", "Y\n", "yes\n", "YES\n"} {
		r := strings.NewReader(input)
		if !confirmInsert(r) {
			t.Errorf("confirmInsert(%q) = false, want true", input)
		}
	}
}

func TestConfirmInsert_No(t *testing.T) {
	for _, input := range []string{"n\n", "N\n", "no\n", "\n", "anything\n"} {
		r := strings.NewReader(input)
		if confirmInsert(r) {
			t.Errorf("confirmInsert(%q) = true, want false", input)
		}
	}
}
