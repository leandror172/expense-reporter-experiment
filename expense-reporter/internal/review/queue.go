package review

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"expense-reporter/internal/feedback"
	"expense-reporter/pkg/utils"
)

func ReadQueue(csvPath string) ([]QueueEntry, error) {
	file, err := os.Open(csvPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", csvPath, err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.Comma = ';'
	reader.LazyQuotes = true
	reader.FieldsPerRecord = -1 // disable auto-check; we validate length explicitly below

	entries := []QueueEntry{}
	lineNumber := 0
	headerSeen := false

	for {
		record, err := reader.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("failed to read line %d: %w", lineNumber+1, err)
		}
		lineNumber++

		if len(record) == 1 && strings.TrimSpace(record[0]) == "" {
			continue
		}

		if !headerSeen {
			headerSeen = true
			continue
		}

		if len(record) != 8 {
			return nil, fmt.Errorf("line %d: expected 8 fields, got %d", lineNumber, len(record))
		}

		item := strings.TrimSpace(record[0])
		date := strings.TrimSpace(record[1])
		valueStr := strings.TrimSpace(record[2])
		subcategory := strings.TrimSpace(record[3])
		category := strings.TrimSpace(record[4])
		confidenceStr := strings.TrimSpace(record[5])
		autoInsertedStr := strings.TrimSpace(record[6])
		expenseType := strings.TrimSpace(record[7])

		perInstallment, _, err := utils.ParseCurrencyWithInstallments(valueStr)
		if err != nil {
			return nil, fmt.Errorf("line %d: invalid value: %w", lineNumber, err)
		}

		confidence, err := strconv.ParseFloat(confidenceStr, 64)
		if err != nil {
			return nil, fmt.Errorf("line %d: invalid confidence: %w", lineNumber, err)
		}

		// "true"/"false" is the exact inverse of what the writers emit (fmt %v on a bool
		// in writeClassifiedCSV, a literal "false" in writeReviewCSV). This reader used to
		// accept ONLY "1"/"0" — a spelling no producer in the tree ever wrote — so `review`
		// hard-errored on every real batch-auto file (T-54). It went unnoticed because the
		// only inputs this parser ever saw were fixtures hand-authored to satisfy it.
		//
		// Deliberately strict rather than strconv.ParseBool: ParseBool would also swallow
		// "1"/"0"/"T"/"f", re-opening the same gap where the reader accepts spellings
		// nothing emits and no test covers. One producer, one spelling.
		var autoInserted bool
		switch autoInsertedStr {
		case "true":
			autoInserted = true
		case "false":
			autoInserted = false
		default:
			return nil, fmt.Errorf("line %d: invalid auto_inserted value %q (want \"true\" or \"false\")", lineNumber, autoInsertedStr)
		}

		// The id is hashed from the date column exactly as it appears in the CSV. Since
		// T-41 slice 3 that column is the CANONICAL DD/MM/YYYY (batch-auto's dateCell), so
		// this id is the SAME one both JSONL logs carry — which is what lets reviewed.json
		// join them. Before slice 3 the column held the raw input, so a bare-dated row got
		// a review-only id that apply would look up and never find.
		entries = append(entries, QueueEntry{
			ID:           feedback.GenerateID(item, date, perInstallment),
			Item:         item,
			Date:         date,
			RawValue:     valueStr,
			Value:        perInstallment,
			Confidence:   confidence,
			AutoInserted: autoInserted,
			Predicted: Predicted{
				Category:    category,
				Subcategory: subcategory,
				Type:        expenseType,
			},
		})
	}

	return entries, nil
}
