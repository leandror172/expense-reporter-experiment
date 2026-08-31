package review

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"expense-reporter/internal/feedback"
	"expense-reporter/internal/parse"
)

// ReadQueue reads a classified CSV into the review queue, returning the reviewable
// entries and the raw lines of any rows that could not be reviewed.
//
// The second return exists because batch-auto DELIBERATELY records a row it failed to
// parse, keeping the original text in the item column with empty date/value cells rather
// than rendering a zero time as 01/01/0001. This reader used to hard-error on exactly that
// shape, so a single unparseable row killed the entire review step — measured on real
// data, 4 rows in 69 (T-42 scout, S1). Skipping them is only half the fix: the caller MUST
// report them, or a loud failure silently becomes a lost expense.
//
// This tolerance is deliberately narrow. An empty date/value pair is a DOCUMENTED producer
// output; a malformed float, a bad confidence or a wrong field count is corruption, and
// those still hard-error. Widening this to "skip anything that fails to parse" would turn a
// corrupt file into a quietly short queue.
func ReadQueue(csvPath string) ([]QueueEntry, []string, error) {
	file, err := os.Open(csvPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open file %s: %w", csvPath, err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.Comma = ';'
	reader.LazyQuotes = true
	reader.FieldsPerRecord = -1 // disable auto-check; we validate length explicitly below

	entries := []QueueEntry{}
	unreviewable := []string{}
	lineNumber := 0
	headerSeen := false

	for {
		record, err := reader.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, nil, fmt.Errorf("failed to read line %d: %w", lineNumber+1, err)
		}
		lineNumber++

		if len(record) == 1 && strings.TrimSpace(record[0]) == "" {
			continue
		}

		if !headerSeen {
			headerSeen = true
			continue
		}

		if len(record) != 9 {
			return nil, nil, fmt.Errorf("line %d: expected 9 fields, got %d", lineNumber, len(record))
		}

		item := strings.TrimSpace(record[0])
		date := strings.TrimSpace(record[1])
		valueStr := strings.TrimSpace(record[2])
		subcategory := strings.TrimSpace(record[3])
		category := strings.TrimSpace(record[4])
		confidenceStr := strings.TrimSpace(record[5])
		autoInsertedStr := strings.TrimSpace(record[6])
		expenseType := strings.TrimSpace(record[7])
		keywordHint := strings.TrimSpace(record[8])

		// The unparsed-row shape batch-auto writes: raw text kept in the item column,
		// date and value blank. Recognised BEFORE any field parsing, because it is the
		// value parse that used to reject it.
		if date == "" || valueStr == "" {
			unreviewable = append(unreviewable, item)
			continue
		}

		// Through the parse boundary, not utils directly. This reader used to call
		// utils.ParseCurrencyWithInstallments itself, which meant it never applied the BR
		// thousands normalization internal/parse performs — so writeClassifiedCSV could
		// emit a "1.234,56" this side hard-errored on, killing the whole review step over
		// one row, with both halves' own tests green (s73). One parse site is what makes
		// that divergence unrepresentable instead of merely fixed; it is also how the
		// T-64 multiplier notation reaches this side without being taught twice.
		//
		// The error is not re-labelled here: parse.Value already wraps with
		// ErrInvalidValue, whose text is "invalid value", so the message a human reads is
		// unchanged and callers gain errors.Is on the field sentinel.
		perInstallment, installments, err := parse.Value(valueStr)
		if err != nil {
			return nil, nil, fmt.Errorf("line %d: %w", lineNumber, err)
		}

		confidence, err := strconv.ParseFloat(confidenceStr, 64)
		if err != nil {
			return nil, nil, fmt.Errorf("line %d: invalid confidence: %w", lineNumber, err)
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
			return nil, nil, fmt.Errorf("line %d: invalid auto_inserted value %q (want \"true\" or \"false\")", lineNumber, autoInsertedStr)
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
			Installments: installments,
			Confidence:   confidence,
			AutoInserted: autoInserted,
			KeywordHint:  keywordHint,
			Predicted: Predicted{
				Category:    category,
				Subcategory: subcategory,
				Type:        expenseType,
			},
		})
	}

	return entries, unreviewable, nil
}
