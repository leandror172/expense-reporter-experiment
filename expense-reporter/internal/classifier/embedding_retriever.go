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

	candidates := dedupePoolByKey(query, pool, store)
	ranked := rankBySimilarity(queryVec, candidates, store)

	if len(ranked) < k {
		return ranked
	}
	return ranked[:k]
}

// dedupePoolByKey builds the ordered list of distinct candidate examples
// (first-seen order), deduped by normalized key (itemKey — the disk cache is
// keyed by exact raw item text instead; two keys by design). The query itself
// is excluded (leave-one-out), as is any item missing from the vector store.
// Among same-key rows the one with higher source priority wins; if equal,
// first-seen is kept.
func dedupePoolByKey(query string, pool []Example, store map[string][]float64) []Example {
	queryKey := itemKey(query)
	var candidates []Example
	idx := make(map[string]int)

	for _, e := range pool {
		key := itemKey(e.Item)
		if key == queryKey || !contains(store, e.Item) {
			continue
		}

		if pos, exists := idx[key]; exists {
			if sourcePriority(e.Source) < sourcePriority(candidates[pos].Source) {
				candidates[pos] = e
			}
		} else {
			candidates = append(candidates, e)
			idx[key] = len(candidates) - 1
		}
	}

	return candidates
}

// rankBySimilarity orders candidates by descending cosine similarity to the
// query vector. It ranks a permutation of indices so candidates and
// similarities never desynchronize; SliceStable over the index slice keeps
// first-seen order for equal-similarity ties — guaranteed to occur, because
// training duplicates share identical vectors — so ranking is deterministic.
func rankBySimilarity(queryVec []float64, candidates []Example, store map[string][]float64) []Example {
	sims := make([]float64, len(candidates))
	order := make([]int, len(candidates))
	for i, e := range candidates {
		sims[i] = cosineSimilarity(queryVec, store[e.Item])
		order[i] = i
	}
	sort.SliceStable(order, func(a, b int) bool {
		return sims[order[a]] > sims[order[b]]
	})

	ranked := make([]Example, len(order))
	for i, o := range order {
		ranked[i] = candidates[o]
	}
	return ranked
}

// contains checks if a key exists in the store map.
func contains(store map[string][]float64, key string) bool {
	_, ok := store[key]
	return ok
}
