# P1 — Results

**Questions:** (A4) How long until emitted telemetry is queryable? (A1) How fast is `query_range`?

## Run
```
# terminal 1
OTLP_ENDPOINT=localhost:4317 python emit_otlp.py            # -> {"run_id":"...","emit_ms":...}
# terminal 2
SIGNOZ_URL=http://localhost:8080 SIGNOZ_API_KEY=<PAT> RUN_ID=<...> EMIT_MS=<...> python query_latency.py
```

## The four timestamps (measure the whole waterfall, not just one number)
```
T0  emit OTLP           emit_otlp.py -> emit_ms
T1  collector received  (optional) collector debug exporter, verbosity: detailed
T2  queryable           query_latency.py first hit -> ingestion_lag = T2 - T0
T3  visible on dashboard = T2 + panel refresh cadence (a UI setting, NOT a bug)
```
**Diagnosis:**
- **T0→T2 slow** → ingestion (collector/ClickHouse batching). *This is the one that gates LEARN.*
- **T2→T3 slow but T0→T2 fast** → UI/refresh only; not an architecture risk. Lower the panel refresh interval.

## Numbers (repeat 5×, record median)
| Run | T0→T2 ingestion_lag_s | query_rtt_ms (T2 read) | T2→T3 refresh_s |
|-----|-----------------------|------------------------|-----------------|
| 1 | | | |
| 2 | | | |
| 3 | | | |
| 4 | | | |
| 5 | | | |
| **median** | | | |

## Verdict
- A4 ingestion lag < 10 s → ☐ PASS / ☐ FAIL
- A1 query rtt < 2 s → ☐ PASS / ☐ FAIL
- **Consequence:** if A1 FAILs, LEARN can't poll fast enough → fall back to alert-driven (P2). If both PASS, `query_range` polling is our LEARN trigger.

→ update `../SUMMARY.md`

## Auth note (SigNoz v0.132.0 — discovered in P0)
- OTLP **ingestion needs no auth**; `emit_otlp.py` works as-is (marker span already accepted).
- `query_range` **requires a PAT**. v0.132 has no scriptable JSON login (`/api/v1/login` serves the SPA). Create a PAT once in the UI: **http://localhost:8081 → Settings → API Keys**, then:
  ```bash
  export SIGNOZ_URL=http://localhost:8081 SIGNOZ_API_KEY=<pat>
  ```
- Admin already provisioned via `/api/v1/register`: `argus@local.dev` (local/throwaway creds).
- Service account **needs a role** (`signoz-viewer`/`editor`/`admin`) or the key gets **403** `"only viewers/editors/admins can access this resource"`.

## RESULTS — 2026-07-09 🟢
**A1 query API RTT:** ~25–56 ms (100+ samples, `/api/v4/query_range`) → **PASS (<2s)**.
**A4 ingestion lag (emit → queryable):**

| run | lag_ms |
|-----|--------|
| 1 | 1518 |
| 2 | 4346 |
| 3 | 4917 |
| 4 | 4701 |
| 5 | 5351 |
| **median** | **4701 (~4.7s)** |

→ **PASS (<10s)**. Range 1.5–5.4s (includes ~0.3s poll granularity + docker-exec overhead → true lag slightly lower).

**Method:** measured against ClickHouse (`signoz_traces.distributed_signoz_index_v3`, `attributes_string['argus.run_id']`) — the store SigNoz queries. Query layer adds only the ~30ms RTT above, so emit→ClickHouse ≈ emit→queryable.

**Architecture implication:** **LEARN loop is viable** — behavioral data queryable within ~5s, inside a windowed-detection budget. Confirms ADR-0003 (PREVENT inline; LEARN tolerates ~5s). No ADR changed.

**Follow-up (not blocking):** v4 *list* query needs `aggregateOperator:noop` + `selectColumns`; or use v5 `/api/v5/query_range`. Nail the exact payload when wiring the real LEARN poller.
