package cmd

import (
	"bufio"
	"errors"
	"expense-reporter/internal/appender"
	"expense-reporter/internal/classifier"
	"expense-reporter/internal/config"
	"expense-reporter/internal/feedback"
	"expense-reporter/internal/parse"
	taxonomy "expense-reporter/internal/taxonomy"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var addDryRun bool
var addDataDir string
var addType string
var addYear int

var addPredictedSubcategory string
var addPredictedCategory string
var addClassificationID string
var addConfidence float64
var addModel string

var addCmd = &cobra.Command{
	Use:   "add \"<item>;<DD/MM[/YYYY]>;<##,##[/N]>;<subcat>\"",
	Short: "Add a single expense",
	Long: `Add a single expense to the expense log.

The expense format is: <item_description>;<DD/MM or DD/MM/YYYY>;<value>;<sub_category>

Installment notation: append /N to the value to expand into N monthly log entries.

Examples:
  expense-reporter add "Uber Centro;15/04/2026;35,50;Uber/Taxi"
  expense-reporter add "Compras Carrefour;03/01/2026;150,00;Supermercado"
  expense-reporter add "Curso online;15/11/2026;90,00/3;Amazon"

Notes:
  - Date accepts DD/MM (defaults to current year) or DD/MM/YYYY
  - Value uses Brazilian format (##,## with comma as decimal)
  - Installments expand into N dated log entries — cross-year dates use the real next-year date
  - Subcategory must exist in the taxonomy`,
	Args: cobra.ExactArgs(1),
	RunE: runAdd,
}

func init() {
	addCmd.Flags().BoolVar(&addDryRun, "dry-run", false, "Validate and parse without inserting into log")
	addCmd.Flags().StringVar(&addDataDir, "data-dir", "data/classification", "(deprecated, no longer used: add resolves via config/taxonomy.json since T-13)")
	addCmd.Flags().StringVar(&addType, "type", "", "Expense type (Fixas/Variáveis/Extras/Adicionais) — required only for subcategories that exist under more than one type")
	addCmd.Flags().IntVar(&addYear, "year", 0, "Fallback year for bare DD/MM dates (outranks config date_year; an explicit year in the date always wins)")
	addCmd.Flags().StringVar(&addPredictedSubcategory, "predicted-subcategory", "", "Model's top prediction for subcategory")
	addCmd.Flags().StringVar(&addPredictedCategory, "predicted-category", "", "Model's predicted category")
	addCmd.Flags().StringVar(&addClassificationID, "classification-id", "", "ID from the prior classify call (for cross-reference)")
	addCmd.Flags().Float64Var(&addConfidence, "confidence", 0.0, "Model's confidence score")
	addCmd.Flags().StringVar(&addModel, "model", "", "Model name used for classification")
	rootCmd.AddCommand(addCmd)
}

func runAdd(cmd *cobra.Command, args []string) error {
	appCfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// T-41: the parse boundary owns all input normalization; the config is
	// loaded first because date_year feeds the year-precedence ladder.
	pe, subcategory, err := parse.ExpenseString(args[0], parse.Options{Year: addYear, ConfigYear: appCfg.DateYear})
	if err != nil {
		return fmt.Errorf("invalid expense format: expected \"item;DD/MM[/YYYY];value[/N];subcategory\": %w", err)
	}

	sheets, err := loadTaxonomyTree(appCfg)
	if err != nil {
		return err
	}

	// T-13: resolve the full (type, category) path from taxonomy.json — the single
	// source of truth — instead of deriving category from the feature dictionary and
	// type from a separate lookup that could disagree.
	typ, category, err := resolveFullPath(sheets, subcategory, addType, stdinIsInteractive(), os.Stdin)
	if err != nil {
		return err
	}

	if addDryRun {
		return runAddDryRun(cmd, pe.Item, pe.DateString(), pe.Value, typ, subcategory, category)
	}

	if logPath := appCfg.ExpensesLogFilePath(); logPath != "" {
		if err := appender.ExpandAndAppend(logPath, pe.Item, pe.Date, pe.Value, pe.Installments, typ, category, subcategory); err != nil {
			fmt.Fprintf(os.Stderr, "⚠  expense log: %v\n", err)
		}
	}

	if addPredictedSubcategory != "" {
		logPredictedFeedback(appCfg, pe.Item, pe.DateString(), pe.Value, subcategory, category,
			addPredictedSubcategory, addPredictedCategory, addClassificationID,
			addConfidence, addModel)
	} else {
		logManualFeedback(appCfg, pe.Item, pe.DateString(), pe.Value, subcategory, category)
	}

	fmt.Println("✓ Expense added successfully!")
	return nil
}

// AddOutput represents the structured output of an add --dry-run command.
// Type is the expense type resolved from the taxonomy full path (T-13); omitted
// when empty so the field is additive for callers that don't consume it.
type AddOutput struct {
	Item        string  `json:"item"`
	Value       float64 `json:"value"`
	Date        string  `json:"date"`
	Type        string  `json:"type,omitempty"`
	Subcategory string  `json:"subcategory"`
	Category    string  `json:"category"`
	Action      string  `json:"action"`
}

func runAddDryRun(cmd *cobra.Command, item, date string, value float64, typ, subcategory, category string) error {
	jsonMode, _ := cmd.Flags().GetBool("json")

	if jsonMode {
		return printJSON(AddOutput{
			Item:        item,
			Value:       value,
			Date:        date,
			Type:        typ,
			Subcategory: subcategory,
			Category:    category,
			Action:      "would_insert",
		})
	}

	fmt.Printf("Dry run — would insert:\n")
	fmt.Printf("  Item:        %s\n", item)
	fmt.Printf("  Date:        %s\n", date)
	fmt.Printf("  Value:       %.2f\n", value)
	if typ != "" {
		fmt.Printf("  Type:        %s\n", typ)
	}
	fmt.Printf("  Subcategory: %s\n", subcategory)
	if category != "" {
		fmt.Printf("  Category:    %s\n", category)
	}
	return nil
}

// logParsedManualFeedback appends feedback log entry using pre-parsed values.
// Non-fatal: any failure silently skips logging.
func logParsedManualFeedback(item, date string, value float64, subcategory, category string) {
	appCfg, err := config.Load()
	if err != nil {
		return
	}
	logManualFeedback(appCfg, item, date, value, subcategory, category)
}

// resolveFullPath resolves a bare subcategory to its (type, category) via the
// taxonomy. Unambiguous leaves (99/104) resolve from the name alone. The handful of
// leaves that repeat across types are resolved, in order, by: the --type flag; an
// interactive prompt when stdin is a terminal; otherwise a hard error — never a
// silent guess, so an unattended run that omits --type fails loudly rather than
// routing to the wrong sheet.
// resolveFullPath is split from its I/O (interactive + in) so the resolution logic
// is deterministically testable. interactive reports whether prompting is allowed;
// in supplies the prompt's answer when it is.
func resolveFullPath(sheets []taxonomy.ExpenseType, subcategory, typeFlag string, interactive bool, in io.Reader) (typ, category string, err error) {
	typ, category, err = taxonomy.ResolveLeaf(sheets, subcategory, typeFlag)
	if err == nil {
		return typ, category, nil
	}
	if errors.Is(err, taxonomy.ErrLeafNotFound) {
		return "", "", fmt.Errorf("subcategory %q is not in the taxonomy", subcategory)
	}

	// ErrLeafAmbiguous: the leaf name exists under more than one type.
	candidates := taxonomy.TypesForLeaf(sheets, subcategory)
	if typeFlag != "" {
		return "", "", fmt.Errorf("subcategory %q is not under type %q (valid types: %s)",
			subcategory, typeFlag, strings.Join(candidates, ", "))
	}
	if !interactive {
		return "", "", fmt.Errorf("subcategory %q exists under multiple types (%s); pass --type to disambiguate",
			subcategory, strings.Join(candidates, ", "))
	}
	chosen, err := promptForType(subcategory, candidates, in)
	if err != nil {
		return "", "", err
	}
	return taxonomy.ResolveLeaf(sheets, subcategory, chosen)
}

// stdinIsInteractive reports whether stdin is a terminal (character device), used to
// decide whether an ambiguous leaf may be resolved by prompting the user.
func stdinIsInteractive() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

// promptForType asks the user to pick one of the candidate types for an ambiguous
// subcategory and returns the chosen type name, reading the answer from in.
func promptForType(subcategory string, candidates []string, in io.Reader) (string, error) {
	fmt.Printf("Subcategory %q exists under multiple types: %s\n", subcategory, strings.Join(candidates, ", "))
	fmt.Print("Enter the type: ")
	scanner := bufio.NewScanner(in)
	if !scanner.Scan() {
		return "", fmt.Errorf("no type provided for ambiguous subcategory %q", subcategory)
	}
	answer := strings.TrimSpace(scanner.Text())
	if answer == "" {
		return "", fmt.Errorf("no type provided for ambiguous subcategory %q", subcategory)
	}
	return answer, nil
}

// logManualFeedback appends a manual entry to classifications.jsonl.
// Non-fatal: warns on stderr if writing fails.
func logManualFeedback(appCfg *config.Config, item, date string, value float64, subcategory, category string) {
	path := appCfg.ClassificationsFilePath()
	if path == "" {
		return
	}
	entry := feedback.NewManualEntry(item, date, value, subcategory, category)
	if err := feedback.Append(path, entry); err != nil {
		fmt.Fprintf(os.Stderr, "⚠  feedback log: %v\n", err)
	}
}

// logPredictedFeedback writes a confirmed or corrected feedback entry to classifications.jsonl,
// depending on whether the user's chosen subcategory matches the model's prediction.
// Non-fatal: warns on stderr if the classification-id cross-reference misses or if the write fails.
func logPredictedFeedback(appCfg *config.Config, item, date string, value float64,
	chosenSubcategory, chosenCategory, predictedSubcategory, predictedCategory, classificationID string,
	confidence float64, model string) {

	path := appCfg.ClassificationsFilePath()

	if classificationID != "" && path != "" {
		_, found, err := feedback.FindLatestEntry(path, classificationID)
		if !found || err != nil {
			fmt.Fprintf(os.Stderr, "⚠  feedback log: classification-id %q not found, continuing without cross-reference\n", classificationID)
		}
	}

	if path == "" {
		return
	}

	predicted := classifier.Result{
		Subcategory: predictedSubcategory,
		Category:    predictedCategory,
		Confidence:  confidence,
	}

	var entry feedback.Entry
	if chosenSubcategory == predictedSubcategory {
		entry = feedback.NewConfirmedEntry(item, date, value, predicted, model)
	} else {
		entry = feedback.NewCorrectedEntry(item, date, value, predicted, model, chosenSubcategory, chosenCategory)
	}

	if err := feedback.Append(path, entry); err != nil {
		fmt.Fprintf(os.Stderr, "⚠  feedback log: %v\n", err)
	}
}
