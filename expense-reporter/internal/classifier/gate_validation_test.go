//go:build replay

// Behavioral acceptance for T-32 (agreement gate). Recomputes the SHIPPED gate
// (MatchStrength + IsAutoInsertable) over the FROZEN 649-replay predictions in
// model.jsonl — NO re-inference, deterministic, runs in well under a second — and
// reports coverage/precision.
//
// Why this exists: the agreement-gate design rests on FINDINGS-model.md's ~95% subcat
// precision / ~31% coverage at spec=1.0 ∧ model⊕keyword agreement. That number came
// from an offline analysis, not the shipped code. This test confirms the shipped
// predicate reproduces it, and cross-checks that MatchStrength's TopScore is byte-equal
// to the top_score the replay harness stored — i.e. the shipped signal IS the measured
// signal. If they diverge, the predicate ≠ the measurement and T-32 is not done.
//
// The shipped gate is STRICTER than the raw finding (it also drops Ambiguous ties and
// applies the Diversos exclusion), so its coverage is ≤ and its precision ≥ the raw
// spec=1.0∧agreement band.
//
// Run: go test -tags=replay -run TestValidateAgreementGate649 ./internal/classifier/ -v
package classifier

import (
	"bufio"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestValidateAgreementGate649(t *testing.T) {
	dataDir := replayEnv("REPLAY_DATA_DIR", "../../../data/classification")
	modelPath := replayEnv("REPLAY_MODEL_OUT", "../../../.claude/scratch/replay-649/model.jsonl")
	excluded := []string{"Diversos"} // matches config.json auto_insert_excluded (the real auto path)

	keywords, err := LoadKeywordIndex(dataDir)
	if err != nil {
		t.Fatalf("loading keyword index: %v", err)
	}

	f, err := os.Open(modelPath)
	if err != nil {
		t.Fatalf("opening model.jsonl (run TestReplayModel649 first): %v", err)
	}
	defer f.Close()

	var total, topScoreMismatch int
	var gateN, gateSubcatOK, gateFullOK int // shipped agreement gate
	var bandN, bandFullOK int               // pure top_score>=1.0 band (cross-checks stored top_score)

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 1<<20), 1<<20)
	for sc.Scan() {
		if strings.TrimSpace(sc.Text()) == "" {
			continue
		}
		var r modelResult
		if err := json.Unmarshal(sc.Bytes(), &r); err != nil {
			t.Fatalf("decoding model.jsonl: %v", err)
		}
		total++

		signal := MatchStrength(r.Item, keywords)
		if signal.TopScore != r.TopScore {
			topScoreMismatch++
		}
		if signal.TopScore >= 1.0 {
			bandN++
			if r.CorrectFullPath {
				bandFullOK++
			}
		}

		result := Result{Type: r.PredType, Category: r.PredCat, Subcategory: r.PredSubcat}
		if IsAutoInsertable(result, signal, excluded) {
			gateN++
			if r.CorrectSubcat {
				gateSubcatOK++
			}
			if r.CorrectFullPath {
				gateFullOK++
			}
		}
	}
	if err := sc.Err(); err != nil {
		t.Fatalf("scanning model.jsonl: %v", err)
	}

	pct := func(a, b int) float64 {
		if b == 0 {
			return 0
		}
		return 100 * float64(a) / float64(b)
	}

	t.Logf("rows=%d   shipped-vs-stored top_score mismatches=%d", total, topScoreMismatch)
	t.Logf("pure top_score>=1.0 band : coverage %d/%d (%.1f%%)   full-path precision %.1f%%   [FINDINGS: 34.2%% / 86.9%%]",
		bandN, total, pct(bandN, total), pct(bandFullOK, bandN))
	t.Logf("SHIPPED AGREEMENT GATE   : coverage %d/%d (%.1f%%)   subcat precision %.1f%%   full-path precision %.1f%%   [FINDINGS agreement: ~31%% / ~95%% subcat]",
		gateN, total, pct(gateN, total), pct(gateSubcatOK, gateN), pct(gateFullOK, gateN))

	// Hard checks — these are the acceptance gates.
	if total == 0 {
		t.Fatalf("model.jsonl is empty — run TestReplayModel649 first")
	}
	if topScoreMismatch != 0 {
		t.Errorf("shipped MatchStrength.TopScore diverges from the harness-stored top_score on %d/%d rows — the shipped predicate is NOT the measured predicate", topScoreMismatch, total)
	}
	if gateN == 0 {
		t.Fatalf("agreement gate admitted 0 rows — signal wiring is broken")
	}
	if p := pct(gateSubcatOK, gateN); p < 90 {
		t.Errorf("agreement-gate subcat precision %.1f%% is below the expected ~95%% band (FINDINGS-model.md) — predicate may have regressed", p)
	}
}
