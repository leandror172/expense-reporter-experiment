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

// TestValidateRenderableYear_Boundaries covers the renderable range directly.
// It cannot be driven through Fields for the upper bound: year 9999 is beyond
// any realistic current year, so validateYearNotBeyondCurrent rejects it first
// and the renderability boundary would never be exercised. Testing the contract
// where it lives keeps that coverage honest instead of silently masked (T-46).
func TestValidateRenderableYear_Boundaries(t *testing.T) {
	assert.NoError(t, validateRenderableYear(1), "year 1 renders as 0001")
	assert.NoError(t, validateRenderableYear(9999), "year 9999 is the last 4-digit year")
	assert.Error(t, validateRenderableYear(0), "year 0 is not reachable but is out of contract")
	assert.Error(t, validateRenderableYear(-5))
	assert.Error(t, validateRenderableYear(10000))
}

// TestFields_RejectsYearBeyondCurrent covers the provisional future-entry block
// (T-47). The rule is year-scale on purpose: a date later in the CURRENT year is
// still accepted, because the failure being blocked is a mistyped year, which
// makes generate-workbook silently drop the row. Installment expansion is
// deliberately unaffected — it never re-enters this boundary.
func TestFields_RejectsYearBeyondCurrent(t *testing.T) {
	now := time.Date(2026, 7, 25, 12, 0, 0, 0, time.UTC)

	cases := []struct {
		name    string
		dateStr string
		opts    Options
		wantErr bool
	}{
		{"past date this year", "15/04/2026", Options{Now: now}, false},
		{"later this year is allowed", "30/12/2026", Options{Now: now}, false},
		{"cross-year installment start date", "15/11/2026", Options{Now: now}, false},
		{"prior year", "15/04/2025", Options{Now: now}, false},
		{"next year in the string", "15/04/2027", Options{Now: now}, true},
		{"far future in the string", "15/04/2050", Options{Now: now}, true},
		{"typo year in the string", "15/04/2205", Options{Now: now}, true},
		{"next year via --year", "15/04", Options{Year: 2027, Now: now}, true},
		{"next year via config date_year", "15/04", Options{ConfigYear: 2027, Now: now}, true},
		{"bare date never resolves forward", "30/12", Options{Now: now}, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Fields("Uber Centro", tc.dateStr, "35,50", tc.opts)
			if tc.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "future year",
					"error should say why it was refused")
				return
			}
			require.NoError(t, err)
		})
	}
}

// TestFields_YearSource pins which rung of the year ladder supplied the year.
// The source is asserted alongside the resolved date so a case that reports the
// right rung for the wrong reason still fails. The clock is injected so these
// cases do not change meaning with the calendar year the suite runs in.
func TestFields_YearSource(t *testing.T) {
	now := time.Date(2026, 7, 15, 10, 0, 0, 0, time.UTC)

	cases := []struct {
		name        string
		dateStr     string
		opts        Options
		wantSource  YearSource
		wantDateStr string
	}{
		{
			name:        "year written into the date string",
			dateStr:     "15/04/2023",
			opts:        Options{Now: now},
			wantSource:  YearFromDateString,
			wantDateStr: "15/04/2023",
		},
		{
			name:        "year flag",
			dateStr:     "15/04",
			opts:        Options{Year: 2024, Now: now},
			wantSource:  YearFromFlag,
			wantDateStr: "15/04/2024",
		},
		{
			name:        "configured default year",
			dateStr:     "15/04",
			opts:        Options{ConfigYear: 2024, Now: now},
			wantSource:  YearFromConfig,
			wantDateStr: "15/04/2024",
		},
		{
			name:        "clock fallback, date already past this year",
			dateStr:     "15/04",
			opts:        Options{Now: now},
			wantSource:  YearFromClock,
			wantDateStr: "15/04/2026",
		},
		{
			name:        "clock fallback, future date resolves to last year",
			dateStr:     "23/07",
			opts:        Options{Now: now},
			wantSource:  YearFromClock,
			wantDateStr: "23/07/2025",
		},
		{
			name:        "date string outranks both flag and config",
			dateStr:     "15/04/2023",
			opts:        Options{Year: 2024, ConfigYear: 2022, Now: now},
			wantSource:  YearFromDateString,
			wantDateStr: "15/04/2023",
		},
		{
			name:        "flag outranks config",
			dateStr:     "15/04",
			opts:        Options{Year: 2024, ConfigYear: 2022, Now: now},
			wantSource:  YearFromFlag,
			wantDateStr: "15/04/2024",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			expense, err := Fields("Uber Centro", tc.dateStr, "35,50", tc.opts)
			require.NoError(t, err)
			assert.Equal(t, tc.wantSource, expense.YearSource)
			assert.Equal(t, tc.wantDateStr, expense.DateString())
		})
	}
}

// TestYearSource_ZeroValueIsUnknown pins the enum's zero value, which no
// successful parse can produce and so cannot be covered by the table above.
// It matters because every error path returns ParsedExpense{}: if the zero value
// named a real rung, a failed parse would report a plausible source, and the
// stale-date_year warning that reads this field would decide on a value no parse
// ever produced. Same reasoning as the zero Date formatting as 01/01/0001 —
// wrong loudly rather than wrong quietly.
func TestYearSource_ZeroValueIsUnknown(t *testing.T) {
	assert.Equal(t, YearSourceUnknown, ParsedExpense{}.YearSource)

	expense, err := Fields("Uber Centro", "not-a-date", "35,50", Options{})
	require.Error(t, err)
	assert.Equal(t, YearSourceUnknown, expense.YearSource, "a failed parse must not report a rung")
}

// TestFields_ErrorsNameTheFieldThatFailed pins that every rejection is
// attributable to a field via errors.Is, so no caller has to match on message
// text to know what went wrong. Message matching is what this replaces: the CLI
// wants to name the offending argument, and a future chat layer has to explain
// the failure in Portuguese, which it cannot do by reading English strings.
//
// The load-bearing rows are the three year validations. They fail for quite
// different reasons — unparseable, unrenderable, beyond the current year — yet
// all three are the DATE field's problem, and a caller that only handled
// "unparseable" would print the wrong hint for the other two.
func TestFields_ErrorsNameTheFieldThatFailed(t *testing.T) {
	// A fixed clock keeps the beyond-current-year row from depending on the run
	// date, which would otherwise turn green into red in some future year.
	july2026 := Options{Now: time.Date(2026, 7, 29, 0, 0, 0, 0, time.UTC)}

	tests := []struct {
		name     string
		item     string
		dateStr  string
		valueStr string
		opts     Options
		wantIs   error
		wantNot  error
	}{
		{
			name:     "empty date",
			item:     "Posto Ipiranga",
			dateStr:  "",
			valueStr: "35,50",
			opts:     july2026,
			wantIs:   ErrInvalidDate,
			wantNot:  ErrInvalidValue,
		},
		{
			name:     "unparseable date",
			item:     "Posto Ipiranga",
			dateStr:  "not-a-date",
			valueStr: "35,50",
			opts:     july2026,
			wantIs:   ErrInvalidDate,
			wantNot:  ErrInvalidValue,
		},
		{
			name:     "year outside the renderable range",
			item:     "Posto Ipiranga",
			dateStr:  "15/04/12345",
			valueStr: "35,50",
			opts:     july2026,
			wantIs:   ErrInvalidDate,
			wantNot:  ErrInvalidValue,
		},
		{
			name:     "year beyond the current one",
			item:     "Posto Ipiranga",
			dateStr:  "15/04/2027",
			valueStr: "35,50",
			opts:     july2026,
			wantIs:   ErrInvalidDate,
			wantNot:  ErrInvalidValue,
		},
		{
			name:     "unparseable value",
			item:     "Posto Ipiranga",
			dateStr:  "15/04/2024",
			valueStr: "not-a-number",
			opts:     july2026,
			wantIs:   ErrInvalidValue,
			wantNot:  ErrInvalidDate,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Fields(tt.item, tt.dateStr, tt.valueStr, tt.opts)

			require.Error(t, err)
			assert.ErrorIs(t, err, tt.wantIs)
			assert.NotErrorIs(t, err, tt.wantNot, "a failure must name one field, not both")
		})
	}
}

// TestFields_EmptyItemCarriesNoFieldSentinel pins the deliberate absence. Adding
// an item sentinel later is easy; having one that no caller reads is API that
// misleads the next reader into thinking someone branches on it.
func TestFields_EmptyItemCarriesNoFieldSentinel(t *testing.T) {
	_, err := Fields("", "15/04/2024", "35,50", Options{})

	require.Error(t, err)
	assert.NotErrorIs(t, err, ErrInvalidDate)
	assert.NotErrorIs(t, err, ErrInvalidValue)
}

// TestFields_WrappedErrorKeepsTheSpecificCause guards the half of the wrapping
// that errors.Is cannot see. Wrapping with a single %w would satisfy every
// assertion above while replacing "year 12345 is not renderable" with a bare
// "invalid date" — the caller would know WHICH field failed and no longer know
// WHY. Both halves have to survive, so both are asserted.
func TestFields_WrappedErrorKeepsTheSpecificCause(t *testing.T) {
	_, err := Fields("Posto Ipiranga", "15/04/12345", "35,50", Options{})

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidDate, "the field must still be identifiable")
	assert.ErrorContains(t, err, "12345", "the underlying reason must survive the wrap")
}

// TestValue_AcceptsEveryFormFieldsAccepts pins the date-free door onto the boundary.
//
// Value exists for the same reason Date does: a caller may arrive with only ONE field
// still in string form. review.ReadQueue is that caller for the value — its date column
// has been canonical since T-41 slice 3, and only the value token is still text. Without
// this entry point it called utils directly, which is how BR thousands normalization came
// to exist on one side of the classified.csv seam and not the other (s73).
//
// The thousands case is the one that was broken. The multiplier case is T-64.
func TestValue_AcceptsEveryFormFieldsAccepts(t *testing.T) {
	cases := []struct {
		name             string
		valueStr         string
		wantValue        float64
		wantInstallments int
	}{
		{name: "plain BR decimal", valueStr: "35,50", wantValue: 35.50, wantInstallments: 1},
		{name: "thousands separator", valueStr: "1.234,56", wantValue: 1234.56, wantInstallments: 1},
		{name: "divisor states the total", valueStr: "405,25/4", wantValue: 101.3125, wantInstallments: 4},
		{name: "multiplier states the per-installment", valueStr: "405,25 x4", wantValue: 405.25, wantInstallments: 4},
		{name: "thousands and multiplier together", valueStr: "1.234,56 x3", wantValue: 1234.56, wantInstallments: 3},
		{name: "surrounding whitespace", valueStr: "  35,50  ", wantValue: 35.50, wantInstallments: 1},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			value, installments, err := Value(tc.valueStr)
			require.NoError(t, err)
			assert.InDelta(t, tc.wantValue, value, 0.0001)
			assert.Equal(t, tc.wantInstallments, installments)
		})
	}
}

// TestValue_AgreesWithFields is the anti-drift guard for the delegation. Two entry points
// that parse the same token must not be free to disagree — that freedom is exactly what
// produced the seam bug this refactor closes.
//
// It compares Value against Fields rather than against literals on purpose: a literal would
// pin what the author believed, while this pins that the two doors stay the same door. The
// dates are irrelevant here and fixed to a full date so no year rung participates.
func TestValue_AgreesWithFields(t *testing.T) {
	for _, valueStr := range []string{"35,50", "1.234,56", "405,25/4", "405,25 x4", "646,25 4x"} {
		t.Run(valueStr, func(t *testing.T) {
			expense, err := Fields("Posto Ipiranga", "15/04/2026", valueStr, Options{})
			require.NoError(t, err)

			value, installments, err := Value(valueStr)
			require.NoError(t, err)

			assert.Equal(t, expense.Value, value, "both doors must read the same value")
			assert.Equal(t, expense.Installments, installments, "both doors must read the same count")
		})
	}
}

// TestValue_NamesTheFieldAndKeepsTheCause holds Value to the same two-part wrapping
// contract Fields already meets: errors.Is must identify WHICH field failed, and the
// specific reason must survive the wrap. A caller that can only read English message text
// is not a caller this boundary supports — the eventual consumers are an error log keyed
// by phase and a PT-BR chat layer.
func TestValue_NamesTheFieldAndKeepsTheCause(t *testing.T) {
	_, _, err := Value("405,25 x99")

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidValue, "the field must be identifiable")
	assert.NotErrorIs(t, err, ErrInvalidDate, "and must not be confusable with the other field")
	assert.ErrorContains(t, err, "99", "the underlying reason must survive the wrap")
}
