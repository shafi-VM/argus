# P4 — Results

**Question (A5):** does the hero dashboard provision from JSON and render from data?

**The dashboard is a contract** (`dashboard.json`): every panel ↔ exactly one metric, low-cardinality labels only. Build the panels to those metric names so the mock emitter AND the real Go/Python instrumentation target one schema — zero rework later.

## Steps
1. Build the "Intelligence Health" dashboard in the SigNoz UI (panels listed in `dashboard.json`).
2. Feed it **mocked OTel metrics** (low-cardinality) so panels populate.
3. **Export** the dashboard JSON → overwrite `dashboard.json` here.
4. Import it into a **fresh** SigNoz instance → confirm every panel renders.
5. Screenshot each panel → `screenshots/`.

## Verdict
- Exported JSON re-imports cleanly on a fresh instance → ☐ PASS / ☐ FAIL
- All panels render from seeded metrics (no blank panels) → ☐ PASS / ☐ FAIL
- The 🟢 Infra vs 🔴 Intelligence split reads clearly on a projector → ☐ yes / ☐ no

**This is the pitch artifact.** Build it Day 1 from mocks; it de-risks *and* becomes the demo.

→ update `../SUMMARY.md`

## RESULTS — 2026-07-09 🟢
Built **as code** via the SigNoz API (`build_dashboard.py`) — no manual clicking. Dashboard `019f4602-...`, **8 panels**, all returning live data:
- gauges (infra/intelligence health, cost, users): `timeAggregation:avg, spaceAggregation:avg`; value panels `reduceTo:last`
- counters (grounding/loops/recoveries): `timeAggregation:rate, spaceAggregation:sum` (Sum, cumulative)

Verified every panel against `/api/v5/query_range` (real values: infra=1.0, intelligence 0.85→0.26, grounding ~0.67/min). Clean re-importable JSON vendored to `dashboard.json`.

**Unlocks:** the **v5 query API** is the LEARN read path. Metric contract (`argus_*`, low-cardinality labels) locked. Visual screenshot = final pitch tick (pending).

> ⚠️ Correction (see Day-4 note): server *latency* is ~85ms, but query *freshness* is not. Measured 2026-07-22: metric `query_range` lags ~60s behind ingest; trace `query_range` ~13s. LEARN therefore reads the **trace** decision signal, not the metric gauge (`internal/learn/signoz.go`). The ~85ms figure was latency-not-freshness — the exact assumption this proof re-tested.

## DAY 4 — fast (trace-derived) moving panels 🟢 — 2026-07-22
The original 8 panels are all `dataSource:metrics`. Fine for the **flat** infra gauge (a constant line looks identical lagged), fatal for the **moving** intelligence signal in a live 5-min demo (~60s lag ⇒ fall/recover shows a minute late). `add_trace_panels.py` appends two `dataSource:traces` panels reading `argus.decision` spans (~13s fresh — the same signal LEARN's loop reads, so dashboard and loop agree):

- **Decisions over time** — `count()` grouped by `argus.decision`, stacked. The money visual: green `pass` bars displaced by amber `recovered` under drift, then returning. No formula (highest confidence).
- **Intelligence Health — live** — formula `A/B` = `pass / (pass+recovered+refused)`; `upstream_error`/`transport_error` excluded (red-team R2). The fast twin of the laggy metric gauge.

Both stored in canonical **v5 builder** shape (SigNoz normalized the POST) and verified render-correct against `/api/v5/query_range`: grouped query returns `[recovered, pass]` series; formula legs `pass=61, behavioral=499 → 0.122`. Idempotent (replaces panels tagged `[traces]`); vendors the full metrics+traces dashboard back to `dashboard.json`.

**Retires:** the #1 demo-freshness risk — the hero screen now shows infra flat-green beside intelligence moving live, at trace speed. Run after `build_dashboard.py`:
`SIGNOZ_URL=… SIGNOZ_API_KEY=… python add_trace_panels.py`
