// Package classifier provides expense classification using local Ollama models,
// with few-shot example injection to improve accuracy.
package classifier

import (
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"
)

// ExampleSource indicates where the example came from.
type ExampleSource int

const (
	SourceTraining  ExampleSource = iota // from training_data_complete.json
	SourceConfirmed                      // from classifications.jsonl, status=confirmed
	SourceCorrected                      // from classifications.jsonl, status=corrected
)

// Example is a labeled expense used as a few-shot prompt example.
type Example struct {
	Item        string
	Value       float64
	Date        string // DD/MM format (normalized on load)
	Subcategory string
	Category    string
	Source      ExampleSource
}

// KeywordEntry is the index entry for one keyword token.
type KeywordEntry struct {
	DominantSubcategory string
	Specificity         float64  // 0.0–1.0; ratio of dominant_count/frequency
	Subcategories       []string // all subcategories this keyword appears in
}

// KeywordIndex maps lowercase token → KeywordEntry.
type KeywordIndex map[string]KeywordEntry

// nonAlphanumRe matches runs of characters that are not Unicode letters, digits, or spaces.
var nonAlphanumRe = regexp.MustCompile(`[^\p{L}\p{N} ]+`)

// SelectExamples returns up to topK examples from pool that best match the expense item.
//
// Algorithm:
//  1. Tokenize item: lowercase, strip non-letter/digit chars, split on whitespace,
//     keep tokens with rune length >= 2.
//  2. For each token, look it up in keywords; accumulate per-subcategory max-specificity.
//  3. High-specificity match (top score >= 0.7): examples from the dominant subcategory only.
//     Ambiguous match (top score < 0.7): interleave examples from the top-2 subcategories.
//  4. Within each bucket, sort: Corrected > Training > Confirmed.
//  5. Return up to topK. Returns nil when no keyword matches.
func SelectExamples(item string, pool []Example, keywords KeywordIndex, topK int) []Example {
	if topK <= 0 {
		return nil
	}

	tokens := tokenize(item)
	if len(tokens) == 0 {
		return nil
	}

	// Collect per-subcategory max specificity across all matched tokens.
	subcategoryScores := make(map[string]float64)
	for _, token := range tokens {
		entry, exists := keywords[token]
		if !exists {
			continue
		}
		for _, subcategory := range entry.Subcategories {
			if entry.Specificity > subcategoryScores[subcategory] {
				subcategoryScores[subcategory] = entry.Specificity
			}
		}
	}

	if len(subcategoryScores) == 0 {
		return nil
	}

	// Sort subcategories by specificity descending.
	type subcatScore struct {
		subcategory string
		score       float64
	}
	sorted := make([]subcatScore, 0, len(subcategoryScores))
	for subcat, score := range subcategoryScores {
		sorted = append(sorted, subcatScore{subcat, score})
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].score > sorted[j].score
	})

	var result []Example

	if sorted[0].score >= 0.7 {
		// High-specificity: examples from the single dominant subcategory.
		result = bucketExamples(pool, sorted[0].subcategory)
	} else {
		// Ambiguous: interleave examples from top-2 subcategories.
		limit := 2
		if len(sorted) < limit {
			limit = len(sorted)
		}
		buckets := make([][]Example, limit)
		for i := 0; i < limit; i++ {
			buckets[i] = bucketExamples(pool, sorted[i].subcategory)
		}
		// Round-robin interleave.
		maxLen := 0
		for _, b := range buckets {
			if len(b) > maxLen {
				maxLen = len(b)
			}
		}
		for i := 0; i < maxLen; i++ {
			for _, b := range buckets {
				if i < len(b) {
					result = append(result, b[i])
				}
			}
		}
	}

	if len(result) > topK {
		result = result[:topK]
	}
	return result
}

// tokenize lowercases item, strips non-alphanumeric chars, splits on whitespace,
// and returns tokens with rune length >= 2.
func tokenize(item string) []string {
	lower := strings.ToLower(item)
	cleaned := nonAlphanumRe.ReplaceAllString(lower, " ")
	fields := strings.Fields(cleaned)
	tokens := fields[:0]
	for _, f := range fields {
		if utf8.RuneCountInString(f) >= 2 {
			tokens = append(tokens, f)
		}
	}
	return tokens
}

// bucketExamples returns examples matching subcat, sorted by source priority
// (Corrected first, then Training, then Confirmed).
func bucketExamples(pool []Example, subcat string) []Example {
	var out []Example
	for _, ex := range pool {
		if ex.Subcategory == subcat {
			out = append(out, ex)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		return sourcePriority(out[i].Source) < sourcePriority(out[j].Source)
	})
	return out
}

// sourcePriority returns a sort key for ExampleSource (lower = higher priority).
func sourcePriority(src ExampleSource) int {
	switch src {
	case SourceCorrected:
		return 0
	case SourceTraining:
		return 1
	case SourceConfirmed:
		return 2
	default:
		return 3
	}
}
