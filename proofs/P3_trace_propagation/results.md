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
