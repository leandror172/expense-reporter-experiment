package feedback

import (
	"crypto/sha256"
	"encoding/json"
	"expense-reporter/internal/classifier"
	"fmt"
	"os"
	"strings"
	"time"
)

// Status represents the outcome of a classification event.
type Status string

const (
	StatusConfirmed Status = "confirmed"
	StatusCorrected Status = "corrected"
	StatusManual    Status = "manual"
)

// Now is exported so tests can inject a fixed timestamp.
var Now = time.Now

// Entry is one line in classifications.jsonl.
type Entry struct {
	ID                   string  `json:"id"`
	Item                 string  `json:"item"`
	Date                 string  `json:"date"`
	Value                float64 `json:"value"`
	PredictedSubcategory string  `json:"predicted_subcategory"`
	PredictedCategory    string  `json:"predicted_category"`
	Confidence           float64 `json:"confidence"`
	ActualSubcategory    string  `json:"actual_subcategory"`
	ActualCategory       string  `json:"actual_category"`
	Model                string  `json:"model"`
	Status               Status  `json:"status"`
	Timestamp            string  `json:"timestamp"`
}

// GenerateID returns the first 12 hex chars of sha256(normalized(item)|date|value).
func GenerateID(item, date string, value float64) string {
	// Normalize item: lowercase + trim whitespace
	normalized := strings.ToLower(strings.TrimSpace(item))
	// Build deterministic input string
	input := normalized + "|" + date + "|" + fmt.Sprintf("%.2f", value)
	// Hash and return prefix
	hash := sha256.Sum256([]byte(input))
	return fmt.Sprintf("%x", hash)[:12]
}

// Append marshals entry as a single JSON line and appends it to path (creates if absent).
func Append(path string, entry Entry) error {
	// Open or create the file for appending
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("opening feedback file: %w", err)
	}
	defer f.Close()

	// Marshal to a single JSON line
	line, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("marshaling feedback entry: %w", err)
	}

	// Write line + newline
	if _, err := f.Write(append(line, '\n')); err != nil {
		return fmt.Errorf("writing feedback entry: %w", err)
	}
	return nil
}

// NewConfirmedEntry builds a confirmed Entry where predicted == actual.
func NewConfirmedEntry(item, date string, value float64, predicted classifier.Result, model string) Entry {
	return Entry{
		ID:                   GenerateID(item, date, value),
		Item:                 item,
		Date:                 date,
		Value:                value,
		PredictedSubcategory: predicted.Subcategory,
		PredictedCategory:    predicted.Category,
		Confidence:           predicted.Confidence,
		ActualSubcategory:    predicted.Subcategory,
		ActualCategory:       predicted.Category,
		Model:                model,
		Status:               StatusConfirmed,
		Timestamp:            Now().UTC().Format(time.RFC3339),
	}
}

// NewManualEntry builds a manual Entry with empty predicted fields.
func NewManualEntry(item, date string, value float64, subcategory, category string) Entry {
	return Entry{
		ID:                   GenerateID(item, date, value),
		Item:                 item,
		Date:                 date,
		Value:                value,
		PredictedSubcategory: "",
		PredictedCategory:    "",
		Confidence:           0.0,
		ActualSubcategory:    subcategory,
		ActualCategory:       category,
		Model:                "",
		Status:               StatusManual,
		Timestamp:            Now().UTC().Format(time.RFC3339),
	}
}
