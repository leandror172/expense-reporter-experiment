package capture

// Every outcome here is produced by Classify, so the D5 ordering (as typed first,
// repairs only afterwards) is exercised through the same door production uses. Two
// rows are marked REQUIRED: they are the plan's two mutation targets — run repairs
// before or instead of the as-typed parse and the D5 row goes red; take the first valid
// candidate instead of demanding exactly one and the D4 row goes red.

import (
	"errors"
	"testing"
	"time"

	"expense-reporter/internal/parse"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepairCandidates(t *testing.T) {
	tests := []struct {
		name string
		text string
		want []string
	}{
		{
			name: "comma and the value's own comma",
			text: "Almoço teste, 19/05; 50,00",
			want: []string{"Almoço teste; 19/05; 50,00", "Almoço teste, 19/05; 50;00", "Almoço teste, 19;05; 50,00"},
		},
		{
			name: "no comma, one slash",
			text: "Uber; 05/07; 16:20",
			want: []string{"Uber; 05;07; 16:20"},
		},
		{
			name: "four fields add the merge last",
			text: "Cartao Teste; ADM; 09/01; 405,25 x4",
			want: []string{"Cartao Teste; ADM; 09/01; 405;25 x4", "Cartao Teste; ADM; 09;01; 405,25 x4", "Cartao Teste ADM; 09/01; 405,25 x4"},
		},
		{
			name: "merge joins the two fields with exactly one space",
			text: "A ;  B; 01/05; 1,00",
			want: []string{"A ;  B; 01/05; 1;00", "A ;  B; 01;05; 1,00", "A B; 01/05; 1,00"},
		},
		{name: "five fields get no merge", text: "a; b; c; d; e", want: nil},
		{name: "empty text", text: "", want: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := repairCandidates(tt.text)
			if len(tt.want) == 0 {
				assert.Empty(t, got)
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}

	t.Run("the text itself is never a candidate", func(t *testing.T) {
		text := "Padaria; 03/05; 12,50"
		assert.NotContains(t, repairCandidates(text), text)
	})
}

// repairCase is one row of TestClassify_Repairs.
type repairCase struct {
	name         string
	text         string
	attachment   Attachment
	sentAt       time.Time
	now          time.Time
	wantClass    Class
	wantBucket   Bucket
	wantLine     string
	wantRepair   string
	wantErrIs    error
	wantReadings int
}

func TestClassify_Repairs(t *testing.T) {
	tests := []repairCase{
		{
			name:       "comma where the first semicolon should be",
			text:       "Almoço teste, 19/05; 50,00",
			sentAt:     day(2025, time.May, 19),
			wantClass:  Converted,
			wantBucket: BucketRepaired,
			wantLine:   "Almoço teste;19/05/2025;50,00",
			wantRepair: "Almoço teste; 19/05; 50,00",
		},
		{
			name:       "comma where the second semicolon should be",
			text:       "Gás 2 bujões; 15/05, 216,00",
			sentAt:     day(2025, time.May, 15),
			wantClass:  Converted,
			wantBucket: BucketRepaired,
			wantLine:   "Gás 2 bujões;15/05/2025;216,00",
			wantRepair: "Gás 2 bujões; 15/05; 216,00",
		},
		{
			name:       "slash typed instead of the second semicolon",
			text:       "Tela celular; 25/07/ 90,00",
			sentAt:     day(2025, time.July, 25),
			wantClass:  Converted,
			wantBucket: BucketRepaired,
			wantLine:   "Tela celular;25/07/2025;90,00",
			wantRepair: "Tela celular; 25/07; 90,00",
		},
		{
			// The comma edit reads "21/08/ 1200" as the year 1200, which the boundary
			// ACCEPTS (1..9999, past side unguarded). Only the window rule on candidates
			// removes that reading — measured on the 2025 export, s75.
			name:       "slash typo whose value spells a past year: the window rule keeps exactly one reading",
			text:       "Dentista; 21/08/ 1200,00",
			sentAt:     day(2025, time.August, 21),
			wantClass:  Converted,
			wantBucket: BucketRepaired,
			wantLine:   "Dentista;21/08/2025;1200,00",
			wantRepair: "Dentista; 21/08; 1200,00",
		},
		{
			name:       "stray semicolon inside the item",
			text:       "Cartao Teste; ADM; 09/01; 405,25 x4",
			sentAt:     day(2025, time.January, 9),
			wantClass:  Converted,
			wantBucket: BucketRepaired,
			wantLine:   "Cartao Teste ADM;09/01/2025;405,25 x4",
			wantRepair: "Cartao Teste ADM; 09/01; 405,25 x4",
		},
		{
			name:       "REQUIRED (plan D5): a line that parses as typed is never repaired",
			text:       "Compras; 15/05; 900,00/3",
			sentAt:     day(2025, time.May, 15),
			wantClass:  Converted,
			wantBucket: BucketConverted,
			wantLine:   "Compras;15/05/2025;900,00/3",
			wantRepair: "",
		},
		{
			// Slash edit: 15/05/2025 + 2025,50. Comma edit: the date "15/05/ 2025" + the
			// value 50 — inside the window, because the value happens to spell the year.
			name:         "REQUIRED (plan D4): two readings is an ambiguity, never a guess",
			text:         "Bar; 15/05/ 2025,50",
			sentAt:       day(2025, time.May, 15),
			wantClass:    Rejected,
			wantBucket:   BucketAmbiguous,
			wantReadings: 2,
		},
		{
			name:       "no edit parses: the as-typed value reason is kept",
			text:       "Uber; 05/07; 16:20",
			sentAt:     day(2025, time.July, 5),
			wantClass:  Rejected,
			wantBucket: BucketBadValue,
			wantErrIs:  parse.ErrInvalidValue,
		},
		{
			name:       "no edit parses: the as-typed field count is kept",
			text:       "É 450,00 três meses, tá?",
			sentAt:     day(2025, time.July, 9),
			wantClass:  Rejected,
			wantBucket: Bucket("rejected: 1 field"),
		},
		{
			name:       "composed defects are not repaired (plan D6)",
			text:       "Teste; Terpenos coolterps; 346,40; 23/10",
			sentAt:     day(2025, time.October, 23),
			wantClass:  Rejected,
			wantBucket: Bucket("rejected: 4 fields"),
		},
		{
			name:       "swapped date and value is not repaired",
			text:       "Comida cinema; 109,39; 14/01",
			sentAt:     day(2025, time.January, 14),
			wantClass:  Rejected,
			wantBucket: BucketBadDate,
			wantErrIs:  parse.ErrInvalidDate,
		},
		{
			name:       "a space where the first semicolon belongs is not a repair D4 knows",
			text:       "San michel 22/12; 69,83",
			sentAt:     day(2025, time.December, 24),
			wantClass:  Rejected,
			wantBucket: Bucket("rejected: 2 fields"),
		},
		{
			name:       "conversation with a comma is still ignored before any repair",
			text:       "Não quer mais, né?",
			sentAt:     day(2025, time.May, 6),
			wantClass:  Ignored,
			wantBucket: BucketIgnored,
		},
		{
			name:       "receipt is untouched",
			text:       "",
			attachment: PhotoAttachment,
			sentAt:     day(2025, time.May, 4),
			wantClass:  Receipt,
			wantBucket: BucketReceipts,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.now.IsZero() {
				tt.now = defaultNow
			}
			m := Message{ID: 1, Text: tt.text, Attachment: tt.attachment, SentAt: tt.sentAt}
			assertRepairOutcome(t, Classify(m, tt.now), tt)
		})
	}
}

// assertRepairOutcome is the one verdict-verb of this table.
func assertRepairOutcome(t *testing.T, got Outcome, tt repairCase) {
	t.Helper()
	assert.Equal(t, tt.wantClass, got.Class)
	if tt.wantBucket != "" {
		assert.Equal(t, tt.wantBucket, got.Bucket())
	}
	switch got.Class {
	case Converted:
		assert.Equal(t, tt.wantLine, got.Line)
		assert.Equal(t, tt.wantRepair, got.Repair)
		assert.NoError(t, got.Err)
	case Rejected:
		require.Error(t, got.Err)
		if tt.wantErrIs != nil {
			assert.ErrorIs(t, got.Err, tt.wantErrIs)
		}
		if tt.wantReadings > 0 {
			var ambiguousErr AmbiguousRepairError
			require.True(t, errors.As(got.Err, &ambiguousErr))
			assert.Len(t, ambiguousErr.Readings, tt.wantReadings)
		}
	case Receipt, Ignored:
		assert.NoError(t, got.Err)
		assert.Empty(t, got.Line)
		assert.Empty(t, got.Repair)
	}
}

func TestAmbiguousRepairError_NamesEveryReading(t *testing.T) {
	msg := AmbiguousRepairError{Readings: []string{"a; 1/1; 1", "b; 1/1; 1"}}.Error()
	assert.Contains(t, msg, "2 different")
	assert.Contains(t, msg, "a; 1/1; 1")
	assert.Contains(t, msg, "b; 1/1; 1")
}
