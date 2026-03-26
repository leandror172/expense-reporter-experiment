package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"expense-reporter/internal/classifier"
)

func TestToCandidates(t *testing.T) {
	results := []classifier.Result{
		{
			Subcategory: "Uber/Taxi",
			Category:    "Transporte",
			Confidence:  0.95,
		},
		{
			Subcategory: "Supermercado",
			Category:    "Alimentação",
			Confidence:  0.87,
		},
	}

	candidates := toCandidates(results)

	require.Len(t, candidates, 2)
	assert.Equal(t, "Uber/Taxi", candidates[0].Subcategory)
	assert.Equal(t, "Transporte", candidates[0].Category)
	assert.Equal(t, 0.95, candidates[0].Confidence)
	assert.Equal(t, "Supermercado", candidates[1].Subcategory)
	assert.Equal(t, "Alimentação", candidates[1].Category)
	assert.Equal(t, 0.87, candidates[1].Confidence)
}

func TestClassifyOutputJSON(t *testing.T) {
	output := ClassifyOutput{
		Item:  "Uber Centro",
		Value: 35.50,
		Date:  "15/04",
		Candidates: []CandidateOutput{
			{Subcategory: "Uber/Taxi", Category: "Transporte", Confidence: 0.95},
			{Subcategory: "Ônibus", Category: "Transporte", Confidence: 0.42},
		},
	}

	jsonData, err := json.Marshal(output)
	require.NoError(t, err)

	var parsed ClassifyOutput
	err = json.Unmarshal(jsonData, &parsed)
	require.NoError(t, err)

	assert.Equal(t, output.Item, parsed.Item)
	assert.Equal(t, output.Value, parsed.Value)
	assert.Equal(t, output.Date, parsed.Date)
	require.Len(t, parsed.Candidates, 2)
	assert.Equal(t, "Uber/Taxi", parsed.Candidates[0].Subcategory)
	assert.Equal(t, 0.95, parsed.Candidates[0].Confidence)

	// Verify candidates key exists in raw JSON
	assert.Contains(t, string(jsonData), `"candidates"`)
}

func TestAutoOutputJSON(t *testing.T) {
	result := &CandidateOutput{
		Subcategory: "Uber/Taxi",
		Category:    "Transporte",
		Confidence:  0.95,
	}

	output := AutoOutput{
		Item:       "Uber Centro",
		Value:      35.50,
		Date:       "15/04",
		Action:     "would_insert",
		Result:     result,
		Candidates: []CandidateOutput{*result},
		Message:    "Uber Centro → Uber/Taxi (Transporte) — 95% confidence, ready to insert",
	}

	jsonData, err := json.Marshal(output)
	require.NoError(t, err)

	var parsed AutoOutput
	err = json.Unmarshal(jsonData, &parsed)
	require.NoError(t, err)

	assert.Equal(t, "would_insert", parsed.Action)
	require.NotNil(t, parsed.Result)
	assert.Equal(t, *result, *parsed.Result)
	assert.Len(t, parsed.Candidates, 1)
	assert.Equal(t, output.Message, parsed.Message)
}

func TestAutoOutputJSON_NilResult(t *testing.T) {
	output := AutoOutput{
		Item:       "Restaurante Desconhecido",
		Value:      89.90,
		Date:       "03/01",
		Action:     "review",
		Result:     nil,
		Candidates: []CandidateOutput{{Subcategory: "Alimentação", Category: "Alimentação", Confidence: 0.45}},
		Message:    "top confidence 45% is below threshold 85%",
	}

	jsonData, err := json.Marshal(output)
	require.NoError(t, err)

	// omitempty: "result" key should be absent when nil
	assert.NotContains(t, string(jsonData), `"result":`)
}

func TestPrintJSON(t *testing.T) {
	testData := ClassifyOutput{
		Item:  "Test",
		Value: 10.0,
		Date:  "01/01",
		Candidates: []CandidateOutput{
			{Subcategory: "Sub", Category: "Cat", Confidence: 0.5},
		},
	}

	// Capture stdout via pipe
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)

	os.Stdout = w
	printErr := printJSON(testData)
	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, err = buf.ReadFrom(r)
	require.NoError(t, err)
	r.Close()

	require.NoError(t, printErr)

	output := strings.TrimSpace(buf.String())

	// Verify valid JSON
	var parsed ClassifyOutput
	err = json.Unmarshal([]byte(output), &parsed)
	require.NoError(t, err)
	assert.Equal(t, "Test", parsed.Item)

	// Verify 2-space indent
	assert.Contains(t, output, "  \"item\":")
}
