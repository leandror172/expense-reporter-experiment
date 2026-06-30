package cmd

import (
	"expense-reporter/internal/classifier"
	"expense-reporter/internal/config"
	taxonomy "expense-reporter/internal/taxonomy"
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

	appCfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	sheets, err := loadTaxonomyTree(appCfg)
	if err != nil {
		return err
	}

	cfg := classifier.Config{
		OllamaURL: "http://localhost:11434",
		Model:     classifyModel,
		DataDir:   classifyDataDir,
		TopN:      classifyTopN,
	}

	results, err := classifier.Classify(item, value, date, sheets, cfg)
	if err != nil {
		return fmt.Errorf("classification failed: %w", err)
	}

	if outputJSON {
		return printJSON(ClassifyOutput{
			Item:       item,
			Value:      value,
			Date:       date,
			Candidates: toCandidates(results),
		})
	}

	fmt.Printf("Classifying: %s  R$ %.2f  %s\n\n", item, value, date)
	for i, r := range results {
		bar := confidenceBar(r.Confidence)
		fmt.Printf("  %d. %-30s %-20s %s %.0f%%\n", i+1, r.Subcategory, r.Category, bar, r.Confidence*100)
	}
	return nil
}

// loadTaxonomyTree loads the expense taxonomy tree (config/taxonomy.json) used by
// the classifier to constrain predictions to valid full paths (T-13). It is the
// single taxonomy source for classification — the feature dictionary is no longer
// consulted for category/type resolution.
func loadTaxonomyTree(appCfg *config.Config) ([]taxonomy.ExpenseType, error) {
	path := appCfg.TaxonomyFilePath()
	if path == "" {
		return nil, fmt.Errorf("taxonomy path not configured")
	}
	types, _, err := taxonomy.LoadTaxonomy(path, "", "", 0)
	if err != nil {
		return nil, fmt.Errorf("loading taxonomy: %w", err)
	}
	return types, nil
}

func confidenceBar(confidence float64) string {
	filled := int(confidence * 10)
	if filled > 10 {
		filled = 10
	}
	return "[" + strings.Repeat("█", filled) + strings.Repeat("░", 10-filled) + "]"
}
