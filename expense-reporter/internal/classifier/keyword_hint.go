package classifier

// KeywordHint returns the subcategory the keyword layer would suggest to a human
// reviewer, or "" when there is nothing worth showing.
//
// It is the ADVISORY sibling of IsAutoInsertable: the gate fires on agreement, this
// fires on DISAGREEMENT, and both demand the same unambiguous maximum-specificity
// signal. When the model and the keyword already agree there is no second opinion to
// offer, so an agreeing keyword returns "" rather than the (identical) subcategory.
//
// Measured on the first real monthly close (session 72): this predicate fires on 14 of
// 65 reviewed rows and the keyword is right 71.4% of the time, recovering 10 of the 25
// corrections that close needed. At that precision the hint MUST stay advisory —
// auto-applying it would inject errors on roughly 3 of every 14 rows. The caller
// displays it beside the model's answer; nothing may act on it.
//
// Pure function of the signal plus the model's answer, so it can be computed anywhere
// both are known.
func KeywordHint(signal MatchSignal, modelSubcategory string) string {
	if !signal.Matched {
		return ""
	}
	if signal.Ambiguous {
		return ""
	}
	if signal.TopScore < 1.0 {
		return ""
	}
	if modelSubcategory == signal.TopSubcategory {
		return ""
	}
	return signal.TopSubcategory
}
