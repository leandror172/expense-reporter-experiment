package appender

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"expense-reporter/internal/feedback"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExpandAndAppend_SingleEntry(t *testing.T) {
	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "expenses_log.jsonl")

	err := ExpandAndAppend(logPath, "Grocery", time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC), 45.99, 1, "expense", "Food", "Groceries")
	require.NoError(t, err)

	file, err := os.Open(logPath)
	require.NoError(t, err)
	defer file.Close()

	scanner := bufio.NewScanner(file)
	assert.True(t, scanner.Scan())
	line := scanner.Text()
	assert.False(t, scanner.Scan()) // Only one line

	var entry map[string]interface{}
	err = json.Unmarshal([]byte(line), &entry)
	require.NoError(t, err)

	assert.Equal(t, "Grocery", entry["item"])
	assert.Equal(t, "15/03/2026", entry["date"])
	assert.Equal(t, 45.99, entry["value"])
	assert.Equal(t, "expense", entry["type"])
	assert.Equal(t, "Food", entry["category"])
	assert.Equal(t, "Groceries", entry["subcategory"])
}

func TestExpandAndAppend_InstallmentsProduceNEntries(t *testing.T) {
	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "expenses_log.jsonl")

	err := ExpandAndAppend(logPath, "Netflix", time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC), 30.0, 3, "expense", "Entertainment", "Streaming")
	require.NoError(t, err)

	file, err := os.Open(logPath)
	require.NoError(t, err)
	defer file.Close()

	scanner := bufio.NewScanner(file)
	assert.True(t, scanner.Scan())
	line1 := scanner.Text()
	assert.True(t, scanner.Scan())
	line2 := scanner.Text()
	assert.True(t, scanner.Scan())
	line3 := scanner.Text()
	assert.False(t, scanner.Scan()) // Exactly 3 lines

	var entry1, entry2, entry3 map[string]interface{}
	err = json.Unmarshal([]byte(line1), &entry1)
	require.NoError(t, err)
	err = json.Unmarshal([]byte(line2), &entry2)
	require.NoError(t, err)
	err = json.Unmarshal([]byte(line3), &entry3)
	require.NoError(t, err)

	assert.Equal(t, "Netflix (1/3)", entry1["item"])
	assert.Equal(t, "15/03/2026", entry1["date"])
	assert.Equal(t, 30.0, entry1["value"])

	assert.Equal(t, "Netflix (2/3)", entry2["item"])
	assert.Equal(t, "15/04/2026", entry2["date"])
	assert.Equal(t, 30.0, entry2["value"])

	assert.Equal(t, "Netflix (3/3)", entry3["item"])
	assert.Equal(t, "15/05/2026", entry3["date"])
	assert.Equal(t, 30.0, entry3["value"])
}

func TestExpandAndAppend_CrossYearInstallmentDate(t *testing.T) {
	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "expenses_log.jsonl")

	err := ExpandAndAppend(logPath, "Subscription", time.Date(2026, 11, 15, 0, 0, 0, 0, time.UTC), 30.0, 3, "expense", "Entertainment", "Streaming")
	require.NoError(t, err)

	file, err := os.Open(logPath)
	require.NoError(t, err)
	defer file.Close()

	scanner := bufio.NewScanner(file)
	assert.True(t, scanner.Scan())
	line1 := scanner.Text()
	assert.True(t, scanner.Scan())
	line2 := scanner.Text()
	assert.True(t, scanner.Scan())
	line3 := scanner.Text()
	assert.False(t, scanner.Scan()) // Exactly 3 lines

	var entry1, entry2, entry3 map[string]interface{}
	err = json.Unmarshal([]byte(line1), &entry1)
	require.NoError(t, err)
	err = json.Unmarshal([]byte(line2), &entry2)
	require.NoError(t, err)
	err = json.Unmarshal([]byte(line3), &entry3)
	require.NoError(t, err)

	assert.Equal(t, "Subscription (1/3)", entry1["item"])
	assert.Equal(t, "15/11/2026", entry1["date"])

	assert.Equal(t, "Subscription (2/3)", entry2["item"])
	assert.Equal(t, "15/12/2026", entry2["date"])

	assert.Equal(t, "Subscription (3/3)", entry3["item"])
	assert.Equal(t, "15/01/2027", entry3["date"])
}

func TestAddMonths_YearWrap(t *testing.T) {
	result := addMonths(time.Date(2026, 12, 15, 0, 0, 0, 0, time.UTC), 2)
	expected := time.Date(2027, 2, 15, 0, 0, 0, 0, time.UTC)
	assert.Equal(t, expected, result)
}

func TestAddMonths_DayOverflow(t *testing.T) {
	result := addMonths(time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC), 1)
	expected := time.Date(2026, 2, 28, 0, 0, 0, 0, time.UTC)
	assert.Equal(t, expected, result)
}

func TestFormatDate(t *testing.T) {
	date := time.Date(2026, 3, 5, 10, 30, 45, 123456789, time.UTC)
	result := formatDate(date)
	assert.Equal(t, "05/03/2026", result)
}

// TestPredictEntryIDs_MatchesExpandAndAppend is the drift guard for --resume: the ids
// PredictEntryIDs computes must equal, in order, the ids ExpandAndAppend actually writes
// to the log for the same inputs. If the two ever diverge (e.g. a missing (i/N) suffix or
// month increment in one path), resume would skip the wrong rows — this property fails loudly.
func TestPredictEntryIDs_MatchesExpandAndAppend(t *testing.T) {
	type testCase struct {
		name          string
		item          string
		date          time.Time
		value         float64
		count         int
		expectedCount int
	}

	tests := []testCase{
		{
			name:          "single installment",
			item:          "Grocery",
			date:          time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC),
			value:         45.99,
			count:         1,
			expectedCount: 1,
		},
		{
			name:          "three installments",
			item:          "Netflix",
			date:          time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC),
			value:         30.0,
			count:         3,
			expectedCount: 3,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tempDir := t.TempDir()
			logPath := filepath.Join(tempDir, "expenses_log.jsonl")

			err := ExpandAndAppend(logPath, tc.item, tc.date, tc.value, tc.count, "expense", "Cat", "Sub")
			require.NoError(t, err)

			// Read IDs from the log file
			file, err := os.Open(logPath)
			require.NoError(t, err)
			defer file.Close()

			scanner := bufio.NewScanner(file)
			var loggedIDs []string
			for scanner.Scan() {
				line := scanner.Text()
				if line == "" {
					continue
				}
				var entry feedback.ExpenseEntry
				err = json.Unmarshal([]byte(line), &entry)
				require.NoError(t, err)
				loggedIDs = append(loggedIDs, entry.ID)
			}
			require.NoError(t, scanner.Err())

			// Get predicted IDs
			predictedIDs := PredictEntryIDs(tc.item, tc.date, tc.value, tc.count)

			// Assert they match in order and length
			assert.Equal(t, loggedIDs, predictedIDs)
			assert.Len(t, predictedIDs, tc.expectedCount)
		})
	}
}
