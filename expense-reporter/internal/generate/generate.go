// Package generate builds a complete expense workbook from a taxonomy file and
// an optional entries log, per the workbook generator spec v2
// (.claude/plans/workbook-generator-spec.md). Ported from the convergence-verified
// scratch builder (.claude/scratch/template-builder).
package generate

import (
	"fmt"
	"os"
	"path/filepath"

	"expense-reporter/internal/taxonomy"
	"github.com/xuri/excelize/v2"
)

// headroomRows, perGroupPctRows and dataYear are package state set by Generate()
// from its Options before building (ported from the scratch builder's consts;
// the CLI is single-shot, so mutable package state is acceptable here).
var headroomRows = 0 // §3.2: regenerate-don't-insert → no spare rows by default

// perGroupPctRows toggles the per-group "% sobre despesas/receita" rows (§4.2).
var perGroupPctRows = true

// dataYear is the config year applied to entry dates.
var dataYear = 2026

// Options configures one workbook generation run.
type Options struct {
	TaxonomyPath string // taxonomy JSON file (spec §1.1) — required
	EntriesPath  string // expenses_log.jsonl-format entries; empty = skeleton only
	OutPath      string // output .xlsx path — required
	Year         int    // year applied to entry dates (DD/MM in the log has no year)
	Headroom     int    // spare data rows per block beyond max-entries (spec §3.2; default 0)
}

// Generate builds the workbook described by opts and writes it to opts.OutPath.
func Generate(opts Options) error {
	if opts.TaxonomyPath == "" {
		return fmt.Errorf("taxonomy path is required")
	}
	if opts.OutPath == "" {
		return fmt.Errorf("output path is required")
	}
	dataYear = opts.Year
	headroomRows = opts.Headroom

	expenseSheets, revenueBlocks, err := taxonomy.LoadTaxonomy(opts.TaxonomyPath, opts.EntriesPath)
	if err != nil {
		return err
	}
	return buildWorkbook(expenseSheets, revenueBlocks, opts.OutPath)
}

// buildWorkbook renders the loaded taxonomy+entries into an xlsx file (port of
// the scratch builder's run()).
func buildWorkbook(expenseSheets []taxonomy.ExpenseSheet, revenueBlocks []taxonomy.RevenueBlock, outPath string) error {
	f := excelize.NewFile()
	defer f.Close()

	st, err := newStyles(f)
	if err != nil {
		return fmt.Errorf("styles: %w", err)
	}
	lbl := newPtBRLabels()
	reg := newLayoutRegistry()

	// Build source sheets first so Listas can wire to their total rows.
	if err := buildRevenueSheet(f, st, lbl, revenueBlocks, reg); err != nil {
		return fmt.Errorf("revenue: %w", err)
	}
	for _, sh := range expenseSheets {
		if err := buildExpenseSheet(f, st, lbl, sh, reg); err != nil {
			return fmt.Errorf("expense %s: %w", sh.Name, err)
		}
	}
	if err := buildSummarySheet(f, st, lbl, reg); err != nil {
		return fmt.Errorf("listas: %w", err)
	}

	if err := orderSheets(f, lbl, expenseSheets); err != nil {
		return err
	}

	// Stale-value fix (spec §6): update linked values, then force recalc on load.
	if err := f.UpdateLinkedValue(); err != nil {
		return fmt.Errorf("update linked: %w", err)
	}
	yes := true
	if err := f.SetCalcProps(&excelize.CalcPropsOptions{FullCalcOnLoad: &yes}); err != nil {
		return fmt.Errorf("calc props: %w", err)
	}

	return saveWorkbook(f, outPath)
}

// orderSheets removes the default sheet and orders: Listas, Receitas, then the
// expense sheets in taxonomy order. MoveSheet(source, target) moves source
// before target, so we walk the order backward.
func orderSheets(f *excelize.File, lbl Labels, expenseSheets []taxonomy.ExpenseSheet) error {
	if err := f.DeleteSheet("Sheet1"); err != nil {
		return err
	}
	order := []string{summarySheetName, lbl.RevenueSheet}
	for _, sh := range expenseSheets {
		order = append(order, sh.Name)
	}
	for i := len(order) - 2; i >= 0; i-- {
		if err := f.MoveSheet(order[i], order[i+1]); err != nil {
			return fmt.Errorf("move %s: %w", order[i], err)
		}
	}
	if i, _ := f.GetSheetIndex(summarySheetName); i >= 0 {
		f.SetActiveSheet(i)
	}
	return nil
}

// saveWorkbook resolves the output path, ensures its directory, and writes the file.
func saveWorkbook(f *excelize.File, outPath string) error {
	out, err := filepath.Abs(outPath)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		return err
	}
	if err := f.SaveAs(out); err != nil {
		return fmt.Errorf("save %s: %w", out, err)
	}
	return nil
}
