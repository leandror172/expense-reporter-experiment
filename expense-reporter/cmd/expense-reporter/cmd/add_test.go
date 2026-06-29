package cmd

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"expense-reporter/internal/config"
	"expense-reporter/internal/feedback"
	taxonomy "expense-reporter/internal/taxonomy"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// addTestSheets is a small taxonomy with a unique leaf (Spotify) and a leaf that
// repeats across types (Dentista ∈ Variáveis, Extras — both under Saúde), used to
// drive resolveFullPath's disambiguation branches.
func addTestSheets() []taxonomy.ExpenseType {
	return []taxonomy.ExpenseType{
		{Name: "Fixas", Cats: []taxonomy.Category{
			{Name: "Assinaturas", Subs: []taxonomy.Subcat{{Name: "Spotify"}}},
		}},
		{Name: "Variáveis", Cats: []taxonomy.Category{
			{Name: "Saúde", Subs: []taxonomy.Subcat{{Name: "Dentista"}}},
		}},
		{Name: "Extras", Cats: []taxonomy.Category{
			{Name: "Saúde", Subs: []taxonomy.Subcat{{Name: "Dentista"}}},
		}},
	}
}

// TestResolveFullPath covers the T-13 add-path resolution hybrid. The Go test
// process is not a TTY, so the ambiguous-without-flag case exercises the
// non-interactive hard-error branch (never a silent guess).
func TestResolveFullPath(t *testing.T) {
	sheets := addTestSheets()

	noInput := strings.NewReader("")

	t.Run("unambiguous leaf resolves and ignores --type", func(t *testing.T) {
		typ, cat, err := resolveFullPath(sheets, "Spotify", "Variáveis", false, noInput)
		require.NoError(t, err)
		assert.Equal(t, "Fixas", typ)
		assert.Equal(t, "Assinaturas", cat)
	})

	t.Run("ambiguous leaf resolves with correct --type", func(t *testing.T) {
		typ, cat, err := resolveFullPath(sheets, "Dentista", "Extras", false, noInput)
		require.NoError(t, err)
		assert.Equal(t, "Extras", typ)
		assert.Equal(t, "Saúde", cat)
	})

	t.Run("ambiguous leaf with wrong --type errors and lists valid types", func(t *testing.T) {
		_, _, err := resolveFullPath(sheets, "Dentista", "Fixas", false, noInput)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Variáveis")
		assert.Contains(t, err.Error(), "Extras")
	})

	t.Run("ambiguous leaf without --type hard-errors when non-interactive", func(t *testing.T) {
		_, _, err := resolveFullPath(sheets, "Dentista", "", false, noInput)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "--type")
	})

	t.Run("ambiguous leaf resolved by interactive prompt", func(t *testing.T) {
		typ, cat, err := resolveFullPath(sheets, "Dentista", "", true, strings.NewReader("Extras\n"))
		require.NoError(t, err)
		assert.Equal(t, "Extras", typ)
		assert.Equal(t, "Saúde", cat)
	})

	t.Run("absent leaf errors", func(t *testing.T) {
		_, _, err := resolveFullPath(sheets, "Inexistente", "", false, noInput)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not in the taxonomy")
	})
}

// newCmdWithJSONFlag creates a cobra.Command with a --json flag for testing.
func newCmdWithJSONFlag(jsonEnabled bool) *cobra.Command {
	cmd := &cobra.Command{}
	cmd.Flags().Bool("json", false, "")
	if jsonEnabled {
		cmd.Flags().Set("json", "true")
	}
	return cmd
}

func TestAddOutputJSON(t *testing.T) {
	tests := []struct {
		name  string
		input AddOutput
	}{
		{
			name: "all fields populated",
			input: AddOutput{
				Item:        "Uber Centro",
				Value:       35.50,
				Date:        "15/04",
				Subcategory: "Uber/Taxi",
				Category:    "Transporte",
				Action:      "would_insert",
			},
		},
		{
			name: "empty category from failed taxonomy lookup",
			input: AddOutput{
				Item:        "Coffee Shop",
				Value:       12.90,
				Date:        "03/01",
				Subcategory: "Cafeteria",
				Category:    "",
				Action:      "would_insert",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			jsonData, err := json.Marshal(tc.input)
			require.NoError(t, err)

			var parsed AddOutput
			err = json.Unmarshal(jsonData, &parsed)
			require.NoError(t, err)

			assert.Equal(t, tc.input, parsed)

			// All fields present in raw JSON (no omitempty hiding empty category)
			raw := string(jsonData)
			for _, key := range []string{"item", "value", "date", "subcategory", "category", "action"} {
				assert.Contains(t, raw, fmt.Sprintf(`"%s"`, key))
			}
		})
	}
}

func TestRunAddDryRun_JSON(t *testing.T) {
	cmd := newCmdWithJSONFlag(true)

	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)

	os.Stdout = w
	runErr := runAddDryRun(cmd, "Uber Centro", "15/04", 35.50, "Variáveis", "Uber/Taxi", "Transporte")
	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, err = buf.ReadFrom(r)
	require.NoError(t, err)
	r.Close()

	require.NoError(t, runErr)

	var parsed AddOutput
	err = json.Unmarshal(buf.Bytes(), &parsed)
	require.NoError(t, err)

	assert.Equal(t, "Uber Centro", parsed.Item)
	assert.Equal(t, 35.50, parsed.Value)
	assert.Equal(t, "15/04", parsed.Date)
	assert.Equal(t, "Variáveis", parsed.Type)
	assert.Equal(t, "Uber/Taxi", parsed.Subcategory)
	assert.Equal(t, "Transporte", parsed.Category)
	assert.Equal(t, "would_insert", parsed.Action)
}

func TestRunAddDryRun_Text(t *testing.T) {
	cmd := newCmdWithJSONFlag(false)

	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)

	os.Stdout = w
	runErr := runAddDryRun(cmd, "Uber Centro", "15/04", 35.50, "Variáveis", "Uber/Taxi", "Transporte")
	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, err = buf.ReadFrom(r)
	require.NoError(t, err)
	r.Close()

	require.NoError(t, runErr)

	output := buf.String()
	assert.Contains(t, output, "Dry run — would insert:")
	assert.Contains(t, output, "Item:        Uber Centro")
	assert.Contains(t, output, "Date:        15/04")
	assert.Contains(t, output, "Value:       35.50")
	assert.Contains(t, output, "Type:        Variáveis")
	assert.Contains(t, output, "Subcategory: Uber/Taxi")
	assert.Contains(t, output, "Category:    Transporte")
}

func TestRunAddDryRun_Text_EmptyCategory(t *testing.T) {
	cmd := newCmdWithJSONFlag(false)

	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)

	os.Stdout = w
	runErr := runAddDryRun(cmd, "Coffee", "03/01", 12.90, "", "Cafeteria", "")
	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, err = buf.ReadFrom(r)
	require.NoError(t, err)
	r.Close()

	require.NoError(t, runErr)

	output := buf.String()
	assert.Contains(t, output, "Dry run — would insert:")
	assert.NotContains(t, output, "Type:")
	assert.NotContains(t, output, "Category:")
}

func TestLogPredictedFeedback(t *testing.T) {
	tests := []struct {
		name                  string
		chosenSubcategory     string
		chosenCategory        string
		predictedSubcategory  string
		predictedCategory     string
		classificationID      string
		confidence            float64
		model                 string
		noClassificationsPath bool
		expectStatus          feedback.Status
		expectStderrContains  string
	}{
		{
			name:                 "chosen matches predicted — confirmed entry written",
			chosenSubcategory:    "Uber/Taxi",
			chosenCategory:       "Transporte",
			predictedSubcategory: "Uber/Taxi",
			predictedCategory:    "Transporte",
			classificationID:     "",
			confidence:           0.92,
			model:                "my-classifier-q3",
			expectStatus:         feedback.StatusConfirmed,
		},
		{
			name:                 "chosen differs from predicted — corrected entry written",
			chosenSubcategory:    "Combustível",
			chosenCategory:       "Transporte",
			predictedSubcategory: "Uber/Taxi",
			predictedCategory:    "Transporte",
			classificationID:     "",
			confidence:           0.92,
			model:                "my-classifier-q3",
			expectStatus:         feedback.StatusCorrected,
		},
		{
			name:                 "classification-id not found — stderr warning, entry still written",
			chosenSubcategory:    "Combustível",
			chosenCategory:       "Transporte",
			predictedSubcategory: "Uber/Taxi",
			predictedCategory:    "Transporte",
			classificationID:     "nonexistent123",
			confidence:           0.85,
			model:                "my-classifier-q3",
			expectStatus:         feedback.StatusCorrected,
			expectStderrContains: `classification-id "nonexistent123" not found`,
		},
		{
			name:                  "no classifications path configured — no file written, no panic",
			chosenSubcategory:     "Uber/Taxi",
			chosenCategory:        "Transporte",
			predictedSubcategory:  "Uber/Taxi",
			predictedCategory:     "Transporte",
			noClassificationsPath: true,
			confidence:            0.92,
			model:                 "my-classifier-q3",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tempDir := t.TempDir()
			classificationsPath := filepath.Join(tempDir, "classifications.jsonl")
			expensesLogPath := filepath.Join(tempDir, "expenses_log.jsonl")

			var appCfg *config.Config
			if tc.noClassificationsPath {
				appCfg = &config.Config{ExpensesLogPath: expensesLogPath}
			} else {
				appCfg = &config.Config{
					ClassificationsPath: classificationsPath,
					ExpensesLogPath:     expensesLogPath,
				}
			}

			// Capture stderr
			r, w, err := os.Pipe()
			require.NoError(t, err)
			oldStderr := os.Stderr
			os.Stderr = w

			logPredictedFeedback(appCfg,
				"Uber Centro", "15/04", 35.50,
				tc.chosenSubcategory, tc.chosenCategory,
				tc.predictedSubcategory, tc.predictedCategory,
				tc.classificationID,
				tc.confidence, tc.model,
			)

			w.Close()
			os.Stderr = oldStderr
			stderrBytes, err := io.ReadAll(r)
			require.NoError(t, err)

			if tc.expectStderrContains != "" {
				assert.Contains(t, string(stderrBytes), tc.expectStderrContains)
			} else {
				assert.Empty(t, string(stderrBytes))
			}

			if tc.noClassificationsPath {
				_, statErr := os.Stat(classificationsPath)
				assert.True(t, os.IsNotExist(statErr), "classifications file should not be created when path is empty")
				return
			}

			f, err := os.Open(classificationsPath)
			require.NoError(t, err)
			defer f.Close()

			scanner := bufio.NewScanner(f)
			require.True(t, scanner.Scan(), "expected one line in classifications file")
			var entry feedback.Entry
			require.NoError(t, json.Unmarshal(scanner.Bytes(), &entry))

			assert.Equal(t, tc.expectStatus, entry.Status)
			assert.Equal(t, tc.predictedSubcategory, entry.PredictedSubcategory)
			assert.Equal(t, tc.predictedCategory, entry.PredictedCategory)
			assert.Equal(t, tc.chosenSubcategory, entry.ActualSubcategory)
			assert.Equal(t, tc.chosenCategory, entry.ActualCategory)
			assert.Equal(t, tc.confidence, entry.Confidence)
			assert.Equal(t, tc.model, entry.Model)
		})
	}
}
