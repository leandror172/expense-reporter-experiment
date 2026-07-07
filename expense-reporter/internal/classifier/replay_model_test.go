//go:build replay

// Replay harness — MODEL accuracy half (T-23 gate). Ollama-bound (~16 min no-think).
//
// Replays the 649 human-verified real expenses through the ACTUAL classifier
// (classifier.Classify, in-process, with the exact `auto`/`batch-auto` config:
// FeedbackPath set + T-22 type descriptions applied), leave-one-out, and joins each
// predicted full path to the retrieval-half signal (top_score) and the true label.
//
// This is what the retrieval half could NOT do: the REAL (model, not keyword)
// risk–coverage curve of the E1 specificity gate — the measurement that decides T-23
// gate readiness → WS-D.
//
// Faithfulness / leakage:
//   - Feedback LOO is by ITEM (all same-item copies removed — the 649 hold only 260
//     unique items, heavy recurrence), so the query's own review decision cannot leak.
//   - Training is KEPT (a recurring item legitimately has prior instances in production);
//     the 62 same-subcat training overlaps are reported by stratifying clean vs leaky
//     (in_training), NOT excluded — excluding them would understate the recurrence gate.
//   - no-think (per approved budget); production default is think-on (~+3pp level shift).
//
// Resumable: appends to model.jsonl, skips ids already present.
//
// Run:  go test -tags=replay -run TestReplayModel649 ./internal/classifier/ -v -timeout 60m
package classifier

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"expense-reporter/internal/taxonomy"
)

func parseFloatSafe(s string) float64 {
	if s == "" {
		return 0
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return f
}

func atoiSafe(s string) int {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return n
		}
		n = n*10 + int(c-'0')
	}
	return n
}

type rawReviewLine struct {
	Key  string // dedupKey(item)
	Line string // verbatim JSONL line (preserves all fields LoadFeedbackExamples reads)
}

func loadRawReviewLines(t *testing.T, path string) []rawReviewLine {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("opening feedback for LOO: %v", err)
	}
	defer f.Close()
	var out []rawReviewLine
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 1<<20), 1<<20)
	for sc.Scan() {
		line := sc.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}
		var probe struct {
			Item string `json:"item"`
		}
		if err := json.Unmarshal([]byte(line), &probe); err != nil {
			t.Fatalf("probing feedback line: %v", err)
		}
		out = append(out, rawReviewLine{Key: dedupKey(probe.Item), Line: line})
	}
	return out
}

// trainingItemKeys returns the set of dedupKeys present in training_data_complete.json,
// used to stratify results clean (novel to training) vs leaky (recurs in training).
func trainingItemKeys(t *testing.T, dataDir string) map[string]bool {
	t.Helper()
	ex, err := LoadTrainingExamples(dataDir)
	if err != nil {
		t.Fatalf("loading training for stratification: %v", err)
	}
	keys := make(map[string]bool, len(ex))
	for _, e := range ex {
		keys[dedupKey(e.Item)] = true
	}
	return keys
}

type modelResult struct {
	ID     string  `json:"id"`
	Item   string  `json:"item"`
	Status string  `json:"status"`
	Value  float64 `json:"value"`

	ActualType   string `json:"actual_type"`
	ActualCat    string `json:"actual_category"`
	ActualSubcat string `json:"actual_subcategory"`
	PredType     string `json:"pred_type"`
	PredCat      string `json:"pred_category"`
	PredSubcat   string `json:"pred_subcategory"`

	Confidence float64 `json:"confidence"` // model self-reported (known anti-informative)
	TopScore   float64 `json:"top_score"`  // E1 specificity (external validator)
	DriverFreq int     `json:"driver_freq"`
	InTraining bool    `json:"in_training"` // leaky stratum

	CorrectSubcat   bool `json:"correct_subcat"`
	CorrectCat      bool `json:"correct_category"`
	CorrectType     bool `json:"correct_type"`
	CorrectFullPath bool `json:"correct_fullpath"`
}

func TestReplayModel649(t *testing.T) {
	dataDir := replayEnv("REPLAY_DATA_DIR", "../../../data/classification")
	classPath := replayEnv("REPLAY_CLASS_PATH", "../../classifications.jsonl")
	taxPath := replayEnv("REPLAY_TAX_PATH", "../../config/taxonomy.json")
	descPath := replayEnv("REPLAY_DESC_PATH", "../../config/type-descriptions.json")
	outPath := replayEnv("REPLAY_MODEL_OUT", "../../../.claude/scratch/replay-649/model.jsonl")

	// Taxonomy + T-22 descriptions, exactly as loadTaxonomyTree (classify.go).
	sheets, _, err := taxonomy.LoadTaxonomy(taxPath, "", "", 0)
	if err != nil {
		t.Fatalf("loading taxonomy: %v", err)
	}
	if desc, derr := taxonomy.LoadTypeDescriptions(descPath); derr == nil {
		taxonomy.ApplyDescriptions(sheets, desc)
	}

	keywords, err := LoadKeywordIndex(dataDir)
	if err != nil {
		t.Fatalf("loading keyword index: %v", err)
	}
	rawKW := loadRawKeywords(t, dataDir)
	rows := loadReviewRows(t, classPath)
	rawLines := loadRawReviewLines(t, classPath)
	trainKeys := trainingItemKeys(t, dataDir)

	// Resume: collect ids already written.
	done := map[string]bool{}
	if f, oerr := os.Open(outPath); oerr == nil {
		sc := bufio.NewScanner(f)
		sc.Buffer(make([]byte, 0, 1<<20), 1<<20)
		for sc.Scan() {
			var r struct {
				ID string `json:"id"`
			}
			if json.Unmarshal(sc.Bytes(), &r) == nil && r.ID != "" {
				done[r.ID] = true
			}
		}
		f.Close()
	}

	out, err := os.OpenFile(outPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatalf("opening output: %v", err)
	}
	defer out.Close()
	enc := json.NewEncoder(out)

	looPath := filepath.Join(os.TempDir(), "replay_loo_feedback.jsonl")
	t.Logf("rows=%d unique-training-keys=%d already-done=%d", len(rows), len(trainKeys), len(done))

	limit := 0
	if v := os.Getenv("REPLAY_LIMIT"); v != "" {
		limit = atoiSafe(v)
	}
	minTop := parseFloatSafe(os.Getenv("REPLAY_MIN_TOPSCORE")) // 0 = no band filter
	nothink := os.Getenv("REPLAY_THINK") == ""                 // think-on when REPLAY_THINK set
	t.Logf("config: min_topscore=%.2f nothink=%v", minTop, nothink)

	processed := 0
	for _, r := range rows {
		if done[r.ID] {
			continue
		}

		// External validator signal (recompute faithfully; CHEAP, no Ollama) — computed
		// BEFORE Classify so we can band-filter (think-on top-band re-run) without paying
		// the inference cost on rows we skip.
		tokens := tokenize(r.Item)
		sorted := sortSubcategoriesByScore(calculateSubcategoryScores(tokens, keywords))
		var topScore float64
		var winSub string
		if len(sorted) > 0 {
			topScore = sorted[0].score
			winSub = sorted[0].subcategory
		}
		_, driverFreq, _ := pickDriver(tokens, rawKW, winSub)

		if topScore < minTop { // band filter (REPLAY_MIN_TOPSCORE)
			continue
		}
		if limit > 0 && processed >= limit {
			break
		}
		key := dedupKey(r.Item)

		// Feedback LOO by item: write a temp feedback file excluding all same-item copies.
		var b strings.Builder
		for _, rl := range rawLines {
			if rl.Key == key {
				continue
			}
			b.WriteString(rl.Line)
			b.WriteByte('\n')
		}
		if err := os.WriteFile(looPath, []byte(b.String()), 0644); err != nil {
			t.Fatalf("writing LOO feedback: %v", err)
		}

		cfg := Config{
			Model:        "my-classifier-q3",
			DataDir:      dataDir,
			FeedbackPath: looPath,
			TopN:         3,
			NoThink:      nothink,
		}
		results, cerr := Classify(r.Item, r.Value, r.Date, sheets, cfg)
		if cerr != nil {
			t.Fatalf("classify %q: %v", r.Item, cerr)
		}
		if len(results) == 0 {
			t.Logf("WARN empty result for %q — skipping", r.Item)
			continue
		}
		top := results[0]

		res := modelResult{
			ID:           r.ID,
			Item:         r.Item,
			Status:       r.Status,
			Value:        r.Value,
			ActualType:   r.Type,
			ActualCat:    r.ActualCategory,
			ActualSubcat: r.ActualSubcategory,
			PredType:     top.Type,
			PredCat:      top.Category,
			PredSubcat:   top.Subcategory,
			Confidence:   top.Confidence,
			TopScore:     topScore,
			DriverFreq:   driverFreq,
			InTraining:   trainKeys[key],
		}
		res.CorrectSubcat = top.Subcategory == r.ActualSubcategory
		res.CorrectCat = top.Category == r.ActualCategory
		res.CorrectType = top.Type == r.Type
		res.CorrectFullPath = res.CorrectSubcat && res.CorrectCat && res.CorrectType

		if err := enc.Encode(res); err != nil {
			t.Fatalf("encoding: %v", err)
		}
		processed++
		if processed%50 == 0 {
			t.Logf("... %d processed", processed)
		}
	}
	t.Logf("done: %d newly processed, output %s", processed, outPath)
}
