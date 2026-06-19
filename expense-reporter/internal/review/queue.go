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

		var autoInserted bool
		switch autoInsertedStr {
		case "1":
			autoInserted = true
		case "0":
			autoInserted = false
		default:
			return nil, fmt.Errorf("line %d: invalid auto_inserted value %q", lineNumber, autoInsertedStr)
		}

		// ID uses DD/MM date (no year) — stable within a review-to-apply cycle only
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
				Sheet:       expenseType,
			},
		})
	}

	return entries, nil
}
