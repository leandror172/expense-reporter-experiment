package capture

// Outcomes in these tests are built through the real Classify function, never as
// struct literals: the parse boundary is what fills Line, Date and Err, so a literal
// would test the literal and could describe a combination production cannot produce.

import (
	"errors"
	"testing"
	"time"

	"expense-reporter/internal/parse"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// defaultNow is the clock Classify tests run against unless a row needs its own.
var defaultNow = day(2026, time.September, 3)

// day returns midnight UTC on the given date.
func day(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func TestResolveDateField(t *testing.T) {
	tests := []struct {
		name    string
		field   string
		sentAt  time.Time
		want    string
		wantErr bool
	}{
		{"same day", "03/05", day(2025, time.May, 3), "03/05/2025", false},
		{"reported four days later", "30/04", day(2025, time.May, 4), "30/04/2025", false},
		{"previous year across new year", "31/12", day(2026, time.January, 2), "31/12/2025", false},
		{"next year within the future allowance", "03/01", day(2026, time.December, 28), "03/01/2027", false},
		{"mistyped month is questioned, not resolved", "19/01", day(2025, time.October, 19), "", true},
		{"too far in the future", "05/07", day(2025, time.May, 7), "", true},
		{"not a calendar date", "31/02", day(2025, time.July, 6), "", true},
		{"explicit four-digit year verbatim", "08/07/2025", day(2025, time.July, 9), "08/07/2025", false},
		{"explicit year outside the window is refused, not filed under itself", "12/03/2024", day(2025, time.May, 1), "", true},
		{"absurd past year is refused rather than opening its own month file", "21/08/1200", day(2025, time.August, 21), "", true},
		{"explicit year at the future edge is kept", "08/07/2025", day(2025, time.July, 1), "08/07/2025", false},
		{"explicit year one day past the future edge is refused", "09/07/2025", day(2025, time.July, 1), "", true},
		{"two-digit year expands to this century", "25/07/25", day(2025, time.July, 25), "25/07/2025", false},
		{"single-digit day and month are padded", "3/5", day(2025, time.May, 3), "03/05/2025", false},
		{"surrounding spaces are trimmed", " 03/05 ", day(2025, time.May, 3), "03/05/2025", false},
		{"clock time is not a date", "16:20", day(2025, time.July, 5), "", true},
		{"empty", "", day(2025, time.July, 5), "", true},
		{"three-digit year", "25/07/025", day(2025, time.July, 25), "", true},
		{"four parts", "1/2/3/4", day(2025, time.July, 25), "", true},
		{"letters", "a/b", day(2025, time.July, 25), "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ResolveDateField(tt.field, tt.sentAt)
			if tt.wantErr {
				require.Error(t, err)
				assert.ErrorIs(t, err, parse.ErrInvalidDate)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestClassify(t *testing.T) {
	tests := []struct {
		name           string
		text           string
		attachment     Attachment
		sentAt         time.Time
		now            time.Time
		wantClass      Class
		wantBucket     Bucket
		wantLine       string
		wantDate       time.Time
		wantErrIs      error
		wantFieldCount int
	}{
		{"typed expense, same day", "Padaria; 03/05; 12,50", NoAttachment, day(2025, time.May, 3), defaultNow, Converted, BucketConverted, "Padaria;03/05/2025;12,50", day(2025, time.May, 3), nil, 0},
		{"no spaces around separators", "Padaria;03/05;12,50", NoAttachment, day(2025, time.May, 3), defaultNow, Converted, BucketConverted, "Padaria;03/05/2025;12,50", day(2025, time.May, 3), nil, 0},
		{"thousands separator kept raw in the line", "Mercado; 30/04; 1.234,56", NoAttachment, day(2025, time.May, 4), defaultNow, Converted, BucketConverted, "Mercado;30/04/2025;1.234,56", day(2025, time.April, 30), nil, 0},
		{"divisor installments pass through unexpanded", "Compras; 15/05; 900,00/3", NoAttachment, day(2025, time.May, 15), defaultNow, Converted, BucketConverted, "Compras;15/05/2025;900,00/3", day(2025, time.May, 15), nil, 0},
		{"multiplier installments pass through unexpanded", "Cartao Teste; 09/01; 405,25 x4", NoAttachment, day(2025, time.January, 9), defaultNow, Converted, BucketConverted, "Cartao Teste;09/01/2025;405,25 x4", day(2025, time.January, 9), nil, 0},
		{"explicit year", "Cinema; 08/07/2025; 45,00", NoAttachment, day(2025, time.July, 9), defaultNow, Converted, BucketConverted, "Cinema;08/07/2025;45,00", day(2025, time.July, 8), nil, 0},
		{"comma instead of the first semicolon is repaired (step 2)", "Almoço teste, 06/05; 50,00", NoAttachment, day(2025, time.May, 6), defaultNow, Converted, BucketRepaired, "Almoço teste;06/05/2025;50,00", day(2025, time.May, 6), nil, 0},
		{"stray semicolon inside the item is repaired (step 2)", "Cartao Teste; ADM; 09/01; 405,25 x4", NoAttachment, day(2025, time.July, 6), defaultNow, Converted, BucketRepaired, "Cartao Teste ADM;09/01/2025;405,25 x4", day(2025, time.January, 9), nil, 0},
		{"one field that talks about money is an attempted expense", "É 450,00 três meses, tá?", NoAttachment, day(2025, time.July, 9), defaultNow, Rejected, "rejected: 1 field", "", time.Time{}, nil, 1},
		{"conversation is ignored", "Não quer mais?", NoAttachment, day(2025, time.May, 6), defaultNow, Ignored, BucketIgnored, "", time.Time{}, nil, 0},
		{"clock time as value", "Uber; 05/07; 16:20", NoAttachment, day(2025, time.July, 5), defaultNow, Rejected, BucketBadValue, "", time.Time{}, parse.ErrInvalidValue, 0},
		{"impossible date", "Café; 31/02; 8,00", NoAttachment, day(2025, time.July, 6), defaultNow, Rejected, BucketBadDate, "", time.Time{}, parse.ErrInvalidDate, 0},
		{"REQUIRED (plan D1): resolves into next year, then refused as beyond the current year", "Bolo; 03/01; 20,00", NoAttachment, day(2026, time.December, 28), day(2026, time.December, 28), Rejected, BucketBadDate, "", time.Time{}, parse.ErrInvalidDate, 0},
		{"bare date too far back", "Bolo; 19/01; 20,00", NoAttachment, day(2025, time.October, 19), defaultNow, Rejected, BucketBadDate, "", time.Time{}, parse.ErrInvalidDate, 0},
		{"empty item", "; 03/05; 10,00", NoAttachment, day(2025, time.May, 3), defaultNow, Rejected, BucketRejectedOther, "", time.Time{}, nil, 0},
		{"photo with no text is a receipt", "", PhotoAttachment, day(2025, time.May, 4), defaultNow, Receipt, BucketReceipts, "", time.Time{}, nil, 0},
		{"pdf with no text is a receipt", "", FileAttachment, day(2025, time.May, 5), defaultNow, Receipt, BucketReceipts, "", time.Time{}, nil, 0},
		{"empty text and no attachment is ignored", "", NoAttachment, day(2025, time.May, 5), defaultNow, Ignored, BucketIgnored, "", time.Time{}, nil, 0},
		{"whitespace-only text is ignored", "   ", NoAttachment, day(2025, time.May, 5), defaultNow, Ignored, BucketIgnored, "", time.Time{}, nil, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := Message{
				ID:         1,
				SentAt:     tt.sentAt,
				Text:       tt.text,
				Attachment: tt.attachment,
			}
			if tt.now.IsZero() {
				tt.now = defaultNow
			}
			outcome := Classify(m, tt.now)

			assert.Equal(t, tt.wantClass, outcome.Class)
			if tt.wantBucket != "" {
				assert.Equal(t, tt.wantBucket, outcome.Bucket())
			}

			switch tt.wantClass {
			case Converted:
				assert.Equal(t, tt.wantLine, outcome.Line)
				assert.Equal(t, tt.wantDate, outcome.Date)
				assert.NoError(t, outcome.Err)
			case Rejected:
				assert.Error(t, outcome.Err)
				if tt.wantErrIs != nil {
					assert.ErrorIs(t, outcome.Err, tt.wantErrIs)
				}
				if tt.wantFieldCount != 0 {
					var fce FieldCountError
					require.True(t, errors.As(outcome.Err, &fce))
					assert.Equal(t, tt.wantFieldCount, fce.Got)
				}
			case Receipt, Ignored:
				assert.NoError(t, outcome.Err)
				assert.Empty(t, outcome.Line)
			}
		})
	}
}

func TestSummarize(t *testing.T) {
	t.Run("basic summary", func(t *testing.T) {
		outcomes := []Outcome{
			Classify(Message{ID: 1, Text: "Padaria; 03/05; 12,50", SentAt: day(2025, time.May, 3)}, defaultNow),
			Classify(Message{ID: 2, Text: "", Attachment: PhotoAttachment, SentAt: day(2025, time.May, 4)}, defaultNow),
			Classify(Message{ID: 3, Text: "Não quer mais?", SentAt: day(2025, time.May, 6)}, defaultNow),
			Classify(Message{ID: 4, Text: "Uber; 05/07; 16:20", SentAt: day(2025, time.July, 5)}, defaultNow),
		}

		sum := Summarize(outcomes)
		assert.Equal(t, 4, sum.Total)
		assert.Equal(t, map[Bucket][]int{
			BucketConverted: {1},
			BucketReceipts:  {2},
			BucketIgnored:   {3},
			BucketBadValue:  {4},
		}, sum.IDs)
	})

	t.Run("stream order preserved", func(t *testing.T) {
		outcomes := []Outcome{
			Classify(Message{ID: 7, Text: "Padaria; 03/05; 12,50", SentAt: day(2025, time.May, 3)}, defaultNow),
			Classify(Message{ID: 3, Text: "Mercado; 30/04; 1.234,56", SentAt: day(2025, time.May, 4)}, defaultNow),
		}

		sum := Summarize(outcomes)
		assert.Equal(t, []int{7, 3}, sum.IDs[BucketConverted])
	})

	// D10: three lines of one message, one of them unparseable. The id must appear once
	// per bucket and in BOTH, the counts must follow lines, and Total must still say one
	// message — the four numbers that stop "messages" and "lines" being conflated.
	t.Run("a split list is many lines under one message id", func(t *testing.T) {
		list := Message{
			ID:     9,
			Text:   "Mercado; 05/05; 100,00\nFarmacia; 06/05; 20,00\nPosto; 99/99; 50,00",
			SentAt: day(2025, time.May, 10),
		}

		sum := Summarize(ClassifyAll([]Message{list}, defaultNow))

		assert.Equal(t, 1, sum.Total, "one message was read")
		assert.Equal(t, 3, sum.Lines, "three expense lines were classified")
		assert.Equal(t, 1, sum.Splits, "one multi-line list was split")
		assert.Equal(t, map[Bucket]int{BucketConverted: 2, BucketBadDate: 1}, sum.Counts)
		assert.Equal(t, map[Bucket][]int{BucketConverted: {9}, BucketBadDate: {9}}, sum.IDs,
			"the parent id lands in both buckets, once each")
	})
}

// TestExpandLists pins plan D10's SHAPE test. The rows that matter most are the two that
// must NOT split: a chatty second line means one expense plus a note, and a four-field
// line means the message is not a clean list — in both cases tearing it apart would lose
// or invent an expense. The "one line fails to parse" row is the reason the test counts
// semicolons instead of parsing: the real message that forced this rule has two bad lines
// among ten good ones.
func TestExpandLists(t *testing.T) {
	tests := []struct {
		name  string
		text  string
		want  []string
		split bool
	}{
		{"single line is itself", "a; 03/05; 1,00", []string{"a; 03/05; 1,00"}, false},
		{"two 3-field lines split", "a; 03/05; 1,00\nb; 04/05; 2,00", []string{"a; 03/05; 1,00", "b; 04/05; 2,00"}, true},
		{"blank lines are ignored", "a; 03/05; 1,00\n\n  \nb; 04/05; 2,00", []string{"a; 03/05; 1,00", "b; 04/05; 2,00"}, true},
		{"lines are trimmed", "  a; 03/05; 1,00  \n  b; 04/05; 2,00", []string{"a; 03/05; 1,00", "b; 04/05; 2,00"}, true},
		{"a line that fails to parse still splits", "a; 03/05; 1,00\nb; 99/99; 2,00", []string{"a; 03/05; 1,00", "b; 99/99; 2,00"}, true},
		{"a chatty line blocks the split", "a; 03/05; 1,00\nvou pagar amanha", []string{"a; 03/05; 1,00\nvou pagar amanha"}, false},
		{"a four-field line blocks the split", "a; 03/05; 1,00\nb; 04/05; 2,00; x", []string{"a; 03/05; 1,00\nb; 04/05; 2,00; x"}, false},
		{"empty text is itself", "", []string{""}, false},
		{"whitespace only is itself", "  \n  ", []string{"  \n  "}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parent := Message{ID: 42, Text: tt.text, SentAt: day(2025, time.May, 10), Attachment: PhotoAttachment}

			got := expandLists(parent)

			texts := make([]string, len(got))
			for i, m := range got {
				texts[i] = m.Text
				assert.Equal(t, parent.ID, m.ID, "a line keeps the parent id")
				assert.Equal(t, parent.SentAt, m.SentAt, "a line keeps the parent timestamp")
				assert.Equal(t, parent.Attachment, m.Attachment, "a line keeps the parent attachment")
			}
			assert.Equal(t, tt.want, texts)
			assert.Equal(t, tt.split, len(got) > 1)
		})
	}
}
