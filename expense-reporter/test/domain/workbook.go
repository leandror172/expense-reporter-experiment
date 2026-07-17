//go:build acceptance

// Package domain holds expense-reporter-specific test helpers that the
// acceptance-harness module deliberately excludes: the workbook gate and copy,
// and the config-next-to-the-binary writer. These encode this CLI's own
// conventions (an Excel workbook it may write, os.Executable()-relative config
// resolution) and would be dead weight in a generic CLI-test library.
package domain

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"expense-reporter/test/harness"
)

// RequireWorkbook skips the test if no workbook path is configured.
// Call at the top of any test that needs workbook insertion.
func RequireWorkbook(t *testing.T, workbookPath string) {
	t.Helper()
	if workbookPath == "" {
		t.Skip("skipping: EXPENSE_WORKBOOK_PATH not set")
	}
}

// CopyWorkbookToWorkDir copies the workbook to the test's isolated work directory
// and updates ctx.WorkbookPath to point to the copy. This prevents tests that write
// to the workbook from sharing mutable state across the test suite.
func CopyWorkbookToWorkDir(ctx *harness.Context, workbookPath string) error {
	if workbookPath == "" {
		return nil
	}
	dst := filepath.Join(ctx.WorkDir, filepath.Base(workbookPath))
	if err := copyFile(workbookPath, dst); err != nil {
		return err
	}
	ctx.WorkbookPath = dst
	return nil
}

// copyFile copies src to dst, truncating dst if it exists.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}
