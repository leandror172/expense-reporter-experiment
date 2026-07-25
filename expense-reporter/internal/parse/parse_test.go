package parse

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFields_ExplicitYearWinsOverAllOptions(t *testing.T) {
	opts := Options{
		Year:       2024,
		ConfigYear: 2022,
	}
	expense, err := Fields("Uber Centro", "15/04/2023", "35,50", opts)
	require.NoError(t, err)
	assert.Equal(t, "15/04/2023", expense.DateString())
}

func TestFields_YearOptionResolvesBareDate(t *testing.T) {
	opts := Options{
		Year:       2024,
		ConfigYear: 2022,
	}
	expense, err := Fields("Uber Centro", "15/04", "35,50", opts)
	require.NoError(t, err)
	assert.Equal(t, "15/04/2024", expense.DateString())
}

func TestFields_ConfigYearResolvesBareDate(t *testing.T) {
	opts := Options{
		ConfigYear: 2024,
	}
	expense, err := Fields("Uber Centro", "15/04", "35,50", opts)
	require.NoError(t, err)
	assert.Equal(t, "15/04/2024", expense.DateString())
}

func TestFields_MostRecentNonFutureFallback(t *testing.T) {
	now := time.Date(2026, 7, 22, 10, 0, 0, 0, time.UTC)
	opts := Options{Now: now}
	cases := []struct {
		name           string
		dateStr        string
		wantDateString string
	}{
		{
			name:           "past date resolves to current year",
			dateStr:        "15/04",
			wantDateString: "15/04/2026",
		},
		{
			name:           "today resolves to current year",
			dateStr:        "22/07",
			wantDateString: "22/07/2026",
		},
		{
			name:           "tomorrow is future, resolves to last year",
			dateStr:        "23/07",
			wantDateString: "23/07/2025",
		},
		{
			name:           "end of year is future, resolves to last year",
			dateStr:        "31/12",
			wantDateString: "31/12/2025",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			expense, err := Fields("Uber Centro", tc.dateStr, "35,50", opts)
			require.NoError(t, err)
			assert.Equal(t, tc.wantDateString, expense.DateString())
		})
	}
}

func TestFields_ValueAndInstallments(t *testing.T) {
	dateStr := "15/04/2026"
	cases := []struct {
		name             string
		valueStr         string
		wantValue        float64
		wantInstallments int
		wantRawValue     string
	}{
		{
			name:             "plain BR decimal",
			valueStr:         "35,50",
			wantValue:        35.50,
			wantInstallments: 1,
			wantRawValue:     "35,50",
		},
		{
			name:             "thousands separator",
			valueStr:         "1.234,56",
			wantValue:        1234.56,
			wantInstallments: 1,
			wantRawValue:     "1.234,56",
		},
		{
			name:             "installment total divided",
			valueStr:         "99,90/3",
			wantValue:        33.30,
			wantInstallments: 3,
			wantRawValue:     "99,90/3",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			expense, err := Fields("Uber Centro", dateStr, tc.valueStr, Options{})
			require.NoError(t, err)
			assert.InDelta(t, tc.wantValue, expense.Value, 0.001)
			assert.Equal(t, tc.wantInstallments, expense.Installments)
			assert.Equal(t, tc.wantRawValue, expense.RawValue)
		})
	}
}

func TestFields_TrimsInputs(t *testing.T) {
	expense, err := Fields("  Uber Centro  ", " 15/04/2026 ", " 35,50 ", Options{})
	require.NoError(t, err)
	assert.Equal(t, "Uber Centro", expense.Item)
	assert.Equal(t, "35,50", expense.RawValue)
	assert.Equal(t, "15/04/2026", expense.DateString())
}

func TestFields_Errors(t *testing.T) {
	cases := []struct {
		name     string
		item     string
		dateStr  string
		valueStr string
	}{
		{
			name:     "empty item",
			item:     "",
			dateStr:  "15/04/2026",
			valueStr: "35,50",
		},
		{
			name:     "empty date",
			item:     "Uber",
			dateStr:  "",
			valueStr: "35,50",
		},
		{
			name:     "malformed date",
			item:     "Uber",
			dateStr:  "not-a-date",
			valueStr: "35,50",
		},
		{
			name:     "malformed value",
			item:     "Uber",
			dateStr:  "15/04/2026",
			valueStr: "abc",
		},
		{
			name:     "zero installments",
			item:     "Uber",
			dateStr:  "15/04/2026",
			valueStr: "99,90/0",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Fields(tc.item, tc.dateStr, tc.valueStr, Options{})
			require.Error(t, err)
		})
	}
}

func TestExpenseString_ReturnsStructAndSubcategory(t *testing.T) {
	expense, subcategory, err := ExpenseString("Uber Centro;15/04/2026;35,50;Uber/Taxi", Options{})
	require.NoError(t, err)
	assert.Equal(t, "Uber/Taxi", subcategory)
	assert.Equal(t, "Uber Centro", expense.Item)
	assert.Equal(t, "15/04/2026", expense.DateString())
	assert.InDelta(t, 35.50, expense.Value, 0.001)
	assert.Equal(t, 1, expense.Installments)
}

func TestExpenseString_Errors(t *testing.T) {
	cases := []struct {
		name string
		s    string
	}{
		{
			name: "three fields only",
			s:    "Uber;15/04;35,50",
		},
		{
			name: "empty item",
			s:    ";15/04;35,50;Uber/Taxi",
		},
		{
			name: "empty subcategory",
			s:    "Uber;15/04;35,50;   ",
		},
		{
			name: "empty string",
			s:    "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := ExpenseString(tc.s, Options{})
			require.Error(t, err)
		})
	}
}

func TestDateString_ZeroPads(t *testing.T) {
	expense, err := Fields("x", "05/03/2026", "1,00", Options{})
	require.NoError(t, err)
	assert.Equal(t, "05/03/2026", expense.DateString())
}

// TestFields_RejectsUnrenderableYear guards the DateString() contract (T-46).
// DateString is documented as the canonical DD/MM/YYYY form and is the sole
// producer of the bytes hashed into the join id, so a year it cannot render in
// four digits must be rejected rather than silently emitted. Every rung that
// can supply a year is covered: the year written into the string itself, the
// --year flag, and config date_year. The zero value of Year/ConfigYear means
// "unset" and must keep falling through to the clock rung, never error.
func TestFields_RejectsUnrenderableYear(t *testing.T) {
	now := time.Date(2023, 5, 15, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name        string
		dateStr     string
		opts        Options
		wantErr     bool
		wantDateStr string
	}{
		{"valid full date", "15/04/2023", Options{Now: now}, false, "15/04/2023"},
		{"negative year in string", "15/04/-5", Options{Now: now}, true, ""},
		{"five-digit year in string", "15/04/12345", Options{Now: now}, true, ""},
		{"negative --year", "15/04", Options{Year: -5, Now: now}, true, ""},
		{"--year 10000", "15/04", Options{Year: 10000, Now: now}, true, ""},
		{"--year 1 accepted", "15/04", Options{Year: 1, Now: now}, false, "15/04/0001"},
		{"--year 9999 accepted", "15/04", Options{Year: 9999, Now: now}, false, "15/04/9999"},
		{"--year 0 falls through", "15/04", Options{Year: 0, Now: now}, false, "15/04/2023"},
		{"negative config date_year", "15/04", Options{ConfigYear: -5, Now: now}, true, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expense, err := Fields("Uber Centro", tt.dateStr, "35,50", tt.opts)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "9999",
					"error should name the acceptable range so a bad --year is actionable")
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantDateStr, expense.DateString())
		})
	}
}
