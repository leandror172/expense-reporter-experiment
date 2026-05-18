## Goal

A Lovable web companion to your local `expense-reporter-experiment` Go CLI. The CLI keeps doing the heavy lifting (Excel ingest + local Ollama classification). Lovable becomes the **human-in-the-loop surface**: a review queue for low-confidence rows, a taxonomy editor, and a feedback/analytics log. Single user (you).

## Architecture

```text
Excel statement
   │
   ▼
Go CLI (local)  ──►  Ollama (local)  ──►  classify rows
   │                                          │
   │ POST /api/public/ingest (HMAC-signed)    │
   ▼                                          ▼
Lovable Cloud (Postgres) ◄────── verdicts ─── Lovable web UI
   ▲                                          │
   └──────── GET /api/public/taxonomy ────────┘
            GET /api/public/verdicts?since=…
```

- CLI pushes every classified row to Lovable. Anything below confidence threshold lands in the review queue.
- You triage in the browser: approve / re-categorize / split / add rule.
- CLI polls verdicts + taxonomy on next run; corrections train future runs (and feed your fine-tuning loop).

## Scope (Full workbench)

### 1. Review Dashboard (`/`)
- Inbox of rows with `confidence < threshold` or `status = 'needs_review'`.
- Table: date, description, amount, suggested category, confidence bar, model used, source file.
- Row actions: Approve · Change category · Mark as rule (description pattern → category) · Skip.
- Filters: confidence range, date range, statement, category, status. Bulk-approve selection.
- Keyboard shortcuts (j/k navigate, a approve, c change).

### 2. Taxonomy Editor (`/taxonomy`)
- CRUD on categories (name, parent, color, notes, examples).
- CRUD on rules: regex/contains pattern → category, with priority. Test box that previews matches against recent transactions.
- Versioned: each save bumps `taxonomy_version`; CLI fetches latest on run.

### 3. Feedback Log & Analytics (`/insights`)
- JSONL-style browsable log of every verdict (model decision, your override, timestamps).
- Charts: confidence distribution, override rate per category, model accuracy over time, spend per category per month.
- Export: CSV of corrections (the dataset for your Layer-7 fine-tune).

### 4. Statements browser (`/statements`)
- List of ingested files with counts (total / auto-approved / reviewed / pending).
- Drill into a statement → its rows.

### Auth
- Single-user email/password via Lovable Cloud auth. All app routes behind `_authenticated` layout.

## Backend (Lovable Cloud)

Tables:
- `categories(id, name, parent_id, color, notes, created_at)`
- `rules(id, pattern, match_type, category_id, priority, enabled, created_at)`
- `statements(id, filename, source_hash, imported_at, row_count)`
- `transactions(id, statement_id, txn_date, description, amount, currency, raw)`
- `classifications(id, transaction_id, model, suggested_category_id, confidence, status, reviewed_category_id, reviewed_at, reviewer_note)`
- `taxonomy_versions(id, snapshot jsonb, created_at)`

Status enum: `auto_approved | needs_review | approved | corrected | skipped`.

Public API (HMAC-signed via `/api/public/*`):
- `POST /api/public/ingest` — CLI uploads `{statement, rows[], classifications[]}`.
- `GET  /api/public/taxonomy?since=version` — current categories + rules.
- `GET  /api/public/verdicts?since=ts` — corrections since last poll, for CLI feedback log.

All public routes verify `x-signature: HMAC-SHA256(body, EXPENSE_WEBHOOK_SECRET)` with `timingSafeEqual` before doing anything.

Internal app data access uses TanStack Start server functions with `requireSupabaseAuth`.

## What stays local (unchanged in your Go repo)
- Excel parsing, Ollama calls, Modelfile, fine-tune loop. Lovable never touches your data files or your model.
- You add a thin Go HTTP client (~100 lines) that POSTs to `/api/public/ingest` and pulls verdicts/taxonomy.

## Tech notes

- Frontend: TanStack Start (existing template), shadcn UI, TanStack Query, Recharts for `/insights`.
- Sidebar layout (Review · Statements · Taxonomy · Insights · Settings) using shadcn Sidebar.
- Secret `EXPENSE_WEBHOOK_SECRET` stored in Lovable Cloud, surfaced to CLI via your local env.
- Stable URL `project--<id>.lovable.app/api/public/*` for the CLI to call.

## Build order (suggested next steps)

1. Enable Lovable Cloud + auth (single user) + schema migration.
2. `/api/public/ingest` route + HMAC verify + minimal seed taxonomy.
3. Review dashboard with mock data, then wire to real `classifications`.
4. Taxonomy editor + versioning.
5. `/api/public/taxonomy` and `/api/public/verdicts` for CLI polling.
6. Insights charts + corrections CSV export.
7. README snippet with the Go client example for your repo.

## Out of scope
- Running Ollama, parsing Excel, or fine-tuning inside Lovable.
- Multi-user / team features (can be added later via per-user RLS).
- Mobile app.
