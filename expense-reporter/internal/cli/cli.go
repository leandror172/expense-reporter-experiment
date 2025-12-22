package cli

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

const Version = "1.0.0"

// ParseArgs parses command line arguments
// Returns (command, input, error)
func ParseArgs(args []string) (string, string, error) {
	if len(args) == 0 {
		return "", "", fmt.Errorf("no arguments provided")
	}

	command := args[0]

	switch command {
	case "add":
		if len(args) < 2 {
			return "", "", fmt.Errorf("add command requires expense string argument")
		}
		// Join all remaining args as the expense string (in case user didn't quote it)
		expenseString := strings.Join(args[1:], " ")
		return command, expenseString, nil

	case "help":
		return command, "", nil

	case "version":
		return command, "", nil

	default:
		return "", "", fmt.Errorf("unknown command: %s", command)
	}
}

// PromptForSelection displays options and prompts user to select one
// Returns the selected index (0-based) or error
func PromptForSelection(options []string, input io.Reader, output io.Writer) (int, error) {
	if len(options) == 0 {
		return -1, fmt.Errorf("no options provided")
	}

	// Display options
	fmt.Fprintln(output, "\nMultiple locations found for this subcategory:")
	for i, opt := range options {
		fmt.Fprintf(output, "  %d. %s\n", i+1, opt)
	}
	fmt.Fprintf(output, "\nSelect an option (1-%d): ", len(options))

	// Read user input
	scanner := bufio.NewScanner(input)
	if !scanner.Scan() {
		return -1, fmt.Errorf("failed to read input")
	}

	choiceStr := strings.TrimSpace(scanner.Text())
	choice, err := strconv.Atoi(choiceStr)
	if err != nil {
		return -1, fmt.Errorf("invalid input: expected a number")
	}

	// Validate choice
	if choice < 1 || choice > len(options) {
		return -1, fmt.Errorf("invalid choice: must be between 1 and %d", len(options))
	}

	// Return 0-based index
	return choice - 1, nil
}

// PrintHelp prints usage information
func PrintHelp(output io.Writer) {
	help := `expense-reporter - Automate expense reporting to Excel budget

Usage:
  expense-reporter add "<item_description>;<DD/MM>;<value>;<sub_category>"
  expense-reporter help
  expense-reporter version

Commands:
  add     Add a new expense to the budget spreadsheet
  help    Show this help message
  version Show version information

Expense Format:
  <item_description>;<DD/MM>;<value>;<sub_category>

  - item_description: Description of the expense
  - DD/MM: Day and month (year is always 2025)
  - value: Amount in format ##,## (e.g., 35,50 for R$ 35.50)
  - sub_category: Subcategory name (e.g., "Uber/Taxi", "Supermercado")

Examples:
  expense-reporter add "Uber Centro;15/04;35,50;Uber/Taxi"
  expense-reporter add "Compras Carrefour;03/01;150,00;Supermercado"
  expense-reporter add "Consulta veterinária;10/03;180,00;Orion - Consultas"

Environment Variables:
  EXPENSE_WORKBOOK_PATH   Path to the Excel workbook (optional)
                          Default: ../Planilha_BMeFBovespa_Leandro_OrcamentoPessoal-2025.xlsx

Notes:
  - If a subcategory appears in multiple sheets (e.g., "Dentista" in both
    Variáveis and Extras), you will be prompted to select which one.
  - Smart matching: "Orion - Consultas" will find "Orion" if exact match not found.
  - The tool will find the next empty row in the appropriate month column.
`
	fmt.Fprint(output, help)
}

// PrintVersion prints version information
func PrintVersion(output io.Writer) {
	fmt.Fprintf(output, "expense-reporter version %s\n", Version)
}
