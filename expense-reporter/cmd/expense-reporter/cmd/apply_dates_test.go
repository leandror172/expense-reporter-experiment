package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"expense-reporter/internal/apply"
)

// TestCanonicalDate pins the exact string canonicalDate produces. The output is not
// cosmetic: it is hashed into GenerateID, which is the ONLY key joining
// classifications.jsonl to expenses_log.jsonl. A one-character difference here splits
// one expense across two ids in two files, with nothing reporting an error (T-35).
//
// The load-bearing case is "full form keeps its own year": the supplied year is a
// fallback for dates that omit one, never an override. If it overrode, applying a
// 2025 review under --year 2026 would silently relabel the expense's year.
func TestCanonicalDate(t *testing.T) {
	tests := []struct {
		name    string
		dateStr string
		year    int
		want    string
		wantOk  bool
	}{
		{
			name:    "short form takes the supplied year",
			dateStr: "15/04",
			year:    2026,
			want:    "15/04/2026",
			wantOk:  true,
		},
		{
			name:    "full form keeps its own year even when it differs from the supplied year",
			dateStr: "15/04/2025",
			year:    2026,
			want:    "15/04/2025",
			wantOk:  true,
		},
		{
			name:    "single-digit day and month get zero-padded",
			dateStr: "5/3",
			year:    2026,
			want:    "05/03/2026",
			wantOk:  true,
		},
		{
			name:    "empty string is not ok",
			dateStr: "",
			year:    2026,
			want:    "",
			wantOk:  false,
		},
		{
			name:    "garbage / non-numeric is not ok",
			dateStr: "abc/def",
			year:    2026,
			want:    "",
			wantOk:  false,
		},
		{
			name:    "a date with too many components is not ok",
			dateStr: "15/04/2026/extra",
			year:    2026,
			want:    "",
			wantOk:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotOk := canonicalDate(tt.dateStr, tt.year)
			assert.Equal(t, tt.wantOk, gotOk)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestEntriesWithCanonicalDates_NormalizesEveryForm pins that one pass over the slice
// leaves every entry in the canonical form, whichever form it arrived in — that is the
// property the whole boundary-normalization design rests on: downstream writers can then
// read entry.Date directly and cannot disagree.
func TestEntriesWithCanonicalDates_NormalizesEveryForm(t *testing.T) {
	entries := []apply.ReviewedEntry{
		{Item: "short form", Date: "15/04", Value: 35.50},
		{Item: "full form", Date: "10/05/2024", Value: 160.00},
		{Item: "unparseable", Date: "not-a-date", Value: 60.20},
	}

	got := entriesWithCanonicalDates(entries, 2026)

	require.Len(t, got, len(entries), "no entry may be dropped — appendNewRows reports date failures, not this helper")
	assert.Equal(t, "15/04/2026", got[0].Date, "short form should take the supplied year")
	assert.Equal(t, "10/05/2024", got[1].Date, "full form should keep its own year")
	assert.Equal(t, "not-a-date", got[2].Date, "an unparseable date must be preserved verbatim, not blanked")
}

// TestEntriesWithCanonicalDates_CarriesOtherFieldsThrough guards against the helper
// rebuilding entries from scratch and quietly losing a field.
func TestEntriesWithCanonicalDates_CarriesOtherFieldsThrough(t *testing.T) {
	original := newConfirmedRow("Uber Centro")

	got := entriesWithCanonicalDates([]apply.ReviewedEntry{original}, 2026)

	require.Len(t, got, 1)
	assert.Equal(t, original.Item, got[0].Item)
	assert.Equal(t, original.Value, got[0].Value)
	assert.Equal(t, original.Action, got[0].Action)
	assert.Equal(t, original.Reviewed, got[0].Reviewed)
}

// TestEntriesWithCanonicalDates_DoesNotMutateInput pins the "return, don't mutate"
// contract: the caller's slice is reused later (printSummary echoes entry dates), so
// normalizing in place would change what the summary reports.
func TestEntriesWithCanonicalDates_DoesNotMutateInput(t *testing.T) {
	entries := []apply.ReviewedEntry{{Item: "short form", Date: "15/04", Value: 35.50}}

	entriesWithCanonicalDates(entries, 2026)

	assert.Equal(t, "15/04", entries[0].Date, "the input slice must be left untouched")
}
