package config

import (
	"path/filepath"
	"testing"
)

func TestExpensesLogFilePath_Empty(t *testing.T) {
	c := &Config{}
	if got := c.ExpensesLogFilePath(); got != "" {
		t.Errorf("ExpensesLogFilePath() = %q, want empty string when path is unset", got)
	}
}

func TestExpensesLogFilePath_Absolute(t *testing.T) {
	abs := "/tmp/expenses_log.jsonl"
	c := &Config{ExpensesLogPath: abs}
	if got := c.ExpensesLogFilePath(); got != abs {
		t.Errorf("ExpensesLogFilePath() = %q, want %q", got, abs)
	}
}

func TestExpensesLogFilePath_Relative(t *testing.T) {
	c := &Config{ExpensesLogPath: "expenses_log.jsonl"}
	got := c.ExpensesLogFilePath()
	if !filepath.IsAbs(got) {
		t.Errorf("ExpensesLogFilePath() = %q, want an absolute path for relative input", got)
	}
	if filepath.Base(got) != "expenses_log.jsonl" {
		t.Errorf("ExpensesLogFilePath() base = %q, want expenses_log.jsonl", filepath.Base(got))
	}
}

func TestWorkbookFilePath_Empty(t *testing.T) {
	c := &Config{}
	if got := c.WorkbookFilePath(); got != "" {
		t.Errorf("WorkbookFilePath() = %q, want empty string when path is unset", got)
	}
}

func TestWorkbookFilePath_Absolute(t *testing.T) {
	abs := "/tmp/workbook.xlsx"
	c := &Config{WorkbookPath: abs}
	if got := c.WorkbookFilePath(); got != abs {
		t.Errorf("WorkbookFilePath() = %q, want %q", got, abs)
	}
}

func TestWorkbookFilePath_Relative(t *testing.T) {
	c := &Config{WorkbookPath: "workbook.xlsx"}
	got := c.WorkbookFilePath()
	if !filepath.IsAbs(got) {
		t.Errorf("WorkbookFilePath() = %q, want an absolute path for relative input", got)
	}
	if filepath.Base(got) != "workbook.xlsx" {
		t.Errorf("WorkbookFilePath() base = %q, want workbook.xlsx", filepath.Base(got))
	}
}

func TestClassificationsFilePath_Empty(t *testing.T) {
	c := &Config{}
	if got := c.ClassificationsFilePath(); got != "" {
		t.Errorf("ClassificationsFilePath() = %q, want empty string when path is unset", got)
	}
}

func TestClassificationsFilePath_Absolute(t *testing.T) {
	abs := "/tmp/classifications.jsonl"
	c := &Config{ClassificationsPath: abs}
	if got := c.ClassificationsFilePath(); got != abs {
		t.Errorf("ClassificationsFilePath() = %q, want %q", got, abs)
	}
}

func TestClassificationsFilePath_Relative(t *testing.T) {
	c := &Config{ClassificationsPath: "classifications.jsonl"}
	got := c.ClassificationsFilePath()
	// Must be absolute after resolution
	if !filepath.IsAbs(got) {
		t.Errorf("ClassificationsFilePath() = %q, want an absolute path for relative input", got)
	}
	// Must end with the filename
	if filepath.Base(got) != "classifications.jsonl" {
		t.Errorf("ClassificationsFilePath() base = %q, want classifications.jsonl", filepath.Base(got))
	}
}
