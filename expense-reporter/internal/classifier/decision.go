package classifier

// DefaultHighConfidenceThreshold is the default confidence threshold for auto-insertable decisions.
const DefaultHighConfidenceThreshold = 0.85

// IsAutoInsertable determines if a result meets the criteria to be considered auto-insertable.
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
