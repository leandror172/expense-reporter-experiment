package cmd

import (
	"bytes"
	"strings"
	"testing"

	"expense-reporter/internal/capture"
	"github.com/stretchr/testify/assert"
)

// TestWriteBucketReport tests the writeBucketReport function with various summary inputs.
// The per-line ids are the point: the acceptance scenario pins the same shape end to end,
// and this table pins the two things it cannot reach — the zero-count lines and the
// sort order of rejected buckets.
func TestWriteBucketReport(t *testing.T) {
	tests := []struct {
		name      string
		summary   capture.Summary
		wantLines []string
	}{
		{
			name: "one message per bucket",
			summary: capture.Summary{
				Total: 14,
				IDs: map[capture.Bucket][]int{
					capture.BucketConverted: {1, 2, 10, 11},
					capture.BucketRepaired:  {5, 9, 15},
					capture.BucketReceipts:  {3, 4},
					"rejected: 1 field":     {12},
					capture.BucketAmbiguous: {14},
					capture.BucketBadDate:   {8},
					capture.BucketBadValue:  {7},
					capture.BucketIgnored:   {6},
				},
			},
			wantLines: []string{
				"telegram-import: dry run, nothing written",
				"  14 messages",
				"   4 converted",
				"   3 repaired (one edit made the line parse): 5 9 15",
				"   2 receipts (attachments beside typed expenses, skipped): 3 4",
				"   1 rejected: 1 field: 12",
				"   1 rejected: ambiguous: 14",
				"   1 rejected: bad date: 8",
				"   1 rejected: bad value: 7",
				"   1 ignored (conversation): 6",
			},
		},
		{
			name: "empty export",
			summary: capture.Summary{
				Total: 0,
				IDs:   map[capture.Bucket][]int{},
			},
			wantLines: []string{
				"telegram-import: dry run, nothing written",
				"   0 messages",
				"   0 converted",
				"   0 repaired (one edit made the line parse)",
				"   0 receipts (attachments beside typed expenses, skipped)",
				"   0 ignored (conversation)",
			},
		},
		{
			name: "rejected buckets are sorted by name whatever the map order",
			summary: capture.Summary{
				Total: 3,
				IDs: map[capture.Bucket][]int{
					capture.BucketBadValue: {30},
					"rejected: 2 fields":   {20},
					capture.BucketBadDate:  {10},
				},
			},
			wantLines: []string{
				"telegram-import: dry run, nothing written",
				"   3 messages",
				"   0 converted",
				"   0 repaired (one edit made the line parse)",
				"   0 receipts (attachments beside typed expenses, skipped)",
				"   1 rejected: 2 fields: 20",
				"   1 rejected: bad date: 10",
				"   1 rejected: bad value: 30",
				"   0 ignored (conversation)",
			},
		},
		{
			// D10: one message was a ten-line list whose lines disagreed, so id 42 is in
			// TWO buckets and the converted count (9 lines) exceeds the ids behind it.
			// This is the row that makes writeBucketLine's count/ids split observable —
			// with len(ids) it would print "2 converted" and "1 rejected: bad date".
			name: "a split list counts lines while the ids stay messages",
			summary: capture.Summary{
				Total:  2,
				Lines:  11,
				Splits: 1,
				Counts: map[capture.Bucket]int{
					capture.BucketConverted: 9,
					capture.BucketBadDate:   2,
				},
				IDs: map[capture.Bucket][]int{
					capture.BucketConverted: {41, 42},
					capture.BucketBadDate:   {42},
				},
			},
			wantLines: []string{
				"telegram-import: dry run, nothing written",
				"   2 messages",
				"  11 expense lines (1 multi-line list split)",
				"   9 converted",
				"   0 repaired (one edit made the line parse)",
				"   0 receipts (attachments beside typed expenses, skipped)",
				"   2 rejected: bad date: 42",
				"   0 ignored (conversation)",
			},
		},
		{
			name: "converted never lists ids",
			summary: capture.Summary{
				Total: 3,
				IDs: map[capture.Bucket][]int{
					capture.BucketConverted: {5, 6, 7},
				},
			},
			wantLines: []string{
				"telegram-import: dry run, nothing written",
				"   3 messages",
				"   3 converted",
				"   0 repaired (one edit made the line parse)",
				"   0 receipts (attachments beside typed expenses, skipped)",
				"   0 ignored (conversation)",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			writeBucketReport(&buf, "telegram-import: dry run, nothing written", oneLinePerMessage(tt.summary))
			got := buf.String()
			want := joinedLines(tt.wantLines) + "\n"
			assert.Equal(t, want, got)
		})
	}
}

func joinedLines(lines []string) string {
	return strings.Join(lines, "\n")
}

// oneLinePerMessage fills in the line counts a row left out, as one line per message —
// the world before plan D10, which is what every row above describes. A row that means to
// exercise D10 sets Counts itself and is passed through untouched.
func oneLinePerMessage(s capture.Summary) capture.Summary {
	if s.Counts != nil {
		return s
	}
	s.Counts = make(map[capture.Bucket]int, len(s.IDs))
	for bucket, ids := range s.IDs {
		s.Counts[bucket] = len(ids)
	}
	return s
}
