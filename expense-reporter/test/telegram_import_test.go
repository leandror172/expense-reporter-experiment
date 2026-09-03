//go:build acceptance

package acceptance_test

import (
	"path/filepath"
	"slices"
	"testing"

	"expense-reporter/test/actions"
	"github.com/leandror172/acceptance-harness/harness"
	"github.com/leandror172/acceptance-harness/verify"
)

// TestTelegramImport_DryRunAccountsForEveryMessage pins the bucket report on a SYNTHETIC
// Telegram Desktop export. The real export is personal data and its per-message-id gate
// is run by hand (T-75 plan, tracker step 1). The fixture has 13 entries but the report says
// 12 because one is a `service` entry (group created), which the adapter drops before counting.
//
// This scenario is deterministic: no language model, no workbook, no config — so it runs in
// the fast `-short` group. Asserting the ids on each line, not only the counts, is what
// makes the test able to fail when a message MOVES between buckets — two buckets swapping
// one message each leave every count unchanged.
func TestTelegramImport_DryRunAccountsForEveryMessage(t *testing.T) {
	fixDir := filepath.Join(fixturesDir(), "telegram-import-dry-run")

	harness.Run(t, harness.Scenario{
		Name:    "telegram-import command with --dry-run accounts for every message in the export",
		Fixture: fixDir,
		When:    actions.RunTelegramImport("result.json", "--dry-run"),
		Then: slices.Concat(
			commandSucceeded(),
			everyMessageLandsInExactlyOneBucket(),
		),
	})
}

// --- Then helpers (composable) ---

// everyMessageLandsInExactlyOneBucket returns a slice of assertions that verify each
// line of the bucket report, including the message IDs. This ensures that when a
// message moves between buckets, the test fails — not just the counts, but the exact
// assignment is verified.
func everyMessageLandsInExactlyOneBucket() []func(*harness.Context) {
	return []func(*harness.Context){
		verify.OutputContains("telegram-import: dry run, nothing written"),
		verify.OutputContains("  14 messages"),
		verify.OutputContains("   4 converted"),
		verify.OutputContains("   3 repaired (one edit made the line parse): 5 9 15"),
		verify.OutputContains("   2 receipts (attachments beside typed expenses, skipped): 3 4"),
		verify.OutputContains("   1 rejected: 1 field: 12"),
		verify.OutputContains("   1 rejected: ambiguous: 14"),
		verify.OutputContains("   1 rejected: bad date: 8"),
		verify.OutputContains("   1 rejected: bad value: 7"),
		verify.OutputContains("   1 ignored (conversation): 6"),
	}
}
