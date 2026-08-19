//go:build replay

// Replay harness (A1) — keyword-hint probe, Ollama-free.
//
// Decides whether the review UI should show the KEYWORD's answer alongside the model's.
//
// The first real monthly close (session 71) corrected 38% of reviewed rows, and the
// findings report measured that the keyword layer already held the reviewer's answer in
// ~60% of those corrections. That number came from a Python re-implementation of the
// tokenizer and was explicitly flagged as +/-1-2 ("a production replay would settle it").
// This IS that replay: it drives the real MatchStrength over a real reviewed.json export,
// so the counts are the ones production would have produced.
//
// The question is precision, not accuracy. A hint that fires on a handful of rows and is
// usually right is worth building; one that fires on most rows is noise wearing a badge.
// So every fired hint is split by what the reviewer actually did:
//   - fired on a CORRECTED row, keyword right -> a recovered error (the payoff)
//   - fired on a CONFIRMED row               -> pure noise (the reviewer kept the model's answer)
//
// Two thresholds run through one parameterized path: the strict 1.0 (unambiguous maximum
// specificity, the agreement gate's own bar) and the looser 0.7 that SelectExamples already
// treats as "high specificity".
//
// NOTE the direction of the predicate. It compares the keyword's answer to the MODEL's —
// never to the reviewer's. A predicate that consulted the reviewer's answer would need the
// thing it is trying to predict, and could not run at review time at all.
//
// Run:  go test -tags=replay -run TestReplayKeywordHint ./internal/classifier/ -v
// Paths are relative to the package dir; override via env.
package classifier

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
)

const (
	actionCorrected = "corrected"
	strictThreshold = 1.0
	looseThreshold  = 0.7
)

// taxonomyPath is one (type, category, subcategory) triple as reviewed.json carries it,
// under both the "predicted" and "reviewed" keys.
type taxonomyPath struct {
	Type        string `json:"type"`
	Category    string `json:"category"`
	Subcategory string `json:"subcategory"`
}

// reviewedRow is one entry of a reviewed.json export. Reviewed is a pointer because the
// contract does not require it on a confirmed row, even though the s71 export carries it
// on all 65 (verified: confirmed <=> reviewed.subcategory == predicted.subcategory).
type reviewedRow struct {
	ID        string        `json:"id"`
	Item      string        `json:"item"`
	Action    string        `json:"action"`
	Predicted taxonomyPath  `json:"predicted"`
	Reviewed  *taxonomyPath `json:"reviewed"`
}

type reviewedExport struct {
	Entries []reviewedRow `json:"entries"`
}

// thresholdOutcome is the predicate's verdict for one row at one strictness setting.
type thresholdOutcome struct {
	Threshold float64 `json:"threshold"`
	Fired     bool    `json:"fired"`
	Right     bool    `json:"right"`
}

// hintRow is the per-row JSONL record: the raw signal plus the outcome at each threshold,
// so the rows can be eyeballed and re-analysed without re-running the probe.
type hintRow struct {
	Item               string             `json:"item"`
	Action             string             `json:"action"`
	ModelSubcategory   string             `json:"model_subcategory"`
	FinalSubcategory   string             `json:"final_subcategory"`
	KeywordSubcategory string             `json:"keyword_subcategory"`
	TopScore           float64            `json:"top_score"`
	Matched            bool               `json:"matched"`
	Ambiguous          bool               `json:"ambiguous"`
	Outcomes           []thresholdOutcome `json:"outcomes"`
}

// hintSummary is the aggregate for one threshold.
type hintSummary struct {
	Threshold        float64
	Total            int
	Corrected        int
	Fired            int
	FiredRight       int
	FiredOnConfirmed int
	FiredOnCorrected int
	Recovered        int
}

func loadReviewedExport(t *testing.T, path string) reviewedExport {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading reviewed export %s: %v", path, err)
	}
	var export reviewedExport
	if err := json.Unmarshal(data, &export); err != nil {
		t.Fatalf("parsing reviewed export %s: %v", path, err)
	}
	if len(export.Entries) == 0 {
		t.Fatalf("reviewed export %s has no entries — a probe with no data must not report zeros", path)
	}
	return export
}

func loadProductionKeywords(t *testing.T, dataDir string) KeywordIndex {
	t.Helper()
	keywords, err := LoadKeywordIndex(dataDir)
	if err != nil {
		t.Fatalf("loading keyword index from %s: %v", dataDir, err)
	}
	return keywords
}

// finalSubcategory returns the leaf the reviewer settled on.
func finalSubcategory(row reviewedRow) string {
	if row.Reviewed != nil {
		return row.Reviewed.Subcategory
	}
	return row.Predicted.Subcategory
}

// hintFires reports whether the review UI would surface the keyword's answer: an
// unambiguous match at or above the threshold that DISAGREES with the model. Agreement
// has nothing to show.
func hintFires(signal MatchSignal, modelSubcategory string, threshold float64) bool {
	if !signal.Matched || signal.Ambiguous {
		return false
	}
	if signal.TopScore < threshold {
		return false
	}
	return signal.TopSubcategory != modelSubcategory
}

func outcomesFor(signal MatchSignal, modelSubcategory, final string, thresholds []float64) []thresholdOutcome {
	outcomes := make([]thresholdOutcome, 0, len(thresholds))
	for _, threshold := range thresholds {
		fired := hintFires(signal, modelSubcategory, threshold)
		outcomes = append(outcomes, thresholdOutcome{
			Threshold: threshold,
			Fired:     fired,
			Right:     fired && signal.TopSubcategory == final,
		})
	}
	return outcomes
}

func buildHintRow(row reviewedRow, keywords KeywordIndex, thresholds []float64) hintRow {
	signal := MatchStrength(row.Item, keywords)
	final := finalSubcategory(row)
	return hintRow{
		Item:               row.Item,
		Action:             row.Action,
		ModelSubcategory:   row.Predicted.Subcategory,
		FinalSubcategory:   final,
		KeywordSubcategory: signal.TopSubcategory,
		TopScore:           signal.TopScore,
		Matched:            signal.Matched,
		Ambiguous:          signal.Ambiguous,
		Outcomes:           outcomesFor(signal, row.Predicted.Subcategory, final, thresholds),
	}
}

// outcomeAt finds the precomputed outcome for one threshold. Exact float comparison is
// safe here: both sides come from the same thresholds slice, never from arithmetic.
func outcomeAt(row hintRow, threshold float64) thresholdOutcome {
	for _, outcome := range row.Outcomes {
		if outcome.Threshold == threshold {
			return outcome
		}
	}
	return thresholdOutcome{Threshold: threshold}
}

func (s *hintSummary) accumulate(row hintRow, outcome thresholdOutcome) {
	if row.Action == actionCorrected {
		s.Corrected++
	}
	if !outcome.Fired {
		return
	}
	s.Fired++
	if outcome.Right {
		s.FiredRight++
	}
	if row.Action != actionCorrected {
		s.FiredOnConfirmed++
		return
	}
	s.FiredOnCorrected++
	if outcome.Right {
		s.Recovered++
	}
}

func summarize(rows []hintRow, threshold float64) hintSummary {
	summary := hintSummary{Threshold: threshold, Total: len(rows)}
	for _, row := range rows {
		summary.accumulate(row, outcomeAt(row, threshold))
	}
	return summary
}

func pct(numerator, denominator int) string {
	if denominator == 0 {
		return "n/a"
	}
	return fmt.Sprintf("%.1f%%", 100*float64(numerator)/float64(denominator))
}

func formatSummary(s hintSummary) string {
	var b strings.Builder
	fmt.Fprintf(&b, "keyword hint at specificity >= %.2f\n", s.Threshold)
	fmt.Fprintf(&b, "  fired            %d of %d rows (%s)\n", s.Fired, s.Total, pct(s.Fired, s.Total))
	fmt.Fprintf(&b, "    keyword right  %d (%s of fired)\n", s.FiredRight, pct(s.FiredRight, s.Fired))
	fmt.Fprintf(&b, "    on CORRECTED   %d (%s of fired) — recovered %d\n",
		s.FiredOnCorrected, pct(s.FiredOnCorrected, s.Fired), s.Recovered)
	fmt.Fprintf(&b, "    on CONFIRMED   %d (%s of fired) — noise\n",
		s.FiredOnConfirmed, pct(s.FiredOnConfirmed, s.Fired))
	fmt.Fprintf(&b, "  recall           %d of %d corrections caught (%s)\n",
		s.Recovered, s.Corrected, pct(s.Recovered, s.Corrected))
	return b.String()
}

func writeHintRows(t *testing.T, path string, rows []hintRow) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("creating %s: %v", path, err)
	}
	defer f.Close()
	encoder := json.NewEncoder(f)
	for _, row := range rows {
		if err := encoder.Encode(row); err != nil {
			t.Fatalf("writing hint row: %v", err)
		}
	}
}

// assertStrictOutcomeMatchesProduction ties this probe to the SHIPPED predicate.
//
// The numbers this probe reports are only worth quoting if the thing measured is the thing
// that actually runs. hintFires below is the probe's own parameterized form (it sweeps a
// threshold, which production does not), so at the strict threshold it must agree with
// classifier.KeywordHint on every row — both on whether to fire and on what to say.
//
// A disagreement means one of the two moved. Fail loudly rather than publishing a number
// measured against code nobody ships.
func assertStrictOutcomeMatchesProduction(t *testing.T, entry reviewedRow, keywords KeywordIndex, row hintRow) {
	t.Helper()
	signal := MatchStrength(entry.Item, keywords)
	hint := KeywordHint(signal, entry.Predicted.Subcategory)
	strict := outcomeAt(row, strictThreshold)

	if strict.Fired != (hint != "") {
		t.Fatalf("probe and production disagree on %q: probe fired=%v but KeywordHint returned %q",
			entry.Item, strict.Fired, hint)
	}
	if hint != "" && hint != row.KeywordSubcategory {
		t.Fatalf("production hint %q differs from the probe's keyword %q for %q",
			hint, row.KeywordSubcategory, entry.Item)
	}
}

func TestReplayKeywordHint(t *testing.T) {
	reviewedJSON := replayEnv("REPLAY_REVIEWED_JSON", "../../../.claude/scratch/close-20260818-120248/reviewed.json")
	dataDir := replayEnv("REPLAY_DATA_DIR", "../../../data/classification")
	outPath := replayEnv("REPLAY_HINT_OUT", strings.TrimSuffix(reviewedJSON, ".json")+"-keyword-hint.jsonl")

	export := loadReviewedExport(t, reviewedJSON)
	keywords := loadProductionKeywords(t, dataDir)

	thresholds := []float64{strictThreshold, looseThreshold}
	rows := make([]hintRow, 0, len(export.Entries))
	for _, entry := range export.Entries {
		row := buildHintRow(entry, keywords, thresholds)
		assertStrictOutcomeMatchesProduction(t, entry, keywords, row)
		rows = append(rows, row)
	}
	writeHintRows(t, outPath, rows)

	t.Logf("keyword-hint probe — %d reviewed rows; per-row detail in %s", len(rows), outPath)
	for _, threshold := range thresholds {
		t.Logf("\n%s", formatSummary(summarize(rows, threshold)))
	}
}
