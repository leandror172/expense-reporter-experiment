package classifier

// IsExcluded reports whether a subcategory is in the exclusion list.
func IsExcluded(subcategory string, excluded []string) bool {
	for _, ex := range excluded {
		if subcategory == ex {
			return true
		}
	}
	return false
}

// IsAutoInsertable determines if a result meets the criteria to be considered auto-insertable.
//
// This is the agreement gate: high keyword specificity AND model⊕keyword concurrence.
// It deliberately drops the old confidence check because confidence was measured to be
// uninformative on real data (session-52 replay — the whole 0.85–0.95 band is a coin
// flip). The gate is intentionally high-precision / low-coverage — non-passing rows
// (keyword miss, ambiguous match, sub-maximal specificity, or model/keyword
// disagreement) go to manual review, they are not silently inserted.
//
// Caveat (T-32): agreement is verified at the SUBCATEGORY level, but the appended row
// carries the model's full path (Type/Category). For the 5 cross-type collision leaves
// (Estacionamento/Dentista/Orion/Lilly/Ambos) a subcat-right row can still be
// type-wrong and auto-append. Accepted as a known caveat; revisit if it bites.
func IsAutoInsertable(result Result, signal MatchSignal, excluded []string) bool {
	if !signal.Matched {
		return false
	}
	if signal.Ambiguous {
		return false
	}
	if signal.TopScore < 1.0 {
		return false
	}
	if result.Subcategory != signal.TopSubcategory {
		return false
	}
	return !IsExcluded(result.Subcategory, excluded)
}
