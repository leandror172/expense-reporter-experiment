package apply

import (
	"encoding/json"
)

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

// ReviewedLocation represents a type/category/subcategory triple
type ReviewedLocation struct {
	Type        string `json:"type,omitempty"`
	Category    string `json:"category"`
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

// UnmarshalJSON handles backward compatibility: legacy "sheet" key → Type field.
func (l *ReviewedLocation) UnmarshalJSON(b []byte) error {
	type alias ReviewedLocation // avoid recursion
	var withType struct {
		alias
		LegacySheet string `json:"sheet"`
	}
	if err := json.Unmarshal(b, &withType); err != nil {
		return err
	}
	*l = ReviewedLocation(withType.alias)
	if l.Type == "" {
		l.Type = withType.LegacySheet // fall back to pre-migration key
	}
	return nil
}
