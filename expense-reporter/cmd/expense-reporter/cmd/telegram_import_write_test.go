package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"expense-reporter/internal/capture"
	"expense-reporter/internal/parse"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func msg(id int, y int, m time.Month, d int, text string) capture.Message {
	return capture.Message{
		ID:         id,
		SentAt:     time.Date(y, m, d, 0, 0, 0, 0, time.UTC),
		Text:       text,
		Attachment: capture.NoAttachment,
	}
}

// fixtureOutcomes mirrors the telegram-import-dry-run acceptance fixture through the
// real classifier, so these tests and the acceptance scenarios describe one export.
func fixtureOutcomes() []capture.Outcome {
	now := time.Date(2026, 9, 3, 0, 0, 0, 0, time.UTC)
	msgs := []capture.Message{
		msg(1, 2025, time.May, 3, "Padaria; 03/05; 12,50"),
		msg(2, 2025, time.May, 4, "Mercado; 30/04; 1.234,56"),
		{ID: 3, SentAt: time.Date(2025, time.May, 4, 0, 0, 0, 0, time.UTC), Attachment: capture.PhotoAttachment},
		msg(5, 2025, time.May, 6, "Almoço teste, 06/05; 50,00"),
		msg(6, 2025, time.May, 6, "Não quer mais?"),
		msg(7, 2025, time.July, 5, "Uber; 05/07; 16:20"),
		msg(9, 2025, time.July, 6, "Cartao Teste; ADM; 09/01; 405,25 x4"),
		msg(11, 2025, time.July, 9, "Cinema; 08/07/2025; 45,00"),
		msg(14, 2025, time.May, 15, "Bar; 15/05/ 2025,50"),
	}
	return capture.ClassifyAll(msgs, now)
}

// The deterministic half of the step-3 gate: a line the converter writes is one
// batch-auto's own reader takes back to the same item, date and value. The repaired
// line proves the trailing audit comment is invisible to that reader.
func TestMonthFileLines_RoundTripThroughBatchAutoReader(t *testing.T) {
	months := capture.MonthFiles(fixtureOutcomes())
	require.GreaterOrEqual(t, len(months), 4, "the fixture spans four months; fewer means the split lost lines")
	for _, month := range months {
		for _, line := range month.Lines {
			t.Run(month.Name+"/"+line, func(t *testing.T) {
				pe, err := parse3FieldLine(line, parse.Options{})
				require.NoError(t, err)
				assert.True(t, strings.HasPrefix(line, pe.Item+";"+pe.DateString()+";"+pe.RawValue), line)
			})
		}
	}
}

func TestMonthFileBody_HeaderIsACommentTheReaderSkips(t *testing.T) {
	body := monthFileBody(capture.MonthFile{
		Name: "expenses-2025-05.csv",
		Lines: []string{
			"Padaria;03/05/2025;12,50",
			"Almoço teste;06/05/2025;50,00   # repaired from: Almoço teste, 06/05; 50,00 (message 5)",
		},
	})
	lines := strings.Split(strings.TrimSuffix(body, "\n"), "\n")
	require.Len(t, lines, 3)
	assert.Equal(t, "# expenses-2025-05.csv: expenses dated that month, converted by telegram-import. Text after '   # ' on a line is a comment batch-auto ignores.", lines[0])
	assert.Equal(t, "Padaria;03/05/2025;12,50", lines[1])
	assert.Equal(t, "Almoço teste;06/05/2025;50,00   # repaired from: Almoço teste, 06/05; 50,00 (message 5)", lines[2])
	assert.True(t, strings.HasSuffix(body, "\n"))
}

func TestWriteOutputs(t *testing.T) {
	months := capture.MonthFiles(fixtureOutcomes())
	rejects := capture.Rejects(fixtureOutcomes())

	t.Run("writes every month file and the rejects file", func(t *testing.T) {
		dir := t.TempDir()
		written, err := writeOutputs(dir, false, months, rejects)
		require.NoError(t, err)

		wantNames := []string{
			"expenses-2025-01.csv",
			"expenses-2025-04.csv",
			"expenses-2025-05.csv",
			"expenses-2025-07.csv",
			"telegram-rejects.csv",
		}
		wantLines := []int{1, 1, 2, 1, 2}

		require.Len(t, written, len(wantNames))
		for i, wantName := range wantNames {
			assert.Equal(t, wantName, written[i].Name)
			assert.Equal(t, wantLines[i], written[i].Lines)
		}
		for _, name := range wantNames {
			_, err = os.Stat(filepath.Join(dir, name))
			assert.NoError(t, err)
		}

		content, err := os.ReadFile(filepath.Join(dir, "telegram-rejects.csv"))
		require.NoError(t, err)
		assert.Contains(t, string(content), "Uber; 05/07; 16:20   # invalid value")
		assert.Contains(t, string(content), "(message 7, sent 2025-07-05)")
	})

	t.Run("refuses an existing month file, names it, offers --force, writes nothing", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "expenses-2025-05.csv"), []byte("sentinel\n"), 0o644))

		written, err := writeOutputs(dir, false, months, rejects)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "expenses-2025-05.csv")
		assert.Contains(t, err.Error(), "--force")
		assert.Empty(t, written)

		content, err := os.ReadFile(filepath.Join(dir, "expenses-2025-05.csv"))
		require.NoError(t, err)
		assert.Equal(t, "sentinel\n", string(content))

		for _, name := range []string{"expenses-2025-01.csv", "expenses-2025-04.csv", "expenses-2025-07.csv", "telegram-rejects.csv"} {
			_, err = os.Stat(filepath.Join(dir, name))
			assert.Error(t, err, name)
		}
	})

	t.Run("names every blocker, not only the first", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "expenses-2025-01.csv"), []byte(""), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "telegram-rejects.csv"), []byte(""), 0o644))

		written, err := writeOutputs(dir, false, months, rejects)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "expenses-2025-01.csv")
		assert.Contains(t, err.Error(), "telegram-rejects.csv")
		assert.Empty(t, written)
	})

	t.Run("force replaces the blockers", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "expenses-2025-05.csv"), []byte("sentinel\n"), 0o644))

		written, err := writeOutputs(dir, true, months, rejects)
		require.NoError(t, err)
		assert.Len(t, written, 5)

		content, err := os.ReadFile(filepath.Join(dir, "expenses-2025-05.csv"))
		require.NoError(t, err)
		assert.Contains(t, string(content), "Padaria;03/05/2025;12,50")
		assert.NotContains(t, string(content), "sentinel")
	})

	t.Run("no rejects means no rejects file", func(t *testing.T) {
		dir := t.TempDir()
		written, err := writeOutputs(dir, false, months, nil)
		require.NoError(t, err)

		for _, f := range written {
			assert.NotEqual(t, "telegram-rejects.csv", f.Name)
		}
		_, err = os.Stat(filepath.Join(dir, "telegram-rejects.csv"))
		assert.Error(t, err)
	})

	t.Run("missing directory is an error before anything is written", func(t *testing.T) {
		dir := filepath.Join(t.TempDir(), "absent")
		written, err := writeOutputs(dir, false, months, rejects)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not an existing directory")
		assert.Empty(t, written)

		_, err = os.Stat(filepath.Join(dir, "expenses-2025-01.csv"))
		assert.Error(t, err)
	})
}

func TestWriteFilesReport(t *testing.T) {
	t.Run("with files", func(t *testing.T) {
		var buf bytes.Buffer
		writeFilesReport(&buf, []writtenFile{
			{"expenses-2025-01.csv", 1},
			{"expenses-2025-05.csv", 2},
			{"telegram-rejects.csv", 4},
		})
		expected := "  files:\n    expenses-2025-01.csv   1 line\n    expenses-2025-05.csv   2 lines\n    telegram-rejects.csv   4 lines — repair each in place, then run the file through batch-auto\n"
		assert.Equal(t, expected, buf.String())
	})

	t.Run("no files", func(t *testing.T) {
		var buf bytes.Buffer
		writeFilesReport(&buf, []writtenFile{})
		assert.Empty(t, buf.String())
	})
}
