//go:build acceptance

package domain

import (
	"encoding/json"
	"fmt"

	"github.com/leandror172/acceptance-harness/harness"
)

// Fixture config defaults. The acceptance-harness core applies no domain defaults,
// so these live here — they are the values the local loader applied before the
// engine was extracted, and fixtures omit these keys expecting them.
const (
	defaultThreshold     = 0.85
	defaultTopN          = 3
	defaultAssertionType = "hard"
)

// ExpenseFixtureConfig is a fixture's config.json: the generic half (command,
// extra_args) plus expense-reporter's own fields, which the module's FixtureConfig
// deliberately does not know about.
type ExpenseFixtureConfig struct {
	harness.FixtureConfig

	Model         string  // Ollama model, e.g. "my-classifier-q3"
	AssertionType string  // "hard" (exact) or "soft" (accuracy floor)
	AccuracyFloor float64 // minimum accuracy a soft assertion must hold
	TopN          int     // how many candidates to request
	// Threshold is inert since T-32: --threshold is deprecated and the agreement
	// gate ignores it. Decoded so existing fixture configs still parse; do not
	// build new behavior on it.
	Threshold float64
}

// LoadExpenseFixtureConfig reads a fixture's config.json and applies the defaults
// for keys it omits.
func LoadExpenseFixtureConfig(dir string) (ExpenseFixtureConfig, error) {
	base, err := harness.LoadFixtureConfig(dir)
	if err != nil {
		return ExpenseFixtureConfig{}, err
	}
	cfg := ExpenseFixtureConfig{
		FixtureConfig: base,
		Threshold:     defaultThreshold,
		TopN:          defaultTopN,
		AssertionType: defaultAssertionType,
	}
	for key, target := range map[string]any{
		"model":          &cfg.Model,
		"assertion_type": &cfg.AssertionType,
		"accuracy_floor": &cfg.AccuracyFloor,
		"top_n":          &cfg.TopN,
		"threshold":      &cfg.Threshold,
	} {
		if err := decodeInto(base.Raw, key, target); err != nil {
			return ExpenseFixtureConfig{}, fmt.Errorf("fixture %s: %w", dir, err)
		}
	}
	return cfg, nil
}

// decodeInto unmarshals raw[key] into target, leaving target at its default when
// the key is absent. A present-but-malformed value is an error rather than a
// silent fallback — a typo'd fixture key should fail loudly, not test something
// other than what it says.
func decodeInto(raw map[string]json.RawMessage, key string, target any) error {
	msg, ok := raw[key]
	if !ok {
		return nil
	}
	if err := json.Unmarshal(msg, target); err != nil {
		return fmt.Errorf("decode %q: %w", key, err)
	}
	return nil
}
