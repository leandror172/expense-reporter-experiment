package batch

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// ReportWriter writes batch processing reports to a text file
type ReportWriter struct {
	filePath string
}

// NewReportWriter creates a new report writer
func NewReportWriter(filePath string) *ReportWriter {
	return &ReportWriter{filePath: filePath}
}

// Write writes a detailed batch processing report
func (w *ReportWriter) Write(summary *BatchSummary, sourcePath string) error {
	var report strings.Builder

	// Header
	report.WriteString("Batch Import Report\n")
	report.WriteString("===================\n\n")
	report.WriteString(fmt.Sprintf("Date: %s\n", time.Now().Format("2006-01-02 15:04:05")))
	report.WriteString(fmt.Sprintf("Source: %s\n\n", sourcePath))

	// Summary Statistics
	report.WriteString("Summary Statistics\n")
	report.WriteString("------------------\n")
	report.WriteString(fmt.Sprintf("Total lines processed: %d\n", summary.TotalLines))

	if summary.TotalLines > 0 {
		successPct := float64(summary.SuccessCount) / float64(summary.TotalLines) * 100
		errorPct := float64(summary.ErrorCount) / float64(summary.TotalLines) * 100
		ambiguousPct := float64(summary.AmbiguousCount) / float64(summary.TotalLines) * 100

		report.WriteString(fmt.Sprintf("✓ Successful: %d (%.2f%%)\n", summary.SuccessCount, successPct))
		report.WriteString(fmt.Sprintf("✗ Failed: %d (%.2f%%)\n", summary.ErrorCount, errorPct))
		report.WriteString(fmt.Sprintf("⚠ Ambiguous: %d (%.2f%%)\n", summary.AmbiguousCount, ambiguousPct))
	} else {
		report.WriteString(fmt.Sprintf("✓ Successful: %d\n", summary.SuccessCount))
		report.WriteString(fmt.Sprintf("✗ Failed: %d\n", summary.ErrorCount))
		report.WriteString(fmt.Sprintf("⚠ Ambiguous: %d\n", summary.AmbiguousCount))
	}
	report.WriteString("\n")

	// Successful Insertions
	if summary.SuccessCount > 0 {
		report.WriteString(fmt.Sprintf("Successful Insertions (%d)\n", summary.SuccessCount))
		report.WriteString("-------------------------\n")
		for _, result := range summary.Results {
			if result.Success {
				report.WriteString(fmt.Sprintf("Line %d: %s\n", result.LineNumber, result.ExpenseString))
			}
		}
		report.WriteString("\n")
	}

	// Errors
	if summary.ErrorCount > 0 {
		report.WriteString(fmt.Sprintf("Errors (%d)\n", summary.ErrorCount))
		report.WriteString("---------\n")
		for _, result := range summary.Results {
			if !result.Success && !result.IsAmbiguous {
				report.WriteString(fmt.Sprintf("Line %d: %s\n", result.LineNumber, result.ExpenseString))
				if result.Error != nil {
					report.WriteString(fmt.Sprintf("  Error: %s\n", result.Error.Error()))
				}
			}
		}
		report.WriteString("\n")
	}

	// Ambiguous Entries
	if summary.AmbiguousCount > 0 {
		report.WriteString(fmt.Sprintf("Ambiguous Entries (%d)\n", summary.AmbiguousCount))
		report.WriteString("--------------------\n")
		for _, result := range summary.Results {
			if result.IsAmbiguous {
				report.WriteString(fmt.Sprintf("Line %d: %s\n", result.LineNumber, result.ExpenseString))

				// List sheet options
				if len(result.AmbiguousOpts) > 0 {
					sheetNames := make([]string, len(result.AmbiguousOpts))
					for i, opt := range result.AmbiguousOpts {
						sheetNames[i] = opt.SheetName
					}
					report.WriteString(fmt.Sprintf("  Options: %s\n", strings.Join(sheetNames, ", ")))
				}
			}
		}
		report.WriteString("\n")
	}

	// Write to file
	err := os.WriteFile(w.filePath, []byte(report.String()), 0644)
	if err != nil {
		return fmt.Errorf("failed to write report file: %w", err)
	}

	return nil
}
