//go:build replay

// Replay harness (T-23 / 5.R1) — retrieval half, Ollama-free.
//
// Replays the 649 human-verified real expenses in classifications.jsonl through the
// PRODUCTION few-shot retrieval path (SelectExamples + the real tokenizer/specificity
// scoring), leave-one-out, and emits per-item retrieval metrics as JSONL.
//
// It answers, in one pass and in seconds:
//   - 5.R1 TF-IDF trigger:   keyword miss rate (n_examples==0) + miss-reason breakdown
//   - E1 recurrence-strength: per-item top specificity AND matched-keyword frequency/idf
//     (specificity alone is misleading — a singleton token scores 1.0 but is NOVEL; the
//      recurrence axis is FREQUENCY, so both are emitted — advisor session 52)
//   - candidate/few-shot quality: retrieval_hit (did we surface a right-subcat example?)
//
// This is the CHEAP half. The model-accuracy half (join model output to top_score for the
// real gate risk–coverage) is a SEPARATE Ollama run — keyword_top1_correct here is a
// pure-keyword proxy, NOT the model gate.
//
// Leakage control: the 649 are themselves in the feedback pool, so each query's own entry
// is removed leave-one-out using the SAME dedup key as MergeExamplePools
// (strings.ToLower(strings.TrimSpace(item)), loader.go:180). self_match must stay 0.
//
// Run:  go test -tags=replay -run TestReplayRetrieval649 ./internal/classifier/ -v
// Paths are relative to the package dir; override via env if needed.
package classifier

import (
	"bufio"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func replayEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// rawKeyword holds the dict fields SelectExamples does not load (frequency, idf) so the
// probe can distinguish novel singletons (freq==1) from genuinely recurrent tokens.
type rawKeyword struct {
	Frequency     int      `json:"frequency"`
	Specificity   float64  `json:"specificity"`
	IDF           float64  `json:"idf"`
	Subcategories []string `json:"subcategories"`
}

func loadRawKeywords(t *testing.T, dataDir string) map[string]rawKeyword {
	t.Helper()
	data, err := os.ReadFile(dataDir + "/feature_dictionary_enhanced.json")
	if err != nil {
		t.Fatalf("reading feature dictionary: %v", err)
	}
	var raw struct {
		LexicalFeatures struct {
			Keywords map[string]rawKeyword `json:"keywords"`
		} `json:"lexical_features"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("parsing feature dictionary: %v", err)
	}
	return raw.LexicalFeatures.Keywords
}

// reviewRow is one line of the 649 human-verified labels.
type reviewRow struct {
	ID                string  `json:"id"`
	Item              string  `json:"item"`
	Date              string  `json:"date"`
	Value             float64 `json:"value"`
	ActualSubcategory string  `json:"actual_subcategory"`
	ActualCategory    string  `json:"actual_category"`
	Type              string  `json:"type"`
	Status            string  `json:"status"`
}

func loadReviewRows(t *testing.T, path string) []reviewRow {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("opening classifications.jsonl: %v", err)
	}
	defer f.Close()
	var rows []reviewRow
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 1<<20), 1<<20)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var r reviewRow
		if err := json.Unmarshal([]byte(line), &r); err != nil {
			t.Fatalf("parsing review row: %v", err)
		}
		rows = append(rows, r)
	}
	if err := sc.Err(); err != nil {
		t.Fatalf("scanning classifications.jsonl: %v", err)
	}
	return rows
}

// dedupKey mirrors MergeExamplePools's dedup key exactly (loader.go:180).
func dedupKey(item string) string { return strings.ToLower(strings.TrimSpace(item)) }

// retrievalResult is the per-item emitted record. Field names match the score/analysis
// convention (snake_case JSON) so downstream Python can consume it directly.
type retrievalResult struct {
	ID                string  `json:"id"`
	Item              string  `json:"item"`
	ActualSubcategory string  `json:"actual_subcategory"`
	ActualCategory    string  `json:"actual_category"`
	Type              string  `json:"type"`
	Value             float64 `json:"value"`
	Status            string  `json:"status"`

	NExamples int `json:"n_examples"` // 0 == miss

	// E1 recurrence-strength signals.
	TopScore       float64 `json:"top_score"`        // top specificity across matched tokens
	Top1Subcat     string  `json:"top1_subcat"`      // subcat the keyword layer would pick
	Top2Subcat     string  `json:"top2_subcat"`      // runner-up (ambiguous branch), "" if none
	Top2Score      float64 `json:"top2_score"`       // runner-up score, 0 if none
	DriverToken    string  `json:"driver_token"`     // matched token that set TopScore
	DriverFreq     int     `json:"driver_freq"`      // its corpus frequency (1 == novel singleton)
	DriverIDF      float64 `json:"driver_idf"`       // its inverse-doc-frequency (rarity)
	MaxMatchedFreq int     `json:"max_matched_freq"` // strongest recurrence evidence over all matched tokens
	NMatchedTokens int     `json:"n_matched_tokens"` // how many tokens hit the keyword index

	// Quality / correctness (few-shot INPUT quality, not the model's buttons).
	RetrievalHit       bool   `json:"retrieval_hit"`        // actual_subcat present among retrieved examples
	KeywordTop1Correct bool   `json:"keyword_top1_correct"` // top-scored subcat == actual (pure-keyword proxy)
	MissReason         string `json:"miss_reason"`          // "", "no_keyword_match", "no_examples_in_bucket"
	SelfMatch          bool   `json:"self_match"`           // leakage tripwire — must be false
}

func TestReplayRetrieval649(t *testing.T) {
	dataDir := replayEnv("REPLAY_DATA_DIR", "../../../data/classification")
	classPath := replayEnv("REPLAY_CLASS_PATH", "../../classifications.jsonl")
	outPath := replayEnv("REPLAY_OUT", "../../../.claude/scratch/replay-649/retrieval.jsonl")

	// Production pool + keyword index, wired exactly as classifier.go:95-106.
	training, err := LoadTrainingExamples(dataDir)
	if err != nil {
		t.Fatalf("loading training: %v", err)
	}
	feedback, err := LoadFeedbackExamples(classPath)
	if err != nil {
		t.Fatalf("loading feedback: %v", err)
	}
	pool := MergeExamplePools(training, feedback)
	keywords, err := LoadKeywordIndex(dataDir)
	if err != nil {
		t.Fatalf("loading keyword index: %v", err)
	}
	rawKW := loadRawKeywords(t, dataDir)
	rows := loadReviewRows(t, classPath)

	t.Logf("pool=%d training=%d feedback=%d keywords=%d rows=%d",
		len(pool), len(training), len(feedback), len(keywords), len(rows))

	out, err := os.Create(outPath)
	if err != nil {
		t.Fatalf("creating output: %v", err)
	}
	defer out.Close()
	enc := json.NewEncoder(out)

	const topK = 5 // matches classifier.go:106
	var (
		nMiss, nRetHit, nTop1, nSelf   int
		nNoKW, nNoBucket               int
		nRecurrentToks                 int // driver freq >= 2 (has recurrence evidence)
	)

	for _, r := range rows {
		key := dedupKey(r.Item)

		// Leave-one-out: drop the query's own entry from the pool (kills self-retrieval).
		loo := make([]Example, 0, len(pool))
		for _, ex := range pool {
			if dedupKey(ex.Item) == key {
				continue
			}
			loo = append(loo, ex)
		}

		// Faithful scoring: reuse the exact unexported tokenizer + scorer.
		tokens := tokenize(r.Item)
		scores := calculateSubcategoryScores(tokens, keywords)
		sorted := sortSubcategoriesByScore(scores)
		examples := SelectExamples(r.Item, loo, keywords, topK)

		res := retrievalResult{
			ID:                r.ID,
			Item:              r.Item,
			ActualSubcategory: r.ActualSubcategory,
			ActualCategory:    r.ActualCategory,
			Type:              r.Type,
			Value:             r.Value,
			Status:            r.Status,
			NExamples:         len(examples),
		}

		if len(sorted) > 0 {
			res.TopScore = sorted[0].score
			res.Top1Subcat = sorted[0].subcategory
			if len(sorted) > 1 {
				res.Top2Subcat = sorted[1].subcategory
				res.Top2Score = sorted[1].score
			}
			res.KeywordTop1Correct = sorted[0].subcategory == r.ActualSubcategory
		}

		// Matched-token stats: count matches and the strongest recurrence evidence.
		for _, tok := range tokens {
			kw, ok := rawKW[tok]
			if !ok {
				continue
			}
			res.NMatchedTokens++
			if kw.Frequency > res.MaxMatchedFreq {
				res.MaxMatchedFreq = kw.Frequency
			}
		}
		// Driver = matched token with highest specificity contributing to the winning
		// subcat; its frequency is the novelty/recurrence axis (freq==1 → novel singleton).
		res.DriverToken, res.DriverFreq, res.DriverIDF = pickDriver(tokens, rawKW, res.Top1Subcat)

		for _, ex := range examples {
			if ex.Subcategory == r.ActualSubcategory {
				res.RetrievalHit = true
			}
			if dedupKey(ex.Item) == key {
				res.SelfMatch = true
			}
		}

		switch {
		case len(examples) > 0:
			res.MissReason = ""
		case len(sorted) == 0:
			res.MissReason = "no_keyword_match"
			nNoKW++
		default:
			res.MissReason = "no_examples_in_bucket"
			nNoBucket++
		}

		if res.NExamples == 0 {
			nMiss++
		}
		if res.RetrievalHit {
			nRetHit++
		}
		if res.KeywordTop1Correct {
			nTop1++
		}
		if res.SelfMatch {
			nSelf++
		}
		if res.DriverFreq >= 2 {
			nRecurrentToks++
		}

		if err := enc.Encode(res); err != nil {
			t.Fatalf("encoding result: %v", err)
		}
	}

	n := len(rows)
	pct := func(x int) float64 { return 100 * float64(x) / float64(n) }
	t.Logf("=== RETRIEVAL REPLAY (per-row, n=%d, stream-weighted) ===", n)
	t.Logf("miss rate (n_examples==0):     %d (%.1f%%)   [5.R1 trigger: >10%%]", nMiss, pct(nMiss))
	t.Logf("  ├─ no_keyword_match:         %d (%.1f%%)   → embeddings/5.R2 lever", nNoKW, pct(nNoKW))
	t.Logf("  └─ no_examples_in_bucket:    %d (%.1f%%)", nNoBucket, pct(nNoBucket))
	t.Logf("retrieval_hit (right subcat surfaced): %d (%.1f%%)", nRetHit, pct(nRetHit))
	t.Logf("keyword_top1_correct (proxy):  %d (%.1f%%)", nTop1, pct(nTop1))
	t.Logf("driver_freq>=2 (recurrence evidence):  %d (%.1f%%)", nRecurrentToks, pct(nRecurrentToks))
	t.Logf("self_match (MUST be 0):        %d", nSelf)
	t.Logf("wrote %s", outPath)

	if nSelf != 0 {
		t.Errorf("leakage tripwire: %d self-matches — LOO key does not match MergeExamplePools", nSelf)
	}
}

func containsStr(ss []string, s string) bool {
	for _, x := range ss {
		if x == s {
			return true
		}
	}
	return false
}

// pickDriver returns the matched token with the highest specificity that contributes to
// the winning subcategory, along with its frequency and idf. It is the token that set
// TopScore; its frequency is the novelty/recurrence axis (freq==1 → novel singleton).
func pickDriver(tokens []string, rawKW map[string]rawKeyword, winSubcat string) (string, int, float64) {
	var (
		driver   string
		freq     int
		idf      float64
		bestSpec = -1.0
	)
	for _, tok := range tokens {
		kw, ok := rawKW[tok]
		if !ok {
			continue
		}
		if winSubcat != "" && !containsStr(kw.Subcategories, winSubcat) {
			continue
		}
		if kw.Specificity > bestSpec {
			bestSpec = kw.Specificity
			driver, freq, idf = tok, kw.Frequency, kw.IDF
		}
	}
	return driver, freq, idf
}
