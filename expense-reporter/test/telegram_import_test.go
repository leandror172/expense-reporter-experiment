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
// is run by hand (T-75 plan, tracker step 1). The fixture has 15 entries but the report says
// 14 because one is a `service` entry (group created), which the adapter drops before counting.
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

// TestTelegramImport_MultiLineListBecomesOneExpensePerLine pins plan D10 on a SYNTHETIC
// export. The trigger was real: the 2026-02 export's first run produced `rejected: 21
// fields` — one message holding ten well-formed expense lines, a backlog catch-up list,
// and flattening lost all ten at once.
//
// The scenario is built so the two ways of getting D10 wrong both go red.
//
//   - Message 2 splits into three lines of which ONE cannot parse (`99/99`). If the rule
//     were "split when every line PARSES" instead of "every line is 3-field-SHAPED", this
//     message would not split and `3 converted` would drop to 1. That is not hypothetical:
//     two of the ten real lines carry a `29/12/26` year typo.
//   - Message 3 is one expense plus a chatty line, so it must NOT split. Flattened it is
//     three fields with an unparseable value, which is why `rejected: bad value: 3` is
//     here — if D10 ever loosened to "SOME line is 3-field-shaped", that line would
//     convert and the human's note would vanish.
//
// `5 expense lines (1 multi-line list split)` is the line that distinguishes a split from
// a coincidence: the counts alone are also satisfied by three unsplit messages landing in
// three different buckets. It prints ONLY when something split, which is why every other
// scenario's output is unchanged.
//
// Deterministic — no model, no workbook, no config — so it runs in the `-short` group.
func TestTelegramImport_MultiLineListBecomesOneExpensePerLine(t *testing.T) {
	fixDir := filepath.Join(fixturesDir(), "telegram-import-multiline")

	harness.Run(t, harness.Scenario{
		Name:    "a multi-line message of 3-field lines is imported as one expense per line",
		Fixture: fixDir,
		When:    actions.RunTelegramImport("result.json", "--dry-run"),
		Then: slices.Concat(
			commandSucceeded(),
			eachListLineIsJudgedOnItsOwn(),
		),
	})
}

// --- Then helpers (composable) ---

// eachListLineIsJudgedOnItsOwn describes the D10 report: three messages read, five expense
// lines classified, and the split message's id in TWO buckets because its lines disagree.
func eachListLineIsJudgedOnItsOwn() []func(*harness.Context) {
	return []func(*harness.Context){
		verify.OutputContains("   3 messages"),
		verify.OutputContains("   5 expense lines (1 multi-line list split)"),
		verify.OutputContains("   3 converted"),
		verify.OutputContains("   1 rejected: bad date: 2"),
		verify.OutputContains("   1 rejected: bad value: 3"),
	}
}

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
