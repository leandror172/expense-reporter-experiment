//go:build acceptance

package acceptance_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/leandror172/acceptance-harness/harness"
)

var (
	binaryPath   string
	dataDir      string // absolute path to data/classification
	testWorkbook string // from EXPENSE_WORKBOOK_PATH; empty = workbook tests skip
)

// fixturesDir returns the absolute path to test/fixtures/.
func fixturesDir() string {
	root, err := harness.FindModuleRoot()
	if err != nil {
		panic("fixturesDir: " + err.Error())
	}
	return filepath.Join(root, "test", "fixtures")
}

func TestMain(m *testing.M) {
	// run-acceptance.sh forwards -keep-artifacts / -keep-on-failure to go test, so
	// these must be registered or the script dies on an unknown flag. The module
	// reads the same knobs from HARNESS_KEEP_* and only binds the flags on request,
	// because a library that registers flags at init can collide with its consumer's.
	harness.RegisterFlags()

	moduleRoot, err := harness.FindModuleRoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "TestMain: FindModuleRoot: %v\n", err)
		os.Exit(1)
	}
	dataDir = filepath.Join(moduleRoot, "..", "data", "classification")
	testWorkbook = os.Getenv("EXPENSE_WORKBOOK_PATH")

	binaryPath, err = harness.BuildBinary(moduleRoot, "./cmd/expense-reporter", "expense-reporter")
	if err != nil {
		fmt.Fprintf(os.Stderr, "TestMain: build: %v\n", err)
		os.Exit(1)
	}
	// Every scenario gets ctx.BinaryPath from here, so no Given assigns it.
	harness.UseBinary(binaryPath)

	// Not deferred: os.Exit skips defers, so a deferred cleanup here would never
	// run and every suite run would leak its binary dir under /tmp.
	code := m.Run()
	os.RemoveAll(filepath.Dir(binaryPath))
	os.Exit(code)
}
