package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"expense-reporter/internal/apply"
	"expense-reporter/internal/feedback"
)

// TestCanonicalDateCharacterization tests the behavior of canonicalDate function
// to ensure it correctly handles various date formats and year assignment rules.
func TestCanonicalDateCharacterization(t *testing.T) {
	tests := []struct {
		name    string
		dateStr string
		year    int
		want    string
		wantOk  bool
	}{
		{
			name:    "bare DD/MM takes supplied year",
			dateStr: "15/03",
			year:    2023,
			want:    "15/03/2023",
			wantOk:  true,
		},
		{
			name:    "DD/MM/YYYY keeps its own year",
			dateStr: "15/03/2024",
			year:    2023,
			want:    "15/03/2024",
			wantOk:  true,
		},
		{
			name:    "unparseable date returns empty string and false",
			dateStr: "invalid-date",
			year:    2023,
			want:    "",
			wantOk:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotOk := canonicalDate(tt.dateStr, tt.year)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.wantOk, gotOk)
		})
	}
}

// TestEntriesWithCanonicalDatesCharacterization tests the behavior of entriesWithCanonicalDates
// to ensure it correctly transforms dates while preserving untouched entries and not mutating input.
func TestEntriesWithCanonicalDatesCharacterization(t *testing.T) {
	entry1 := apply.ReviewedEntry{
		ID:         "id1",
		Item:       "item1",
		Date:       "15/03",
		Value:      100.0,
		Confidence: 0.9,
		Action:     apply.ActionConfirmed,
		Reviewed:   &apply.ReviewedLocation{Category: "cat1", Subcategory: "sub1"},
	}
	entry2 := apply.ReviewedEntry{
		ID:         "id2",
		Item:       "item2",
		Date:       "invalid-date",
		Value:      200.0,
		Confidence: 0.8,
		Action:     apply.ActionConfirmed,
		Reviewed:   &apply.ReviewedLocation{Category: "cat2", Subcategory: "sub2"},
	}

	entries := []apply.ReviewedEntry{entry1, entry2}
	copiedEntries := make([]apply.ReviewedEntry, len(entries))
	copy(copiedEntries, entries)

	result := entriesWithCanonicalDates(entries, 2023)

	// Check that the original slice is unchanged
	assert.Equal(t, copiedEntries, entries)

	// Check length is preserved
	assert.Len(t, result, 2)

	// Check first entry was transformed
	assert.Equal(t, "15/03/2023", result[0].Date)
	// Deliberately NOT asserting this entry's ID here. Its date changed, and whether the
	// ID must follow is the defect the next test pins — asserting "unchanged" would
	// characterize the bug as intended behavior and then break when it is fixed.

	// Check second entry is untouched
	assert.Equal(t, "invalid-date", result[1].Date)
	assert.Equal(t, entry2.ID, result[1].ID)
}

// TestEntriesWithCanonicalDatesRewritesIDWhenDateChanges tests that entriesWithCanonicalDates
// correctly rewrites the ID when a date changes to maintain consistency.
//
// This test is currently RED and deliberately so. The function does not rewrite IDs,
// which causes a mismatch between the entry's Date field and its ID, leading to
// potential duplicate application issues in downstream processing where apply looks up
// entries by their ID in classification logs.
//
// The stakes are high: if an entry has a stale ID after date canonicalization,
// it will be treated as new instead of recognized as already applied,
// resulting in duplicates being appended rather than errors being raised.
func TestEntriesWithCanonicalDatesRewritesIDWhenDateChanges(t *testing.T) {
	// Create an entry with a bare DD/MM date
	entry := apply.ReviewedEntry{
		ID:         feedback.GenerateID("item1", "15/03", 100.0),
		Item:       "item1",
		Date:       "15/03",
		Value:      100.0,
		Confidence: 0.9,
		Action:     apply.ActionConfirmed,
		Reviewed:   &apply.ReviewedLocation{Category: "cat1", Subcategory: "sub1"},
	}

	result := entriesWithCanonicalDates([]apply.ReviewedEntry{entry}, 2023)

	// The ID should be updated to reflect the canonical date
	expectedID := feedback.GenerateID("item1", "15/03/2023", 100.0)
	assert.Equal(t, expectedID, result[0].ID)
}
