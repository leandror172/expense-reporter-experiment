//go:build acceptance

package harness

import "testing"

// Context holds state for a single acceptance test scenario.
type Context struct {
	T            *testing.T
	FixtureDir   string
	WorkDir      string            // t.TempDir() — isolated per test
	WorkbookDir  string
	BinaryPath   string
	DataDir      string            // absolute path to classification data directory
	WorkbookPath string            // absolute path to Excel workbook; empty = skip workbook tests
	Artifacts    map[string]string // key → absolute file path
	ExitCode     int
	Stdout       string
	Stderr       string
}

// RequireWorkbook skips the test if no workbook path is configured.
// Call at the top of any test that needs workbook insertion.
func RequireWorkbook(t *testing.T, workbookPath string) {
	t.Helper()
	if workbookPath == "" {
		t.Skip("skipping: EXPENSE_WORKBOOK_PATH not set")
	}
}

// Scenario defines a Given/When/Then acceptance test.
type Scenario struct {
	Name  string
	Given func(*Context)
	When  func(*Context)
	Then  []func(*Context)
}

// Run executes the scenario inside t.Run, calling Given → When → Then[] in order.
func Run(t *testing.T, s Scenario) {
	t.Helper()
	t.Run(s.Name, func(t *testing.T) {
		ctx := &Context{
			T:         t,
			Artifacts: make(map[string]string),
			WorkDir:   t.TempDir(),
		}
		if s.Given != nil {
			t.Log("→ Given: setting up scenario")
			s.Given(ctx)
		}
		if s.When != nil {
			t.Log("→ When: executing command (may take a while — waiting for Ollama)")
			s.When(ctx)
		}
		t.Logf("→ Then: checking %d assertion(s)", len(s.Then))
		for _, step := range s.Then {
			if step != nil {
				step(ctx)
			}
		}
	})
}
