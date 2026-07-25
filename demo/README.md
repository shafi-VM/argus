# Demo runbook — one-take, measured

The 5-minute story, with the deterministic beats driven by `drive.py` and the UI beats
shown live in SigNoz. Follow this exactly and the backup video is a single take.

## 0. Bring up the stack (three terminals)

SigNoz must already be running (Foundry stack; UI on `:8081`, OTLP on `:4317`). Then:

```bash
# 1) replay engine (the deterministic "LLM")
python mockllm/mockllm.py                       # :9099

# 2) argusd — the gateway (PREVENT inline + LEARN poller). Needs a SigNoz key.
SIGNOZ_URL=http://localhost:8081 SIGNOZ_API_KEY=<service-account-key> \
OTLP_ENDPOINT=localhost:4317 ARGUS_UPSTREAM=http://127.0.0.1:9099 \
go run ./cmd/argusd                             # :8088  (Mission Control at :8088/mission)

# 3) generate a real cross-service trace for the waterfall + service-map beats:
python agent/ada.py                             # drives ada-agent -> argusd -> LLM
```

> The `SIGNOZ_API_KEY` is a local throwaway service-account key — never commit it.

## 1. Run the measured beats

```bash
python demo/drive.py
```

It drives the two deterministic system beats and prints their timings:

- **PREVENT** — a hallucination is caught and re-grounded inline; the caller receives the
  corrected answer, never the bad one.
- **LEARN** — sustained drift → Argus quarantines the bad model and reroutes → recovers;
  **every request is HTTP 200 the whole time**.

## Measured beats (2026-07-25, live SigNoz v0.132)

| Beat | Result | Timing | Budget |
|---|---|---|---|
| PREVENT (money moment) | PASS — user never saw `UA99` | **3.3 ms** round-trip incl. re-ground (grounding check itself <0.15 ms) | per-request reflex, «10s |
| LEARN (competitive kill) | PASS — 0 non-200 across the arc | **quarantine ~11s / recover ~37s** | **windowed — narrate over the dashboard, do NOT wait in silence** |

**Perception budget (honest):** PREVENT is a millisecond reflex. LEARN is a *windowed*
control loop — quarantine ~11s, recover ~37s — so it is **not** a <10s snap. Those seconds
are covered by narration over the live SigNoz dashboard (the Intelligence Health line
visibly falling, then recovering) — the wait *is* the wow, not dead air. Pre-warm ~30s of
drifted traffic before the reveal if you want the fall already on screen at the cut.

## UI beats — where to look (presenter-driven)

| Beat | Where | What to show |
|---|---|---|
| Cold open | SigNoz dashboard "Argus — Intelligence Health" | 🟢 `argus_infra_health_ratio` ~1.0 beside 🔴 `argus_intelligence_health_ratio` collapsing; the "Decisions over time" trace panel flips green `pass` → amber `recovered` |
| Incident trace | SigNoz Traces | one waterfall: `invoke_agent ada` → `chat gpt-4o` (`argus.decision=recovered`) → `argus.recovery.reground` |
| SigNoz is the brain | SigNoz Traces / the `argus.learn.*` spans | Argus's quarantine decision was computed *from* a SigNoz `query_range` — turn SigNoz off and Argus goes blind |
| Service map | SigNoz Services | `ada-agent → argusd` (drive traffic **through Ada**, not raw curl, or the edge won't appear) |
| Mission Control | `http://localhost:8088/mission` | System / PREVENT / LEARN state + last action + the ONE chaos button |

## Known-honest limits (say these if asked)

- **Metric→trace exemplar click-through** does not render — SigNoz stores no metric
  exemplars (no `trace_id` on `samples_v4`). Our telemetry emits them; the platform
  can't store them. Correlate by attribute+time instead.
- Grounding is **entity-presence** vs the retrieved context — it blocks emitting an
  identifier not in context (incl. the exfil class), **not** general prompt injection, and
  a **poisoned context** defeats it (see `internal/grounding/exfil_corollary_test.go`).
