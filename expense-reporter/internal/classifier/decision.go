package classifier

// DefaultHighConfidenceThreshold is the default confidence threshold for auto-insertable decisions.
const DefaultHighConfidenceThreshold = 0.85

// IsAutoInsertable determines if a result meets the criteria to be considered auto-insertable.
//
// TODO(T-19): this confidence threshold is the only guard against auto-inserting a wrong
// row, but since T-13 the model is GBNF-constrained to the 112-path enum and cannot
// decline — novel / out-of-domain expenses get forced into a leaf, sometimes above the
// threshold. The threshold no longer reliably routes unknown items to manual review the
// way the pre-T-13 Diversos/0.30 fallback did. Do NOT lean harder on this path (e.g. WS-D
// / T-09 retiring the bare-name fallback) without resolving the escape-hatch gap. See
// tasks.md T-19; validate the forced-misclassification risk on real data first.
func IsAutoInsertable(result Result, threshold float64, excluded []string) bool {
	if result.Confidence < threshold {
		return false
	}
	for _, subcat := range excluded {
		if result.Subcategory == subcat {
			return false
		}
	}
	return true
}
