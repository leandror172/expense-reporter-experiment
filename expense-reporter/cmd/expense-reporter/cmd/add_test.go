package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
	runErr := runAddDryRun(cmd, "Uber Centro", "15/04", 35.50, "Uber/Taxi", "Transporte")
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
	runErr := runAddDryRun(cmd, "Uber Centro", "15/04", 35.50, "Uber/Taxi", "Transporte")
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
	assert.Contains(t, output, "Subcategory: Uber/Taxi")
	assert.Contains(t, output, "Category:    Transporte")
}

func TestRunAddDryRun_Text_EmptyCategory(t *testing.T) {
	cmd := newCmdWithJSONFlag(false)

	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)

	os.Stdout = w
	runErr := runAddDryRun(cmd, "Coffee", "03/01", 12.90, "Cafeteria", "")
	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, err = buf.ReadFrom(r)
	require.NoError(t, err)
	r.Close()

	require.NoError(t, runErr)

	output := buf.String()
	assert.Contains(t, output, "Dry run — would insert:")
	assert.NotContains(t, output, "Category:")
}
