package feedback

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadExpenseIDCounts(t *testing.T) {
	t.Run("missing file returns empty map and nil error", func(t *testing.T) {
		dir := t.TempDir()
		path := dir + "/nonexistent.jsonl"

		counts, err := LoadExpenseIDCounts(path)
		require.NoError(t, err)
		assert.Empty(t, counts)
	})

	t.Run("empty file returns empty map and nil error", func(t *testing.T) {
		dir := t.TempDir()
		path := dir + "/empty.jsonl"
		require.NoError(t, os.WriteFile(path, []byte(""), 0o644))

		counts, err := LoadExpenseIDCounts(path)
		require.NoError(t, err)
		assert.Empty(t, counts)
	})

	t.Run("single entry counts once", func(t *testing.T) {
		dir := t.TempDir()
		path := dir + "/single.jsonl"

		entry1 := NewExpenseEntry("Item 1", "01/01/2025", 10.0, "Subcat 1", "Cat 1")
		data1, err := json.Marshal(entry1)
		require.NoError(t, err)

		require.NoError(t, os.WriteFile(path, append(data1, '\n'), 0o644))

		counts, err := LoadExpenseIDCounts(path)
		require.NoError(t, err)
		assert.Equal(t, map[string]int{entry1.ID: 1}, counts)
	})

	t.Run("multiple entries with same id count multiple times", func(t *testing.T) {
		dir := t.TempDir()
		path := dir + "/duplicate.jsonl"

		entry1 := NewExpenseEntry("Item 1", "01/01/2025", 10.0, "Subcat 1", "Cat 1")
		data1, err := json.Marshal(entry1)
		require.NoError(t, err)

		entry2 := NewExpenseEntry("Item 2", "02/01/2025", 20.0, "Subcat 2", "Cat 2")
		data2, err := json.Marshal(entry2)
		require.NoError(t, err)

		// Write same entry twice
		content := append(append(data1, '\n'), data1...)
		content = append(content, '\n')
		content = append(content, data2...)
		content = append(content, '\n')

		require.NoError(t, os.WriteFile(path, content, 0o644))

		counts, err := LoadExpenseIDCounts(path)
		require.NoError(t, err)
		assert.Equal(t, map[string]int{entry1.ID: 2, entry2.ID: 1}, counts)
	})

	t.Run("blank lines are skipped", func(t *testing.T) {
		dir := t.TempDir()
		path := dir + "/blanks.jsonl"

		entry1 := NewExpenseEntry("Item 1", "01/01/2025", 10.0, "Subcat 1", "Cat 1")
		data1, err := json.Marshal(entry1)
		require.NoError(t, err)

		content := []byte("\n\n" + string(data1) + "\n   \n" + string(data1) + "\n\n")

		require.NoError(t, os.WriteFile(path, content, 0o644))

		counts, err := LoadExpenseIDCounts(path)
		require.NoError(t, err)
		assert.Equal(t, map[string]int{entry1.ID: 2}, counts)
	})

	t.Run("malformed line returns error", func(t *testing.T) {
		dir := t.TempDir()
		path := dir + "/malformed.jsonl"

		entry1 := NewExpenseEntry("Item 1", "01/01/2025", 10.0, "Subcat 1", "Cat 1")
		data1, err := json.Marshal(entry1)
		require.NoError(t, err)

		content := []byte(string(data1) + "\n" + "{malformed json}\n" + string(data1))

		require.NoError(t, os.WriteFile(path, content, 0o644))

		counts, err := LoadExpenseIDCounts(path)
		assert.Error(t, err)
		assert.Nil(t, counts)
	})
}
