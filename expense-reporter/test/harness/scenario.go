//go:build acceptance

package harness

import "testing"

// Context holds state for a single acceptance test scenario.
type Context struct {
	T           *testing.T
	FixtureDir  string
	WorkDir     string            // t.TempDir() — isolated per test
	WorkbookDir string
	BinaryPath  string
	Artifacts   map[string]string // key → absolute file path
	ExitCode    int
	Stdout      string
	Stderr      string
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
			s.Given(ctx)
		}
		if s.When != nil {
			s.When(ctx)
		}
		for _, step := range s.Then {
			if step != nil {
				step(ctx)
			}
		}
	})
}
