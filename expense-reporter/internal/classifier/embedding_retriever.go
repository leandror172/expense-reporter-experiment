package classifier

import (
	"math"
	"sort"
	"strings"
)

// cosineSimilarity computes the cosine similarity between two vectors.
func cosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) || len(a) == 0 || len(b) == 0 {
		return 0.0
	}

	dot := 0.0
	for i := range a {
		dot += a[i] * b[i]
	}

	magA := vectorNorm(a)
	magB := vectorNorm(b)
	if magA == 0 || magB == 0 {
		return 0.0
	}

	return dot / (magA * magB)
}

// vectorNorm computes the Euclidean norm of a vector.
func vectorNorm(v []float64) float64 {
	sum := 0.0
	for _, val := range v {
		sum += val * val
	}
	return math.Sqrt(sum)
}

// itemKey normalizes an item string for deduplication.
func itemKey(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

// TopKEmbeddingExamples ranks examples by cosine similarity to a query vector and returns top k distinct ones.
func TopKEmbeddingExamples(query string, queryVec []float64, pool []Example, store map[string][]float64, k int) []Example {
	if k <= 0 {
		return nil
	}

	queryKey := itemKey(query)
	var reps []Example
	idx := make(map[string]int)

	// Build ordered list of distinct examples (first-seen order), deduped by normalized key
	for _, e := range pool {
		key := itemKey(e.Item)
		if key == queryKey || !contains(store, e.Item) {
			continue
		}

		if pos, exists := idx[key]; exists {
			// Keep the one with higher source priority; if equal, keep first-seen
			if sourcePriority(e.Source) < sourcePriority(reps[pos].Source) {
				reps[pos] = e
			}
		} else {
			reps = append(reps, e)
			idx[key] = len(reps) - 1
		}
	}

	// Rank a permutation of indices so reps and similarities never desynchronize.
	// SliceStable over an index slice keeps first-seen order for equal-similarity ties.
	sims := make([]float64, len(reps))
	order := make([]int, len(reps))
	for i, e := range reps {
		sims[i] = cosineSimilarity(queryVec, store[e.Item])
		order[i] = i
	}
	sort.SliceStable(order, func(a, b int) bool {
		return sims[order[a]] > sims[order[b]]
	})

	ranked := make([]Example, len(order))
	for i, o := range order {
		ranked[i] = reps[o]
	}

	// Return the top k distinct examples.
	if len(ranked) < k {
		return ranked
	}
	return ranked[:k]
}

// contains checks if a key exists in the store map.
func contains(store map[string][]float64, key string) bool {
	_, ok := store[key]
	return ok
}
