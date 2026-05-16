# Review UI — Design Brief for claude.ai/design

> **Audience:** the UI builder (claude.ai/design). You do **not** need to know Go,
> servers, or APIs. You are building one HTML file. All data is embedded before the
> file is ever opened.

## What you're building

A **single self-contained HTML file**: an expense-classification *review tool*.

A person opens it in a browser. They see a queue of expense rows that an automatic
classifier guessed at. For each row they confirm the guess or correct it, using a
**3-level cascading picker** (Sheet → Category → Subcategory). When finished they
click **Export** and the browser downloads a JSON file of their verdicts.

## Hard constraints (non-negotiable)

- **ONE `.html` file.** All CSS and JS inline. No external or CDN resources — the
  file must work fully offline, opened via `file://`, with no internet connection.
- **No build step.** No npm, no JSX, no framework that needs compilation. Vanilla
  JS. If you need a helper, paste it inline into a `<script>` tag.
- **No backend.** No `fetch`, no network calls. The only I/O is: read embedded data
  on load, and trigger a file download on export.

## Data injection contract

The page must contain exactly this element:

```html
<script id="review-data" type="application/json">__REVIEW_DATA__</script>
```

At generation time an external tool replaces the literal string `__REVIEW_DATA__`
with a JSON object. Your JS reads it once on load:

```js
const data = JSON.parse(document.getElementById('review-data').textContent);
```

**For development:** ship the file with the real contents of `review-data.sample.json`
pasted in place of `__REVIEW_DATA__`, so the page works when you open it. Final
integration just swaps that one block back out.

### Shape of the injected object — `ReviewData`

```ts
interface ReviewData {
  source: string;        // original CSV filename, e.g. "classified.csv"
  generatedAt: string;   // ISO 8601 timestamp
  queue: QueueEntry[];
  taxonomy: Taxonomy;
}

interface QueueEntry {
  id: string;            // stable opaque hash — pass through untouched, never edit
  item: string;          // expense description, e.g. "Delivery brod's"
  date: string;          // "DD/MM"
  value: number;         // 53.98
  confidence: number;    // 0..1 — classifier's self-reported confidence
  autoInserted: boolean; // true = classifier was confident; false = needs review
  predicted: {
    sheet?: string;      // OPTIONAL — usually absent today; may appear in future data
    category: string;    // e.g. "Diversos"
    subcategory: string; // e.g. "Delivery"
  };
}

interface Taxonomy {
  sheets: Sheet[];
}
interface Sheet {
  name: string;          // one of: "Fixas" | "Variáveis" | "Extras" | "Adicionais"
  categories: Category[];
}
interface Category {
  name: string;          // e.g. "Saúde"
  subcategories: string[]; // e.g. ["Consultas", "Farmácia", "Dentista"]
}
```

## The picker — core UX

Each row has a 3-level cascading selector: **Sheet → Category → Subcategory**.

- Choosing a **Sheet** populates the **Category** dropdown with that sheet's categories.
- Choosing a **Category** populates the **Subcategory** dropdown.
- Changing a parent dropdown must clear/refresh its children — never show stale child
  options from a previously-selected parent.
- A row is fully classified only when all three levels are set.

### Pre-fill rule (get this exact — it is the heart of the tool)

On load, for each queue entry, use `predicted` to pre-select dropdowns:

1. If `predicted.sheet` is present **and** valid in the taxonomy → pre-select all three.
2. Otherwise, search the taxonomy for every **sheet** that contains a category named
   `predicted.category` which in turn contains `predicted.subcategory`:
   - **exactly 1 sheet matches** → pre-select all three (the sheet is inferred).
   - **more than 1 sheet matches** → pre-select Category + Subcategory, leave **Sheet
     empty and visually highlighted** as "needs a choice". This case is real: the same
     category (e.g. "Saúde") exists under multiple sheets.
   - **0 matches** → leave all three empty and mark the row as needing full attention.

The sample data deliberately includes all three cases — verify each renders correctly.

## Verdict actions per row

The action is **derived from the final state**, not a separate mode:

- final 3-level path **equals** the predicted path → `confirmed`
- final path **differs** from the prediction → `corrected`
- user explicitly skipped the row → `skipped`

You may still provide an explicit **Skip** button and a **Confirm** (accept-as-is)
button for speed, but the exported `action` is computed from the above rule.

## Layout

- **Header:** source filename + live counts — total / reviewed / skipped / remaining.
- **Filters:** by `autoInserted` (default: show needs-review rows, i.e. `false`, only);
  confidence range; free-text search on `item`.
- **Row list** (table or cards): date, item, value, confidence (small bar), the three
  dropdowns, and the current derived action state.
- **Value formatting:** Brazilian — `R$ 1.234,56` (dot for thousands, comma for
  decimal). The `value` field arrives as a plain JS number (`1234.56`).

### Out of scope — do NOT build

- Adding new categories, subcategories, or sheets; editing the taxonomy.
- Charts, analytics, dashboards.
- Login / auth.
- If a row genuinely needs a category that does not exist in the taxonomy, that is a
  known accepted gap — **do not** invent an "Add new…" option. Leave the row
  unclassifiable or skippable.

## Keyboard shortcuts

| Key | Action |
|-----|--------|
| `j` / `k` | move row selection down / up |
| `1` `2` `3` `4` | set selected row's Sheet to Fixas / Variáveis / Extras / Adicionais |
| `a` | confirm selected row (accept the prediction) |
| `s` | skip selected row |
| `e` | focus the Category dropdown of the selected row |
| `Enter` | mark selected row reviewed and advance to the next |

## Export

A prominent **"Export reviewed file"** action. It builds a JSON object and triggers a
browser download named `reviewed.json`.

```ts
interface ReviewedFile {
  reviewedAt: string;   // ISO 8601
  source: string;       // copied verbatim from ReviewData.source
  entries: ReviewedEntry[];
}
interface ReviewedEntry {
  id: string;           // copied through from QueueEntry, unchanged
  item: string;
  date: string;
  value: number;
  confidence: number;
  predicted: {          // the ORIGINAL prediction — copied through verbatim
    sheet?: string;
    category: string;
    subcategory: string;
  };
  action: "confirmed" | "corrected" | "skipped";
  reviewed: {           // the user's final 3-level choice; null when action is "skipped"
    sheet: string;
    category: string;
    subcategory: string;
  } | null;
}
```

Rules:
- Export **every** queue entry exactly once (skipped rows included, with `reviewed: null`).
- `predicted` must be carried through **verbatim** — a downstream tool compares it
  against `reviewed` to record whether the classifier was right.

## Acceptance criteria

- Opens via `file://` offline with no console errors.
- Parses the embedded sample and renders every queue row.
- Pre-fill rule behaves correctly for all three cases (sample data exercises each).
- Cascading dropdowns never display stale child options after a parent changes.
- Export produces a valid `ReviewedFile` with exactly one entry per queue row.
- All keyboard shortcuts work.

## Reference fixtures (in `review-ui-fixtures/`)

- `review-data.sample.json` — paste in place of `__REVIEW_DATA__` for development.
  Its `taxonomy` is a **representative sample**; at runtime the real tool injects the
  full taxonomy read from the user's workbook. Build against the shape, not the contents.
- `reviewed.sample.json` — the exact shape your Export must produce.

## How this fits the larger system (context only — nothing to build here)

A Go CLI (`expense-reporter`) classifies a CSV of expenses. A planned `review`
subcommand reads that classified CSV, builds the taxonomy tree from the Excel
workbook, bakes both into a copy of *your* HTML template, and writes a ready-to-open
`review.html`. The user reviews, exports `reviewed.json`, and a later CLI step applies
those verdicts to the workbook and the feedback log. Your HTML file is purely the
review surface in the middle — data in via the embedded block, verdicts out via the
download.
