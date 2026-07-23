# acceptance-harness v1.1.0 — engine ownership of scenario plumbing

**Status:** PR-ready proposal · **Target repo:** `github.com/leandror172/acceptance-harness`
(local clone `/mnt/i/workspaces/acceptance-harness`, HEAD `12353c9`, latest tag `v1.0.0`)
**Proposed version:** v1.1.0 — **every change is additive; no consumer edit is required to
upgrade.** · **Authored:** session 64 (2026-07-23), from the expense-reporter Given sweep.

---

## Motivation

A naming/duplication sweep across expense-reporter's 62 acceptance scenarios collapsed
**27 Given helper bodies into 3** canonical ones built from 6 "atomic events". Counting what
those atoms actually do exposed the real finding:

| Atom | What it does | Whose concern |
|---|---|---|
| `binaryBuilt()` | `ctx.BinaryPath = binaryPath` — **21 occurrences** | **engine** |
| `fixtureAvailable(fixDir)` | `ctx.FixtureDir = fixDir` — **18 occurrences** | **engine** |
| `inputBatchStaged(fixDir)` | calls `harness.CopyFixtureToWorkDir` — **13 occurrences** | **engine** |
| `trainingCorpusRecorded()` | sets the data-dir env | consumer domain |
| `taxonomyPublished(fixDir)` | publishes a taxonomy file + config key | consumer domain |
| `feedbackLogsConfigured()` | config keys + artifact registration | consumer domain |

**Half the "domain events" are not domain events.** They exist only to populate fields the
harness *declares* (`Context.BinaryPath`, `Context.FixtureDir`) but leaves every consumer to
wire by hand, in every Given, forever.

That gap manufactures bugs, not just boilerplate. Two found this session:

- `expenseTypedDuringBrowserReview(fixDir)` accepted a `fixDir` parameter and **never used
  it** — the calling convention demanded the parameter, and Go does not flag an unused
  function parameter. The fixture was silently unregistered.
- `cycleCompletedThroughApply(fixDir)` claimed a three-step history while its body set only
  `BinaryPath` + `FixtureDir`. When a Given's whole body is engine plumbing, there is nothing
  left for its name to be accountable to.

Separately, making Givens composable required the consumer's config writer to *merge* keys
instead of overwriting the file. Implementing that needed a package-level
`map[*harness.Context]map[string]any` guarded by a mutex — **only** because `Context` offers
no scenario-scoped state and `Run` offers no "Givens are finished" moment. Every consumer
with a config file hits this same wall.

---

## Change 1 — `Scenario.Fixture`

The fixture directory is scenario metadata, not a Given side effect.

```diff
 type Scenario struct {
 	Name  string
+	// Fixture is the scenario's fixture directory. When set, Run assigns it to
+	// ctx.FixtureDir before Given runs, so Givens need not take or wire a fixture
+	// path. Optional: leave empty and a Given may set ctx.FixtureDir itself.
+	Fixture string
 	Given func(*Context)
 	When  func(*Context)
 	Then  []func(*Context)
 }
```

```diff
 	ctx := &Context{
 		T:         t,
 		Artifacts: make(map[string]string),
 		Env:       make(map[string]string),
 		WorkDir:   workDir,
 	}
+	ctx.FixtureDir = s.Fixture
 	if s.Given != nil {
```

**Compatibility:** zero value `""` reproduces today's behaviour exactly — a Given that assigns
`ctx.FixtureDir` still wins, because it runs after. **Kills:** 18 assignments and the
unused-`fixDir`-parameter bug class in this consumer alone.

## Change 2 — engine-owned binary path

Every Given in every consumer starts with `ctx.BinaryPath = <the binary TestMain built>`. The
binary is built once per suite, so the default belongs at package scope, with a per-scenario
override for suites testing more than one binary.

```go
// UseBinary sets the binary Run assigns to ctx.BinaryPath for every scenario.
// Call once from TestMain after BuildBinary. A Scenario.Binary overrides it.
func UseBinary(path string) { defaultBinary = path }

var defaultBinary string
```

```diff
 type Scenario struct {
 	Name    string
 	Fixture string
+	// Binary overrides the package default from UseBinary for this scenario.
+	Binary  string
 	Given   func(*Context)
```

```diff
+	ctx.BinaryPath = defaultBinary
+	if s.Binary != "" {
+		ctx.BinaryPath = s.Binary
+	}
 	ctx.FixtureDir = s.Fixture
```

**Compatibility:** if `UseBinary` is never called, `ctx.BinaryPath` starts `""` and consumer
Givens assign it exactly as today. **Kills:** 21 assignments here; structurally, one line from
every Given ever written against this engine.

## Change 3 — scenario-scoped state + a pre-When hook

The enabling pair for composable Givens. Neither is expressible by a consumer today without
a global map.

```diff
 type Context struct {
 	T          *testing.T
 	FixtureDir string
 	WorkDir    string
 	BinaryPath string
 	Env        map[string]string
 	Artifacts  map[string]string
+	// State is scenario-scoped scratch space for consumer setup code that must
+	// accumulate across several Given events (e.g. config keys contributed by
+	// independent events, flushed once via BeforeWhen). Never read by the engine.
+	State      map[string]any
 	ExitCode   int
 	Stdout     string
 	Stderr     string
+
+	beforeWhen []func()
 }
+
+// BeforeWhen registers fn to run after every Given event and before When.
+// Use it to flush state accumulated across composed Givens exactly once —
+// writing a config file per contributing event would make the last writer win
+// and silently drop the others' keys.
+func (c *Context) BeforeWhen(fn func()) { c.beforeWhen = append(c.beforeWhen, fn) }
```

```diff
 	if s.Given != nil {
 		t.Log("→ Given: setting up scenario")
 		s.Given(ctx)
 	}
+	for _, fn := range ctx.beforeWhen {
+		fn()
+	}
 	if s.When != nil {
```

Plus `State: make(map[string]any)` in the `Context` literal.

**What it replaces, consumer-side** — expense-reporter's current workaround:

```go
// before: package-level map + mutex, because Context had nowhere to accumulate
var pendingConfigMu sync.Mutex
var pendingConfig = map[*harness.Context]map[string]interface{}{}

// after: no globals, no mutex, one write instead of N
func SetupBinaryConfig(ctx *harness.Context, cfg map[string]any) error {
	merged, ok := ctx.State["config"].(map[string]any)
	if !ok {
		merged = map[string]any{}
		ctx.State["config"] = merged
		ctx.BeforeWhen(func() { writeConfigFile(ctx, merged) })
	}
	maps.Copy(merged, cfg)
	return nil
}
```

**Compatibility:** both are new; nil `beforeWhen` runs zero hooks. Consumers constructing a
`Context` directly (unit tests of their own verifiers) keep working — `State` nil-maps only
break on write, and only for code that opts into using it.

## Change 4 — `harness.Events(...)` compose helper

```go
// Events folds a sequence of setup events into the single function Scenario.Given
// takes, so a Given can be written as the sequence of events it actually is.
// Nil events are skipped.
func Events(events ...func(*Context)) func(*Context) {
	return func(ctx *Context) {
		for _, event := range events {
			if event != nil {
				event(ctx)
			}
		}
	}
}
```

Named `Events`, not `Given`, to avoid `Given: harness.Given(...)` stutter at the call site.

**Note:** Changes 1–2 remove roughly half the reason to compose in the first place. With them,
expense-reporter's `taxonomyAuthoredWithTrainingData` drops from five events to three.

---

## Explicit non-goal: `Given []func(*Context)`

Considered and **rejected** — full analysis in the consumer repo at
`.claude/given-slice-upstream-analysis.md`. Summary of the evidence:

- All **62** scenarios pass exactly **one** named Given at the call site; none would be
  written as a list. The convention is one domain-named event per scenario, where the name is
  the documentation.
- Decomposing at the call site replaces a domain sentence with a setup checklist — plumbing
  leaking upward.
- The suite's best-designed Given family (`expenseLogSeededWith(fixDir, rows)` + 4 named
  wrappers) achieves composition by **parameterization**, needing neither a slice nor a
  compose helper.
- It would be source-breaking (v2.0.0) across two consumers, adding `slices.Concat` noise to
  62 call sites to enable a pattern zero of them want.

The one honest argument for it — it makes the monolithic-Given anti-pattern harder to write —
is weak, since nothing stops a consumer putting one fat function in the slice.

---

## Tests to add (`harness/scenario_test.go`)

1. `Scenario.Fixture` → `ctx.FixtureDir` before Given observes it.
2. Empty `Fixture` leaves `ctx.FixtureDir` assignable by the Given (compat).
3. `UseBinary` default reaches `ctx.BinaryPath`; `Scenario.Binary` overrides it; a Given
   assigning `BinaryPath` still wins (compat).
4. `BeforeWhen` hooks run **after all** Given events and **before** When, in registration
   order; zero hooks is a no-op.
5. `State` is non-nil on entry and isolated between scenarios.
6. `Events` applies in order and skips nils; `Events()` with no args is a no-op.

## Docs to update

- `docs/PATTERNS.md` — new "Composing a Given from events" section; the Given naming
  discipline that came out of the consumer sweep (event-style past-tense names; the call site
  reads as an English phrase with the argument slotted into the grammar; **duplicate bodies
  are the tell** that a name describes assembly rather than a fact); and the merge trap:
  *if your Givens compose, your domain layer's config writer must accumulate and flush once,
  never write per event.*
- `docs/ADOPTION.md` — `Fixture` / `Binary` / `UseBinary` in the getting-started scenario.
- `README.md` — Scenario struct example.
- `docs/consumer/expense-reporter.md` — note the v1.1.0 adoption.

## Consumer adoption (expense-reporter, after the bump)

1. `harness.UseBinary(binaryPath)` in `TestMain`; delete `binaryBuilt()`.
2. Add `Fixture: fixDir` to scenarios; delete `fixtureAvailable()`; drop now-unused `fixDir`
   parameters from Givens that only forwarded them.
3. Rewrite `domain.SetupBinaryConfig` onto `ctx.State` + `ctx.BeforeWhen`; delete the
   package-level map and mutex.
4. Replace local `given(...)` with `harness.Events(...)`.
5. Full suite must stay 62/62 green; the deterministic group (`./run-acceptance.sh`) is ~8s
   for fast iteration.

Adoption is incremental — every step is optional and old-style Givens keep working, so the
bump itself can land first and the cleanup follow scenario by scenario.

## Risks

- `UseBinary` is package-level mutable state. Set once in `TestMain` before any scenario; it
  is not safe to mutate from a running parallel test. Document, do not guard — a mutex would
  imply a supported use that is not intended.
- `State map[string]any` is untyped by design (the engine carries no consumer types). Consumers
  should key it with a package-private constant to avoid collisions between their own layers.
- `BeforeWhen` runs even when `When` is nil, so hooks stay symmetric with Given. Worth a test.
