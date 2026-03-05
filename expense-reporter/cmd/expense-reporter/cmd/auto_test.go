package cmd

import (
	"strings"
	"testing"
)

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
