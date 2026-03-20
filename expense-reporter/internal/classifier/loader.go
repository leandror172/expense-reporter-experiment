package classifier

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// LoadTrainingExamples reads training examples from <dataDir>/training_data_complete.json.
// Returns nil, nil if the file does not exist (graceful degradation).
func LoadTrainingExamples(dataDir string) ([]Example, error) {
	path := dataDir + "/training_data_complete.json"
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("opening training data: %w", err)
	}
	defer file.Close()

	var rawData struct {
		Expenses []struct {
			Item        string  `json:"item"`
			Date        string  `json:"date"`
			Value       float64 `json:"value"`
			Subcategory string  `json:"subcategory"`
			Category    string  `json:"category"`
		} `json:"expenses"`
	}

	if err = json.NewDecoder(file).Decode(&rawData); err != nil {
		return nil, fmt.Errorf("parsing training data: %w", err)
	}

	examples := make([]Example, len(rawData.Expenses))
	for i, e := range rawData.Expenses {
		examples[i] = Example{
			Item:        e.Item,
			Date:        normalizeTrainingDate(e.Date),
			Value:       e.Value,
			Subcategory: e.Subcategory,
			Category:    e.Category,
			Source:      SourceTraining,
		}
	}
	return examples, nil
}

// LoadFeedbackExamples reads confirmed/corrected examples from a classifications.jsonl file.
// Returns nil, nil if the file does not exist (cold start).
// Lines with status "manual" are skipped.
func LoadFeedbackExamples(path string) ([]Example, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("opening feedback file: %w", err)
	}
	defer file.Close()

	var examples []Example
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}

		var entry struct {
			Item                 string  `json:"item"`
			Date                 string  `json:"date"`
			Value                float64 `json:"value"`
			PredictedSubcategory string  `json:"predicted_subcategory"`
			PredictedCategory    string  `json:"predicted_category"`
			ActualSubcategory    string  `json:"actual_subcategory"`
			ActualCategory       string  `json:"actual_category"`
			Status               string  `json:"status"`
		}

		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			return nil, fmt.Errorf("parsing feedback line: %w", err)
		}

		switch entry.Status {
		case "manual":
			continue
		case "confirmed":
			examples = append(examples, Example{
				Item:        entry.Item,
				Date:        feedbackDate(entry.Date),
				Value:       entry.Value,
				Subcategory: entry.PredictedSubcategory,
				Category:    entry.PredictedCategory,
				Source:      SourceConfirmed,
			})
		case "corrected":
			examples = append(examples, Example{
				Item:        entry.Item,
				Date:        feedbackDate(entry.Date),
				Value:       entry.Value,
				Subcategory: entry.ActualSubcategory,
				Category:    entry.ActualCategory,
				Source:      SourceCorrected,
			})
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading feedback file: %w", err)
	}
	return examples, nil
}

// LoadKeywordIndex reads the keyword index from <dataDir>/feature_dictionary_enhanced.json.
// Returns an error if the file does not exist (keywords are required).
func LoadKeywordIndex(dataDir string) (KeywordIndex, error) {
	path := dataDir + "/feature_dictionary_enhanced.json"
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading feature dictionary: %w", err)
	}

	var raw struct {
		LexicalFeatures struct {
			Keywords map[string]struct {
				DominantSubcategory string   `json:"dominant_subcategory"`
				Specificity         float64  `json:"specificity"`
				Subcategories       []string `json:"subcategories"`
			} `json:"keywords"`
		} `json:"lexical_features"`
	}

	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parsing feature dictionary: %w", err)
	}

	index := make(KeywordIndex, len(raw.LexicalFeatures.Keywords))
	for kw, e := range raw.LexicalFeatures.Keywords {
		index[kw] = KeywordEntry{
			DominantSubcategory: e.DominantSubcategory,
			Specificity:         e.Specificity,
			Subcategories:       e.Subcategories,
		}
	}
	return index, nil
}

// MergeExamplePools combines training and feedback examples.
// Feedback entries take precedence: if the same Item appears in both, the feedback
// version is kept (more recent / potentially corrected).
func MergeExamplePools(training, feedback []Example) []Example {
	if len(training) == 0 && len(feedback) == 0 {
		return nil
	}

	seen := make(map[string]bool, len(feedback))
	result := make([]Example, 0, len(feedback)+len(training))

	for _, ex := range feedback {
		key := strings.ToLower(strings.TrimSpace(ex.Item))
		if !seen[key] {
			seen[key] = true
			result = append(result, ex)
		}
	}
	for _, ex := range training {
		key := strings.ToLower(strings.TrimSpace(ex.Item))
		if !seen[key] {
			result = append(result, ex)
		}
	}
	return result
}

// normalizeTrainingDate converts "YYYY-MM-DD" → "DD/MM".
func normalizeTrainingDate(date string) string {
	parts := strings.SplitN(date, "-", 3)
	if len(parts) != 3 {
		return date
	}
	return parts[2] + "/" + parts[1]
}

// feedbackDate returns the DD/MM prefix of a date string (already DD/MM or DD/MM/YYYY).
func feedbackDate(date string) string {
	if len(date) >= 5 {
		return date[:5]
	}
	return date
}
