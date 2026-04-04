package cmd

import (
	"encoding/json"
	"os"

	"expense-reporter/internal/classifier"
)

// ClassifyOutput represents the structure of classification output.
type ClassifyOutput struct {
	Item       string            `json:"item"`
	Value      float64           `json:"value"`
	Date       string            `json:"date"`
	Candidates []CandidateOutput `json:"candidates"`
}

// CandidateOutput represents a single classification candidate.
type CandidateOutput struct {
	Subcategory string  `json:"subcategory"`
	Category    string  `json:"category"`
	Confidence  float64 `json:"confidence"`
}

// AutoOutput represents the structure of automatic classification output.
// In JSON mode, auto NEVER inserts — it returns a recommendation only.
type AutoOutput struct {
	Item       string            `json:"item"`
	Value      float64           `json:"value"`
	Date       string            `json:"date"`
	Action     string            `json:"action"`
	Result     *CandidateOutput  `json:"result,omitempty"`
	Candidates []CandidateOutput `json:"candidates"`
	Message    string            `json:"message"`
}

// printJSON encodes the given value to JSON and writes it to os.Stdout with 2-space indent.
func printJSON(v any) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(v)
}

// toCandidates converts a slice of classifier.Result to []CandidateOutput.
func toCandidates(results []classifier.Result) []CandidateOutput {
	candidates := make([]CandidateOutput, len(results))
	for i, result := range results {
		candidates[i] = CandidateOutput{
			Subcategory: result.Subcategory,
			Category:    result.Category,
			Confidence:  result.Confidence,
		}
	}
	return candidates
}
