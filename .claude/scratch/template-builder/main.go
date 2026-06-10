package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/xuri/excelize/v2"
)

const outRelPath = "../../workbook-template/template.xlsx"

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run() error {
	f := excelize.NewFile()
	defer f.Close()

	st, err := newStyles(f)
	if err != nil {
		return fmt.Errorf("styles: %w", err)
	}

	expenseSheets, receitasBlocks := buildTaxonomy()
	reg := newLayoutRegistry()

	// Build source sheets first so Listas can wire to their total rows.
	if err := buildReceitas(f, st, receitasBlocks, reg); err != nil {
		return fmt.Errorf("receitas: %w", err)
	}
	for _, sh := range expenseSheets {
		if err := buildExpenseSheet(f, st, sh, reg); err != nil {
			return fmt.Errorf("expense %s: %w", sh.Name, err)
		}
	}
	if err := buildListas(f, st, reg); err != nil {
		return fmt.Errorf("listas: %w", err)
	}

	// Remove the default sheet and order: Listas, Receitas, Fixas, Variáveis, Extras, Adicionais.
	if err := f.DeleteSheet("Sheet1"); err != nil {
		return err
	}
	order := []string{listasName, "Receitas", "Fixas", "Variáveis", "Extras", "Adicionais"}
	// Place sheets in order: moving from the last pair backward, put each sheet directly
	// before its successor. MoveSheet(source, target) moves source before target.
	for i := len(order) - 2; i >= 0; i-- {
		if err := f.MoveSheet(order[i], order[i+1]); err != nil {
			return fmt.Errorf("move %s: %w", order[i], err)
		}
	}
	if i, _ := f.GetSheetIndex(listasName); i >= 0 {
		f.SetActiveSheet(i)
	}

	// Stale-value fix (§6): update linked values, then force recalc on load.
	if err := f.UpdateLinkedValue(); err != nil {
		return fmt.Errorf("update linked: %w", err)
	}
	yes := true
	if err := f.SetCalcProps(&excelize.CalcPropsOptions{FullCalcOnLoad: &yes}); err != nil {
		return fmt.Errorf("calc props: %w", err)
	}

	out, err := filepath.Abs(outRelPath)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		return err
	}
	if err := f.SaveAs(out); err != nil {
		return fmt.Errorf("save: %w", err)
	}
	fmt.Println("wrote", out)
	return nil
}
