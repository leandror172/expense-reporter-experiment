package capture

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// sent is the message builder of these tests: a typed text with no attachment.
func sent(id int, sentAt time.Time, text string) Message {
	return Message{ID: id, SentAt: sentAt, Text: text, Attachment: NoAttachment}
}

// outcomesOf runs the REAL Classify over each message, so the sink is fed exactly what
// production feeds it — never struct literals.
func outcomesOf(msgs ...Message) []Outcome {
	outcomes := make([]Outcome, 0, len(msgs))
	for _, msg := range msgs {
		outcomes = append(outcomes, Classify(msg, defaultNow))
	}
	return outcomes
}

func TestMonthFileName(t *testing.T) {
	tests := []struct {
		name string
		date time.Time
		want string
	}{
		{"january", day(2025, time.January, 9), "expenses-2025-01.csv"},
		{"december", day(2025, time.December, 31), "expenses-2025-12.csv"},
		{"july", day(2026, time.July, 1), "expenses-2026-07.csv"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, MonthFileName(tt.date))
		})
	}
}

// Files come back sorted by name; inside a file, lines keep stream order. A line lands
// in the month of its RESOLVED date (plan D7): message 2 was sent in May for an April
// purchase, message 9 was sent in July for a January one. A repaired line carries its
// origin as a trailing comment batch-auto's reader strips. Receipts, conversation and
// rejects contribute nothing.
func TestMonthFiles_SplitsByResolvedDateInStreamOrder(t *testing.T) {
	msgs := []Message{
		sent(1, day(2025, time.May, 3), "Padaria; 03/05; 12,50"),
		sent(2, day(2025, time.May, 4), "Mercado; 30/04; 1.234,56"),
		{ID: 3, SentAt: day(2025, time.May, 4), Attachment: PhotoAttachment},
		sent(5, day(2025, time.May, 6), "Almoço teste, 06/05; 50,00"),
		sent(6, day(2025, time.May, 6), "Não quer mais?"),
		sent(7, day(2025, time.July, 5), "Uber; 05/07; 16:20"),
		sent(9, day(2025, time.July, 6), "Cartao Teste; ADM; 09/01; 405,25 x4"),
		sent(11, day(2025, time.July, 9), "Cinema; 08/07/2025; 45,00"),
	}
	outcomes := outcomesOf(msgs...)

	got := MonthFiles(outcomes)
	want := []MonthFile{
		{Name: "expenses-2025-01.csv", Lines: []string{"Cartao Teste ADM;09/01/2025;405,25 x4   # repaired from: Cartao Teste; ADM; 09/01; 405,25 x4 (message 9)"}},
		{Name: "expenses-2025-04.csv", Lines: []string{"Mercado;30/04/2025;1.234,56"}},
		{Name: "expenses-2025-05.csv", Lines: []string{"Padaria;03/05/2025;12,50", "Almoço teste;06/05/2025;50,00   # repaired from: Almoço teste, 06/05; 50,00 (message 5)"}},
		{Name: "expenses-2025-07.csv", Lines: []string{"Cinema;08/07/2025;45,00"}},
	}
	assert.Equal(t, want, got)
}

func TestMonthFiles_NothingConvertedMeansNoFiles(t *testing.T) {
	msgs := []Message{
		sent(7, day(2025, time.July, 5), "Uber; 05/07; 16:20"),
		sent(6, day(2025, time.May, 6), "Não quer mais?"),
	}
	outcomes := outcomesOf(msgs...)

	assert.Empty(t, MonthFiles(outcomes))
	assert.Empty(t, MonthFiles(nil))
}

// Receipts and conversation are not rejects (plan D3). The provenance suffix — message
// id and send date — is what lets a human fix a year by hand when re-running the file.
func TestRejects_ListsAttemptedExpensesWithProvenance(t *testing.T) {
	msgs := []Message{
		sent(1, day(2025, time.May, 3), "Padaria; 03/05; 12,50"),
		sent(2, day(2025, time.May, 4), "Mercado; 30/04; 1.234,56"),
		{ID: 3, SentAt: day(2025, time.May, 4), Attachment: PhotoAttachment},
		sent(5, day(2025, time.May, 6), "Almoço teste, 06/05; 50,00"),
		sent(6, day(2025, time.May, 6), "Não quer mais?"),
		sent(7, day(2025, time.July, 5), "Uber; 05/07; 16:20"),
		sent(9, day(2025, time.July, 6), "Cartao Teste; ADM; 09/01; 405,25 x4"),
		sent(11, day(2025, time.July, 9), "Cinema; 08/07/2025; 45,00"),
		sent(14, day(2025, time.May, 15), "Bar; 15/05/ 2025,50"),
	}
	outcomes := outcomesOf(msgs...)

	got := Rejects(outcomes)
	require.Len(t, got, 2)

	assert.Equal(t, "Uber; 05/07; 16:20", got[0].Text)
	assert.True(t, strings.HasPrefix(got[0].Reason, "invalid value"), got[0].Reason)
	assert.True(t, strings.HasSuffix(got[0].Reason, "(message 7, sent 2025-07-05)"), got[0].Reason)

	assert.Equal(t, "Bar; 15/05/ 2025,50", got[1].Text)
	assert.True(t, strings.HasPrefix(got[1].Reason, "ambiguous: 2 different"), got[1].Reason)
	assert.True(t, strings.HasSuffix(got[1].Reason, "(message 14, sent 2025-05-15)"), got[1].Reason)
}

func TestRejects_FlattensLineBreaksInTheText(t *testing.T) {
	msgs := []Message{
		sent(7, day(2025, time.July, 5), "Uber; 05/07; 16:20\nobs"),
	}
	outcomes := outcomesOf(msgs...)

	got := Rejects(outcomes)
	require.Len(t, got, 1)
	assert.Equal(t, "Uber; 05/07; 16:20 obs", got[0].Text)
	assert.True(t, strings.HasSuffix(got[0].Reason, "(message 7, sent 2025-07-05)"), got[0].Reason)
}

func TestRejects_NothingRejectedMeansNoRows(t *testing.T) {
	msgs := []Message{
		sent(1, day(2025, time.May, 3), "Padaria; 03/05; 12,50"),
	}
	outcomes := outcomesOf(msgs...)

	assert.Empty(t, Rejects(outcomes))
}
