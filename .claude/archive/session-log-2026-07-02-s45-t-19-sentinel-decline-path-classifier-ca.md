## 2026-07-02 - Session 45: T-19 sentinel decline path — classifier can answer "none of these"

### Context

Started from PR #38 review + T-19 discussion; user merged PR #38 early in the session, then the session became T-19 end-to-end: probe → design → TDD implementation → live re-probe → PR #40.

### What Was Done

- Merged PR #38 (WS-B slice 4, apply → log-append) — WS-B is now complete.
- Live-probed the T-19 risk: 8 out-of-domain items through `classify` on the real 112-path taxonomy. 2/8 wrong picks at exactly 0.85 (would auto-insert); `Diversos` exists in the enum but the model never picked it top-1 — it dumped into the vaguest-*named* leaf instead.
- Implemented the sentinel decline path (TDD, codegen via my-go-qcoder): `SentinelPath = "NENHUMA DAS OPÇÕES"` appended to the enum + prompt line; `splitResults` maps it to the `Diversos` leaf at FIXED 0.30 confidence; dropped when no Diversos leaf. 4 unit tests in `sentinel_test.go`; full suite green.
- Live re-probe: both pre-fix 0.85 offenders now route to sentinel → Diversos@0.30 → caught by `auto_insert_excluded`.
- `config/config.json` gained the `taxonomy_path` key classify has hard-required since T-13 (was missing; probe failed without it).
- Opened PR #40 (`feat/t19-sentinel-decline` → master, 2 commits).
- tasks.md updated in-session at user request: T-19 closed with residual-risk note; T-22 (taxonomy descriptions) added.
- User renamed the taxonomy leaf "alguma coisa sindicato" → "extra sindicato" (it was acting as the model's improvised dumping ground).

### Decisions Made

- Sentinel is enum+prompt surface only; the pipeline never sees it — `splitResults` converts it to a normal Diversos result, so no consumer schema changes (review/apply/feedback untouched).
- Model-reported confidence on a decline is discarded (fixed 0.30): it scores the forced pick, not pre-constraint uncertainty.
- Residual confident-wrong risk (plausible-looking wrong leaves still return 0.95, e.g. drone→Lazer/Diamba) is a model-accuracy problem, routed to T-14 — do NOT start WS-D before benchmarking.
- T-22 (type-level taxonomy `description` field for the classifier prompt) logged as an idea coupled to T-14; open design question is the authoring home (export flow vs sidecar) since taxonomy.json regeneration would wipe it.

### Next

- Review + merge PR #40 (T-19 sentinel).
- T-14 benchmark, now carrying three riders: model accuracy+speed (q3/q35/qcoder), sentinel-decline rate on out-of-domain items, and the T-22 descriptions A/B.
- Then WS-D (retire bare-name fallback, T-09) — T-19 structural gap is closed, but gate on T-14 results.

### Gotchas

- qcoder full-file rewrites time out even warm — the 30B is CPU-offloaded on the 12 GB GPU and ~300 lines of output exceeds the window. Ask for snippet-sized output and splice; also q3 probe runs evict qcoder (real cold-starts between interleaved model use).
- q3's self-reported confidence has large run-to-run variance on the same items — another argument that auto-insert safety belongs to T-14 calibration, not prompt surgery.
- User feedback saved to memory: grade `generate_code` output (verdict) before/as it lands in the tree, not after the `output_file` write.
