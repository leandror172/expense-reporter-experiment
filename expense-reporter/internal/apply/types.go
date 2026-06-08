package apply

// Action constants for reviewed entries
const (
	ActionConfirmed = "confirmed"
	ActionCorrected = "corrected"
	ActionSkipped   = "skipped"
	ActionPending   = "pending"
)

// ReviewedFile represents the top-level structure of the reviewed JSON file
type ReviewedFile struct {
	ReviewedAt string          `json:"reviewedAt"`
	Source     string          `json:"source"`
	Entries    []ReviewedEntry `json:"entries"`
}

// ReviewedEntry represents a single entry from the UI export
type ReviewedEntry struct {
	ID         string            `json:"id"`
	Item       string            `json:"item"`
	Date       string            `json:"date"`
	Value      float64           `json:"value"`
	Confidence float64           `json:"confidence"`
	Predicted  ReviewedLocation  `json:"predicted"`
	Action     string            `json:"action"`
	Reviewed   *ReviewedLocation `json:"reviewed"`
}

// ReviewedLocation represents a sheet/category/subcategory triple
type ReviewedLocation struct {
	Sheet     string `json:"sheet,omitempty"`
	Category  string `json:"category"`
	Subcategory string `json:"subcategory"`
}

// IsInsertable returns true when Action is confirmed or corrected AND Reviewed is non-nil
func (e *ReviewedEntry) IsInsertable() bool {
	if e.Reviewed == nil {
		return false
	}
	
	switch e.Action {
	case ActionConfirmed, ActionCorrected:
		return true
	default:
		return false
	}
}

// IsAlreadyHandled returns true when priorFound is true (idempotency guard helper)
func (e *ReviewedEntry) IsAlreadyHandled(priorFound bool) bool {
	return priorFound
}

