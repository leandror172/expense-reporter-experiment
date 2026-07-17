//go:build acceptance

package domain

import (
	"os"

	"github.com/leandror172/acceptance-harness/harness"
)

// Keys carried in harness.Context.Env by a scenario's Given, read here to build
// the CLI flags they correspond to.
//
// These were fields on the harness Context before the acceptance-harness module
// was extracted; the module's Context is domain-free, so the values ride in Env
// instead. They are flag defaults rather than scenario state — every read is in
// this package, and each becomes a --data-dir / --workbook argument.
const (
	EnvDataDir      = "DATA_DIR"
	EnvWorkbookPath = "WORKBOOK_PATH"
)

// SetDataDir records the classification data directory for subsequent commands.
func SetDataDir(ctx *harness.Context, dir string) { ctx.Env[EnvDataDir] = dir }

// SetWorkbookPath records the workbook the command should operate on.
func SetWorkbookPath(ctx *harness.Context, path string) { ctx.Env[EnvWorkbookPath] = path }

// DataDir returns the configured classification data directory, or "" if unset.
func DataDir(ctx *harness.Context) string { return ctx.Env[EnvDataDir] }

// WorkbookPath returns the configured workbook path, or "" if unset.
func WorkbookPath(ctx *harness.Context) string { return ctx.Env[EnvWorkbookPath] }

// CommandEnv returns the environment for the command under test: the suite's own
// environment plus ctx.Env. Forwarding ctx.Env honors its documented contract as
// the env-injection seam, so a future Given that sets a variable the binary reads
// works without rediscovering this. The keys above ride along harmlessly — the
// binary takes those two as flags, not from the environment.
func CommandEnv(ctx *harness.Context) []string {
	env := os.Environ()
	for k, v := range ctx.Env {
		env = append(env, k+"="+v)
	}
	return env
}
