//go:build acceptance

// Package expect provides expense-reporter's domain Then assertions: what this
// CLI's artifacts mean (feedback and expense logs, classified CSVs, the review
// HTML, workbook structure). The generic assertions it composes with — exit
// status, output substrings, JSON shape — come from the acceptance-harness
// module's verify package, which owns the verify.* namespace. Scenarios import
// both: verify.CommandSucceeded() alongside expect.ExpenseLogMatches(...).
//
// workbook.go is a stub — placeholder for future workbook content assertions.
package expect
