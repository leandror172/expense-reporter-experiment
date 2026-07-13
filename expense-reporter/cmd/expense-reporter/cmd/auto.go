package cmd

import (
	"bufio"
	"expense-reporter/internal/appender"
	"expense-reporter/internal/classifier"
	"expense-reporter/internal/config"
	"expense-reporter/internal/feedback"
	"expense-reporter/pkg/utils"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var (
	autoModel   string
	autoDataDir string
	autoConfirm bool
	autoThink   bool
)

var autoCmd = &cobra.Command{
	Use:   "auto <item> <value> <DD/MM>",
	Short: "Classify and auto-insert on a confident keyword agreement",
	Long: `Classify an expense and append it automatically when the model's prediction
agrees with an unambiguous, high-specificity keyword match (the agreement gate).
Otherwise prints candidates for manual review.

Examples:
  expense-reporter auto "Uber Centro" 35.50 15/04
  expense-reporter auto "Diarista Letícia" 160,00 05/01 --confirm`,
	Args: cobra.ExactArgs(3),
	RunE: runAuto,
}

func init() {
	rootCmd.AddCommand(autoCmd)
	autoCmd.Flags().StringVar(&autoModel, "model", "my-classifier-q3", "Ollama model to use")
	autoCmd.Flags().StringVar(&autoDataDir, "data-dir", "data/classification", "Path to classification data directory")
	autoCmd.Flags().BoolVar(&autoConfirm, "confirm", false, "Always ask for confirmation before inserting")
	autoCmd.Flags().BoolVar(&autoThink, "think", true, "Allow the model to emit thinking tokens (false = faster, sends think:false)")
}

func runAuto(cmd *cobra.Command, args []string) error {
	item := args[0]

	value, installmentCount, err := utils.ParseCurrencyWithInstallments(args[1])
	if err != nil {
		return fmt.Errorf("invalid value %q: expected a number (e.g. 35.50 or 35,50) or with installments (e.g. 35,50/3)", args[1])
	}

	date := args[2]

	parsedDate, err := utils.ParseDateFlexible(date)
	if err != nil {
		return fmt.Errorf("invalid date %q: expected DD/MM or DD/MM/YYYY", date)
	}

	appCfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	sheets, err := loadTaxonomyTree(appCfg)
	if err != nil {
		return err
	}

	cfg := classifier.Config{
		OllamaURL:    "http://localhost:11434",
		Model:        autoModel,
		DataDir:      autoDataDir,
		FeedbackPath: appCfg.ClassificationsFilePath(),
		TopN:         3,
		NoThink:      !autoThink,
	}

	results, err := classifier.Classify(item, value, date, sheets, cfg)
	if err != nil {
		return fmt.Errorf("classification failed: %w", err)
	}

	if len(results) == 0 {
		return fmt.Errorf("classifier returned no results")
	}

	top := results[0]

	// Agreement gate: the keyword signal is a pure function of the item, computed
	// once here and shared by the JSON and interactive paths. A nil index (load
	// failure) yields a miss → the item routes to review.
	keywords, kerr := classifier.LoadKeywordIndex(autoDataDir)
	if kerr != nil {
		keywords = nil
	}
	signal := classifier.MatchStrength(item, keywords)

	// JSON mode: read-only — classify and return recommendation, never insert.
	if outputJSON {
		topCandidate := &CandidateOutput{
			Subcategory: top.Subcategory,
			Category:    top.Category,
			Confidence:  top.Confidence,
		}

		var action, message string
		if classifier.IsAutoInsertable(top, signal, appCfg.AutoInsertExcluded) {
			action = "would_insert"
			message = fmt.Sprintf("%s → %s (%s) — agrees with keyword match, ready to insert",
				item, top.Subcategory, top.Category)
		} else if classifier.IsExcluded(top.Subcategory, appCfg.AutoInsertExcluded) {
			action = "excluded"
			message = fmt.Sprintf("%q is excluded from auto-insert", top.Subcategory)
		} else {
			action = "review"
			message = gateReviewReason(top, signal)
		}

		return printJSON(AutoOutput{
			Item:             item,
			Value:            value,
			Date:             date,
			Action:           action,
			Result:           topCandidate,
			Candidates:       toCandidates(results),
			Message:          message,
			ClassificationID: feedback.GenerateID(item, date, value),
		})
	}

	if classifier.IsAutoInsertable(top, signal, appCfg.AutoInsertExcluded) {
		if autoConfirm {
			fmt.Printf("Top match: %s (%s) — agrees with keyword match\n", top.Subcategory, top.Category)
			fmt.Printf("Insert? [y/N] ")
			if !confirmInsert(os.Stdin) {
				printCandidates(item, value, date, results)
				fmt.Println("\n⚠  Not appended — cancelled by user.")
				return nil
			}
		}
		return appendExpense(item, date, parsedDate, value, installmentCount, top, appCfg)
	}

	printCandidates(item, value, date, results)
	if classifier.IsExcluded(top.Subcategory, appCfg.AutoInsertExcluded) {
		fmt.Printf("\n⚠  Not appended — \"%s\" is excluded from auto-insert.\n", top.Subcategory)
	} else {
		fmt.Printf("\n⚠  Not appended — %s\n", gateReviewReason(top, signal))
	}
	return nil
}

// gateReviewReason explains, for a result that failed the agreement gate but is not
// excluded, which gate condition it missed — so review output names the cause
// (no keyword match, ambiguous keyword, low specificity, or model/keyword disagreement)
// instead of a stale confidence threshold.
func gateReviewReason(top classifier.Result, signal classifier.MatchSignal) string {
	switch {
	case !signal.Matched:
		return "no keyword match to confirm the prediction"
	case signal.Ambiguous:
		return "the keyword match is ambiguous across subcategories"
	case signal.TopScore < 1.0:
		return fmt.Sprintf("keyword specificity %.2f is below the maximum required for auto-insert", signal.TopScore)
	case top.Subcategory != signal.TopSubcategory:
		return fmt.Sprintf("model predicted %q but the keyword match points to %q", top.Subcategory, signal.TopSubcategory)
	default:
		return "did not meet the auto-insert gate"
	}
}

func appendExpense(item, date string, parsedDate time.Time, value float64, installmentCount int, result classifier.Result, appCfg *config.Config) error {
	// T-13: the type comes straight from the predicted full path — no post-hoc
	// (category, subcategory) lookup that could fail or disagree.
	logPath := appCfg.ExpensesLogFilePath()
	if logPath == "" {
		fmt.Fprintf(os.Stderr, "⚠  expense log: no path configured\n")
	} else {
		if err := appender.ExpandAndAppend(logPath, item, parsedDate, value, installmentCount, result.Type, result.Category, result.Subcategory); err != nil {
			fmt.Fprintf(os.Stderr, "⚠  expense log append failed: %v\n", err)
		}
	}

	fmt.Printf("✓ Appended: %s → %s (%s) — %.0f%% confidence\n",
		item, result.Subcategory, result.Category, result.Confidence*100)
	logConfirmedFeedback(appCfg, item, date, value, result, autoModel)
	return nil
}

// logConfirmedFeedback appends a confirmed entry to classifications.jsonl.
// Non-fatal: logs a warning to stderr if the write fails.
func logConfirmedFeedback(appCfg *config.Config, item, date string, value float64, result classifier.Result, model string) {
	path := appCfg.ClassificationsFilePath()
	if path == "" {
		return
	}
	entry := feedback.NewConfirmedEntry(item, date, value, result, model)
	if err := feedback.Append(path, entry); err != nil {
		fmt.Fprintf(os.Stderr, "⚠  feedback log: %v\n", err)
	}
}

func printCandidates(item string, value float64, date string, results []classifier.Result) {
	fmt.Printf("Classifying: %s  R$ %.2f  %s\n\n", item, value, date)
	for i, r := range results {
		bar := confidenceBar(r.Confidence)
		fmt.Printf("  %d. %-30s %-20s %s %.0f%%\n", i+1, r.Subcategory, r.Category, bar, r.Confidence*100)
	}
}

// confirmInsert reads a line from r and returns true only for "y" or "yes" (case-insensitive).
func confirmInsert(r io.Reader) bool {
	scanner := bufio.NewScanner(r)
	if !scanner.Scan() {
		return false
	}
	answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
	return answer == "y" || answer == "yes"
}
