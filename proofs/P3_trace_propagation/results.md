# P3 — Results

**Question (A6):** does the recovery span land in the SAME trace as the request?

## Run
```
OTLP_ENDPOINT=localhost:4317 python trace_test.py     # -> {"trace_id": "..."}
```
Open the trace_id in SigNoz → Traces.

## Verdict
- One waterfall `agent.request > argus.recovery.reground`, same trace_id → ☐ PASS / ☐ FAIL
- `argus.behavior.blocked` event visible on the request span → ☐ yes / ☐ no
- Screenshot saved to `../P4_dashboard/screenshots/` → ☐

**Why it matters:** this is the "postmortem writes itself" beat. If recovery starts a *new* trace, the demo story breaks. Must be one trace.

→ update `../SUMMARY.md`

## RESULTS — 2026-07-09 🟢
trace_id `ccf3f2dbf6410b41d46915c38c9ee314`, both spans visible in ~1.5s:

| name | span_id | parent_span_id |
|------|---------|----------------|
| agent.request | 24a99447a44e2a10 | (root) |
| argus.recovery.reground | ddd87029c50673b2 | **24a99447a44e2a10** |

→ **PASS.** Recovery is a child of the request; one trace, correctly parented → renders as a single waterfall `agent.request > argus.recovery.reground` in SigNoz. Verified via ClickHouse `distributed_signoz_index_v3`.
