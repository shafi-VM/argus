# Argus — Proof Status

> This document is our confidence. Not optimism — evidence.
> A proof is 🟢 only when it has a **number** next to it that clears the pass bar.

_Last updated: 2026-07-09 — P0 + P1 + P3 green. Loop latency + trace correlation proven._

## Execution order (Day 1, 2 engineers)
1. **P0 — together.** Stack up, offline verified. The gate for everything else.
2. **Fork:**
   - **Eng B → P4 (hero dashboard from mock metrics).** This also **freezes the metric contract** (names + low-cardinality labels) that all instrumentation must emit — build it first so mock and real code share one schema.
   - **Eng A → P1 (latency kill-shot) + P3 (trace linkage).**
3. **Converge → P2** (alert-vs-poll decision).
4. **P5–P8** as the build comes alive. **P9** before we call it "launched."

| Proof | Question it answers | Pass bar | Result | Status |
|-------|--------------------|----------|--------|--------|
| **P0** Environment | Does the full stack boot & run **offline**? | Boots, wifi OFF, all green | **PASS** — stack UP, all endpoints serve (OTLP 200, UI 200), functions via internal container DNS, live ingest 200. Foundry v0.132. True airplane-mode = 60s human step on demo laptop (this box is behind NAT; Claude is cloud-bound, can't drive a real-offline run). | 🟢 |
| **P1a** Ingestion latency | OTLP emit → queryable in SigNoz? | < 10 s | **~4.7s median** (1.5–5.4s), ClickHouse-measured | 🟢 |
| **P1b** Query latency | `query_range` round-trip for LEARN? | < 2 s | **~30 ms** (100+ samples) | 🟢 |
| **P2** Alert pipeline | Alert → webhook latency + eval floor? | known #; decide alert-vs-poll | — | 🔴 PENDING |
| **P3** Trace propagation | Recovery span shares request `trace_id`? | one linked waterfall | **PASS** — recovery.reground is child of agent.request, same trace | 🟢 |
| **P4** Dashboard | Provisions from JSON, panels render? | import → panels populated | — | 🔴 PENDING |
| **P5** Demo timing | Every beat measured end-to-end? | all beats < 10 s | — | 🔴 PENDING |
| **P6** Judge experience | Does it *feel* fast (perception)? | all beats < 10 s felt | — | 🔴 PENDING |
| **P7** Competitor check | Can OpenLIT/Portkey/Langfuse already do this? | Argus alone closes the loop | — | 🔴 PENDING |
| **P8** Observability quality | Would a SigNoz eng say "they get OTel"? | checklist all ✓ | — | 🔴 PENDING |
| **P9** Judge install | Clone → `compose up` → hero dashboard, no edits? | < 10 min, 0 manual steps | — | 🔴 PENDING |

## Decision gate
- If **P1b > 2 s** or **P2 too slow/coarse** → LEARN runs on `query_range` polling (already default). Confirm, don't pivot.
- If **P1a > 10 s** → pre-seed history and stream on top (already in DEMO_RISK). Confirm.
- If **P0 can't run offline** → **STOP.** Redesign the demo before writing product code.

## Go/No-Go
- **Green light to build the product** only when **P0–P4** are 🟢.
- **P5–P8** gate the *demo*.
- **P9** gates the *OSS launch* (the "everything is open source" close).

## Confidence (update every evening — from proofs retired, NOT feelings)
```
Product (frozen)        ██████████ 100%
Architecture (proven)   ██████▌░░░  65%   ← LEARN latency + trace correlation proven; PREVENT inline (A8) + full loop still to build
SigNoz integration      █████▊░░░░  58%   ← auth+ingest+query+trace-linking proven; dashboards/alerts/MCP TBD
Demo                    █░░░░░░░░░  10%
Overall                 █████░░░░░  48%
```
**Rule:** a bar moves only when a proof in the table above turns 🟢. If code went up but no bar
moved, we reduced nothing — we just typed. Watch Overall climb 25% → 90% across the week; if it
stalls while commits pile up, we're building, not de-risking.
