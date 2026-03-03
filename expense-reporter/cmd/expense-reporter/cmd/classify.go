package cmd

import (
	"expense-reporter/internal/classifier"
	"expense-reporter/pkg/utils"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var (
	classifyModel   string
	classifyTopN    int
	classifyDataDir string
)

var classifyCmd = &cobra.Command{
	Use:   "classify <item> <value> <DD/MM>",
	Short: "Classify an expense using a local AI model",
	Long: `Classify an expense and return the top subcategory candidates with confidence scores.

Examples:
  expense-reporter classify "Uber Centro" 35.50 15/04
  expense-reporter classify "Supermercado Extra" 210.00 03/01
  expense-reporter classify "Consulta veterinária" 180.00 10/03 --top 5`,
	Args: cobra.ExactArgs(3),
	RunE: runClassify,
}

func init() {
	rootCmd.AddCommand(classifyCmd)
	classifyCmd.Flags().StringVar(&classifyModel, "model", "my-classifier-q3", "Ollama model to use")
	classifyCmd.Flags().IntVar(&classifyTopN, "top", 3, "Number of candidates to return")
	classifyCmd.Flags().StringVar(&classifyDataDir, "data-dir", "data/classification", "Path to classification data directory")
}

func runClassify(cmd *cobra.Command, args []string) error {
	item := args[0]

	value, err := utils.ParseCurrency(args[1])
	if err != nil {
		return fmt.Errorf("invalid value %q: expected a number (e.g. 35.50 or 35,50)", args[1])
	}

	date := args[2]

	taxonomy, err := classifier.LoadTaxonomy(classifyDataDir)
	if err != nil {
		return fmt.Errorf("loading taxonomy: %w", err)
	}

	cfg := classifier.Config{
		OllamaURL: "http://localhost:11434",
		Model:     classifyModel,
		TopN:      classifyTopN,
	}

	results, err := classifier.Classify(item, value, date, taxonomy, cfg)
	if err != nil {
		return fmt.Errorf("classification failed: %w", err)
	}

	fmt.Printf("Classifying: %s  R$ %.2f  %s\n\n", item, value, date)
	for i, r := range results {
		bar := confidenceBar(r.Confidence)
		fmt.Printf("  %d. %-30s %-20s %s %.0f%%\n", i+1, r.Subcategory, r.Category, bar, r.Confidence*100)
	}
	return nil
}

func confidenceBar(confidence float64) string {
	filled := int(confidence * 10)
	if filled > 10 {
		filled = 10
	}
	return "[" + strings.Repeat("█", filled) + strings.Repeat("░", 10-filled) + "]"
}
