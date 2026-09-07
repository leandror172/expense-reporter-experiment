package review

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The expense-type concept was renamed from "sheet" to "type" in T-05 (47a6dff). The
// rename reached the template's JS READERS but not its WRITERS: prefillFor kept returning
// `{sheet: ...}` while every consumer read `pre.type`, and scheduleSaveRows persisted
// `{sheet: ...}` while applySavedRows read `r.type`. Both read undefined, silently, from
// session 33 to session 77 — through two real monthly closes.
//
// Nothing caught it. render_test.go already guards the JSON key contract, but that guards
// the GO side, which was correct all along; the defect lived entirely in the JavaScript,
// where no Go test reaches (the gap T-61 files). This is the cheapest guard that does
// reach it: the state vocabulary is `type`, so a bare `sheet` used as a data key is the
// signature of the rename drifting apart again.
//
// It is deliberately a VOCABULARY check, not a behavioural one — it cannot prove the page
// works, only that this specific fault line has not reopened. A real behavioural guard
// needs a browser toolchain this Go-only repo does not have (see T-61).
func TestTemplateUsesTypeVocabularyNotSheet(t *testing.T) {
	// Only the BARE word `sheet` is forbidden. Longer identifiers that merely contain it —
	// sheetHasCategory, sheetSel, sheetOrder, needsSheet, initialAmbiguousSheet,
	// ambiguousSheetOptions, setSheetOnSelected, withPredictedSheet — are legitimate and
	// must keep passing, which is what the word boundaries buy.
	objectKey := regexp.MustCompile(`\bsheet\s*:`)
	propAccess := regexp.MustCompile(`\.sheet\b`)

	const why = "the state vocabulary is `type`; a `sheet`-spelled data key is the " +
		"signature of a half-applied rename, and the value reads as undefined at runtime " +
		"instead of raising an error"

	t.Run("template is clean", func(t *testing.T) {
		require.Empty(t, objectKey.FindAllString(TemplateHTML, -1), why)
		require.Empty(t, propAccess.FindAllString(TemplateHTML, -1), why)
	})

	// A guard that has never failed is not known to work. These are the exact shapes the
	// two real defects had, plus the identifiers that must NOT trip it.
	t.Run("guard can fail", func(t *testing.T) {
		assert.True(t, objectKey.MatchString(`return { sheet: p.type };`), "the prefillFor defect")
		assert.True(t, objectKey.MatchString(`sheet: pre.type,`), "the STATE-init defect")
		assert.True(t, propAccess.MatchString(`let x = pre.sheet;`), "a sheet-spelled read")

		assert.False(t, objectKey.MatchString(`needsSheet: false,`), "a real key in the template")
		assert.False(t, objectKey.MatchString(`const sheetOrder = TAX;`), "a real identifier")
		assert.False(t, propAccess.MatchString(`s.sheetSel.focus();`), "a real property")
	})
}
