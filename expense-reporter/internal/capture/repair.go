package capture

// Plan D4 — repairs under T-64's exactly-one-reading rule, lifted one layer up.
//
// A message that fails to parse as typed gets a small, fixed set of SINGLE-EDIT
// candidates. A candidate counts only if the very same predicate that judged the
// original (parseLine) accepts it AND its date lies inside the D1 window of the message
// day. Accept when exactly ONE candidate qualifies; two is an ambiguity and therefore an
// error, never a guess (D4); nothing is composed (D6); and none of this runs for a line
// that already parses (D5, enforced in Classify). The converter and the parser then
// reject for the SAME reason, so the human learns one rule instead of two.

import (
	"strings"
	"time"

	"expense-reporter/internal/parse"
)

// repair is one candidate the line predicate accepted.
type repair struct {
	Text   string
	Parsed parse.ParsedExpense
}

// repairCandidates lists D4's single-edit repairs of text, in a fixed order, never
// including text itself: every ',' replaced — one at a time — by ';'; every '/'
// replaced — one at a time — by ';'; and, when the text has exactly four fields, the
// first two merged with one space (a stray ';' inside the item). Each candidate is
// exactly one edit away from the text (D6). Duplicates are not removed: two different
// edits that happen to yield the same string are still two readings.
func repairCandidates(text string) []string {
	var candidates []string

	// Replace each comma with semicolon
	candidates = append(candidates, replacingEach(text, ',')...)

	// Replace each slash with semicolon
	candidates = append(candidates, replacingEach(text, '/')...)

	// Merge first two fields if there are exactly four fields
	if merged, ok := mergedFirstTwoFields(text); ok {
		candidates = append(candidates, merged)
	}

	return candidates
}

// replacingEach returns a list of strings where each occurrence of old byte in text is replaced with ';' once.
func replacingEach(text string, old byte) []string {
	var result []string
	for i := 0; i < len(text); i++ {
		if text[i] == old {
			newText := text[:i] + ";" + text[i+1:]
			result = append(result, newText)
		}
	}
	return result
}

// mergedFirstTwoFields returns the merged first two fields if there are exactly four semicolon-separated parts.
func mergedFirstTwoFields(text string) (string, bool) {
	parts := strings.Split(text, ";")
	if len(parts) != 4 {
		return "", false
	}
	merged := strings.TrimSpace(parts[0]) + " " + strings.TrimSpace(parts[1]) + ";" + parts[2] + ";" + parts[3]
	return merged, true
}

// validRepairs runs the line predicate over each candidate and returns the ones the
// boundary accepted WHOSE RESOLVED DATE LIES INSIDE THE D1 WINDOW of the message day
// (daysFrom in [-windowPast, +windowFuture]), in candidate order.
// Measured s75 — "Dentista …; 21/08/ 1200,00": the comma edit reads the value's
// integer part as the year 1200, which the boundary ACCEPTS (1..9999, and the past side
// is unguarded by design, T-48), so without this rule a plain slash typo was ambiguous.
//
// SUBSUMED by the s75 step-5 D1 amendment, which applies the same window to EVERY
// resolved date: parseLine now refuses an out-of-window candidate itself, so the check
// below can no longer be the thing that rejects one. It is kept because the two rules
// have different justifications that merely share a threshold — D1's is "a date belongs
// near its message", D4's is "a guess needs more evidence than a statement" — and if D1's
// window were ever relaxed, D4 would still want its own. Delete it only together with a
// test that pins D4's rule at the D1 site.
func validRepairs(text string, sentAt, now time.Time) []repair {
	var valid []repair

	candidates := repairCandidates(text)
	for _, candidate := range candidates {
		parsed, err := parseLine(candidate, sentAt, now)
		if err != nil {
			continue
		}

		days := daysFrom(sentAt, parsed.Date)
		if days >= -windowPast && days <= windowFuture {
			valid = append(valid, repair{Text: candidate, Parsed: parsed})
		}
	}

	return valid
}

// repairedOrRejected decides D4's outcome for a text that failed AS TYPED: no valid
// repair keeps the as-typed error (the human sees the original diagnosis); exactly
// one valid repair converts, with Outcome.Repair set to the repaired text; more than
// one is an AmbiguousRepairError listing every reading that parsed.
func repairedOrRejected(m Message, text string, asTypedErr error, now time.Time) Outcome {
	valid := validRepairs(text, m.SentAt, now)

	switch len(valid) {
	case 0:
		return rejectedOutcome(m, asTypedErr)
	case 1:
		return repairedOutcome(m, valid[0])
	default:
		readings := make([]string, len(valid))
		for i, r := range valid {
			readings[i] = r.Text
		}
		return rejectedOutcome(m, AmbiguousRepairError{Readings: readings})
	}
}

// repairedOutcome is convertedOutcome plus the repaired text, so the report and the
// month file both know the line was touched.
func repairedOutcome(m Message, r repair) Outcome {
	out := convertedOutcome(m, r.Parsed)
	out.Repair = r.Text
	return out
}
