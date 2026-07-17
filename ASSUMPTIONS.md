# ASSUMPTIONS.md — the engineering bible

No opinions. Only experiments. An assumption is not "true" until it has a
**green result and a number next to it**. Until then it is a risk, not a fact.

Prove the ground before pouring concrete. Nothing gets built on top of a 🔴.

_Last synced with `proofs/SUMMARY.md`: 2026-07-17._

| # | Assumption | Status | Result (the number) | Proof | Pass bar |
|---|-----------|--------|---------------------|-------|----------|
| A1 | SigNoz **Query API (`query_range`)** latency is acceptable for LEARN decisions | 🟢 **Proven** | **~30 ms** RTT, 100+ samples (`/api/v5/query_range` ~85 ms incl. dashboard panels) | P1b | < 2 s |
| A2 | **SigNoz MCP** is suitable for the agent "investigation" beat | 🔴 Unproven | — (needs a prototype → **blocked until Jul 20**) | — | Works; seconds OK (beat is optional) |
| A3 | **Alert → webhook** latency + min eval interval | 🟢 **Retired by decision** | Alert latency never measured — **made moot**: P1 showed poll = ~30 ms + ~5 s freshness, so LEARN polls. Alerts = optional demo visual, not the action path. | P2 | Known number; decide alert-vs-poll |
| A4 | **ClickHouse ingestion lag** (OTLP emit → queryable) | 🟢 **Proven** | **~4.7 s median** (1.5–5.4 s range), ClickHouse-measured | P1a | Single-digit seconds |
| A5 | **Dashboard provisioning** from JSON works on `compose up` | 🟡 **Partial** | 8-panel hero **deployed as-code via the API**, all panels return live data, `dashboard.json` vendored. **The `compose up` half is unproven** — auto-provision-at-boot is P9. | P4 | Panels render from seeded data |
| A6 | **Trace correlation** end-to-end (recovery span shares `trace_id`) | 🟢 **Proven** | `argus.recovery.reground` lands as a **child** of `agent.request`, same trace (`parent_span_id` == `span_id`) | P3 | One waterfall: bad→block→reground→ok |
| A7 | **OTel metric cardinality** is manageable | 🟡 **Partial** | Label contract **locked & flowing** (`argus_*`, low-cardinality only). **Burst load-test not run** — the "watch CH under load" half is untested. | P4 (partial) | No prompt/user_id/trace_id on metric labels |
| A8 | **PREVENT** deterministic grounding check (Go-native, no inline ML — ADR-0002) meets latency budget | 🔴 Unproven | — (needs Go code → **blocked until Jul 20**) | — | Adds < 50 ms p95 |
| A9 | The demo can run **100% offline** (no venue network on critical path) | 🔴 Unproven | — (this box is behind NAT; needs a **60 s human step** on the demo laptop) | P0 (partial) | Complete run, wifi off |
| A10 | **Deterministic replay** LLM produces the scripted bad→good behavior | 🔴 Unproven | — (needs the replay harness → **blocked until Jul 20**) | — | Identical every run |

**Score: 4 proven · 2 partial · 4 unproven.** Of the 4 unproven, **3 (A2, A8, A10) require code** and are
therefore blocked by the hackathon rule below. **A9 needs a human, not an engineer.**

### The pre-hackathon constraint (Jul 17 → Jul 20)
Hackathon rule: *"Teams can plan and discuss strategy in advance, but coding and design work
should begin only after the hackathon starts."* Permitted before Jul 20: **written notes, sketches,
diagrams**. Every 🔴 that needs code (A2, A8, A10) is therefore **parked until Jul 20 by rule, not by
capacity**. Do not "get a head start." A disqualified project retires no assumptions at all.

### Rule
Every engineering session picks a 🔴 and turns it green with a measurement.
We do not build features. We retire assumptions.

### Sync rule (added Day 4 — we broke it once)
This file and `proofs/SUMMARY.md` **must not disagree**. On 2026-07-17 this table still read all-🔴
while five proofs had been green for a week — the bible was the last file to hear the news. When a
proof flips in `SUMMARY.md`, flip it here **in the same commit**. A stale bible is worse than no
bible: it silently invites re-proving what's proven, and hides what isn't.
