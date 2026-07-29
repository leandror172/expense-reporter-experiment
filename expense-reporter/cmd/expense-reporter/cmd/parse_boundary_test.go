package cmd

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"expense-reporter/internal/config"
	"expense-reporter/internal/parse"
)

// TestParseOptions_WiresEachYearSourceToItsOwnRung pins which ladder rung each
// input lands on. Both inputs are plain ints, so a transposition —
// Options{Year: cfg.DateYear, ConfigYear: yearFlag} — compiles, passes vet, and
// silently inverts the precedence the --year flag's help text promises. Nothing
// downstream would report it: the expense would just be dated by the config
// instead of by the flag the user typed.
func TestParseOptions_WiresEachYearSourceToItsOwnRung(t *testing.T) {
	opts := parseOptions(2024, &config.Config{DateYear: 2022})

	assert.Equal(t, 2024, opts.Year, "--year belongs on the flag rung")
	assert.Equal(t, 2022, opts.ConfigYear, "config date_year belongs on the config rung")
}

// TestParseOptions_LeavesTheClockUnset documents that the zero Now is deliberate,
// not an omission. parse reads a zero Now as time.Now(), which is what every
// command wants; only tests inject a clock, and they build Options directly.
func TestParseOptions_LeavesTheClockUnset(t *testing.T) {
	opts := parseOptions(2024, &config.Config{DateYear: 2022})

	assert.True(t, opts.Now.IsZero(), "commands must resolve against the real clock")
}

// TestDescribeParseFailure_KeepsTheBoundarysReason is a regression guard for a bug
// written and caught during T-41 slice 2. The first version of describeParseFailure
// answered every date rejection with a fixed "expected DD/MM or DD/MM/YYYY" hint, so
// `auto "Item" 35,50 15/04/2027` — a perfectly well-formed date refused because its
// YEAR is beyond the current one (T-47) — told the user their format was wrong.
//
// The failure mode is worth naming: the field was identified correctly and the reason
// was replaced by a plausible, false one. Inside parse the same hazard is guarded by
// wrapping with two %w verbs; this asserts the command layer honours it too, because
// a caller is free to discard what the boundary carefully preserved.
func TestDescribeParseFailure_KeepsTheBoundarysReason(t *testing.T) {
	_, err := parse.Fields("Posto Ipiranga", "15/04/2027", "35,50",
		parse.Options{Now: time.Date(2026, 7, 29, 0, 0, 0, 0, time.UTC)})
	require.Error(t, err)

	described := describeParseFailure(err)

	assert.ErrorContains(t, described, "future year", "the real objection must survive")
	assert.NotContains(t, described.Error(), "expected DD/MM",
		"a well-formed date must never be reported as a format problem")
}

// TestDescribeParseFailure_AddsInstallmentSyntaxToValueFailures pins the one thing
// the command layer genuinely contributes. parse rejects "abc" without ever
// mentioning that "35,50/3" is legal, so a user who mistyped an installment has no
// way to learn the syntax from the boundary's message alone.
func TestDescribeParseFailure_AddsInstallmentSyntaxToValueFailures(t *testing.T) {
	_, err := parse.Fields("Posto Ipiranga", "15/04/2024", "abc", parse.Options{})
	require.Error(t, err)

	described := describeParseFailure(err)

	assert.ErrorIs(t, described, parse.ErrInvalidValue, "the field must stay identifiable")
	assert.ErrorContains(t, described, "35,50/3", "the accepted installment form must be shown")
	assert.ErrorContains(t, described, "abc", "the rejected token reaches the user via the cause")
}

// TestParseOptions_ProducesTheDocumentedPrecedence checks the claim the --year
// flag help actually makes ("outranks config date_year; an explicit year in the
// date always wins") against the ladder itself, rather than trusting that the
// field names imply it. This composes parseOptions with the real parser, so it
// fails if either the wiring or the ladder's ordering changes.
//
// Years are all in the past on purpose: validateYearNotBeyondCurrent (T-47)
// refuses an entry dated beyond the current year, so a future year here would
// fail for that reason instead of the one under test.
func TestParseOptions_ProducesTheDocumentedPrecedence(t *testing.T) {
	tests := []struct {
		name       string
		dateStr    string
		yearFlag   int
		configYear int
		wantDate   string
		wantSource parse.YearSource
	}{
		{
			name:       "flag outranks config for a bare date",
			dateStr:    "15/04",
			yearFlag:   2024,
			configYear: 2022,
			wantDate:   "15/04/2024",
			wantSource: parse.YearFromFlag,
		},
		{
			name:       "config supplies the year when no flag is passed",
			dateStr:    "15/04",
			yearFlag:   0,
			configYear: 2022,
			wantDate:   "15/04/2022",
			wantSource: parse.YearFromConfig,
		},
		{
			name:       "an explicit year in the date beats both",
			dateStr:    "15/04/2023",
			yearFlag:   2024,
			configYear: 2022,
			wantDate:   "15/04/2023",
			wantSource: parse.YearFromDateString,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := parseOptions(tt.yearFlag, &config.Config{DateYear: tt.configYear})

			pe, err := parse.Fields("Posto Ipiranga", tt.dateStr, "35,50", opts)

			require.NoError(t, err)
			assert.Equal(t, tt.wantDate, pe.DateString())
			assert.Equal(t, tt.wantSource, pe.YearSource, "the reported rung must match the one that fired")
		})
	}
}
