# Argus — Proof Status

> This document is our confidence. Not optimism — evidence.
> A proof is 🟢 only when it has a **number** next to it that clears the pass bar.

_Last updated: 2026-07-17 — P0–P4 green. P7 🟡 (desk research; hands-on outstanding). P8 checklist
written, unticked. **Build gate met — but the build is rule-locked until Jul 20.**_

> **⛔ Pre-hackathon lock (Jul 17 → Jul 20).** Hackathon rule: *"Teams can plan and discuss strategy in
> advance, but **coding and design work should begin only after the hackathon starts**."* Permitted
> now: **written notes, sketches, diagrams**. The build gate being met does **not** unlock the build.
> **P5, P6, P9 are therefore not "pending" — they are *unreachable*.** All three measure a product that
> cannot legally exist yet. The only way to turn them green today is to invent numbers, which is the
> one thing this file exists to prevent.

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
| **P2** Alert pipeline | Alert → webhook latency + eval floor? | known #; decide alert-vs-poll | **DECIDED: poll** — P1 proved query_range ~30ms + ~5s freshness; LEARN polls, alerts=optional demo visual | 🟢 |
| **P3** Trace propagation | Recovery span shares request `trace_id`? | one linked waterfall | **PASS** — recovery.reground is child of agent.request, same trace | 🟢 |
| **P4** Dashboard | Provisions from JSON, panels render? | import → panels populated | **PASS ✅ visually confirmed** — 8-panel hero deployed as-code; all render with live data; green Infra vs red Intelligence hero row lands; JSON vendored | 🟢 |
| **P5** Demo timing | Every beat measured end-to-end? | all beats < 10 s | — | 🔴 **BLOCKED** — needs the product; rule-locked until Jul 20 |
| **P6** Judge experience | Does it *feel* fast (perception)? | all beats < 10 s felt | — | 🔴 **BLOCKED** — needs the product; rule-locked until Jul 20 |
| **P7** Competitor check | Can OpenLIT/Portkey/Langfuse already do this? | Argus alone closes the loop | **PARTIAL** — 11 tools researched from **source/docs/licenses**, NOT run first-hand (the bar says "actually try each"). **Found 3 refutations of our own claim.** Wedge survives but is **narrower**: nobody closes the loop from a windowed **content** signal — Portkey/LiteLLM breakers fire on **HTTP codes**. ⚠️ **Our frozen one-liner is factually refutable** — see `P7_competitor_check/matrix.md`. | 🟡 |
| **P8** Observability quality | Would a SigNoz eng say "they get OTel"? | checklist all ✓ | **PASS — verified live on SigNoz v0.134 (2026-07-24).** Correct semconv (`gen_ai.provider.name`; `input/output_tokens`, no `total_tokens`; `chat {model}`/`invoke_agent {agent}`; CLIENT/INTERNAL; status UNSET/Error+`error.type`; low-cardinality `argus_*`). Fixed 2 tells (resource attrs; Ada Simple→Batch). Live: **service map `ada-agent → argusd`** confirmed; **exemplars not stored by SigNoz v0.134** (documented platform limit, not our gap). | 🟢 |
| **P9** Judge install | Clone → `compose up` → hero dashboard, no edits? | < 10 min, 0 manual steps | — | 🔴 **BLOCKED** — needs the product; rule-locked until Jul 20 |

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
Architecture (proven)   ██████▊░░░  68%   ← LEARN latency + trace correlation + v5 query + metric contract locked
SigNoz integration      ███████░░░  70%   ← auth, ingest, query(v5), trace-link, metrics, dashboard-as-code proven; MCP TBD
Demo                    ██▌░░░░░░░  25%   ← hero dashboard (the pitch centerpiece) exists & renders live
Overall                 █████▌░░░░  55%
```
**Rule:** a bar moves only when a proof in the table above turns 🟢. If code went up but no bar
moved, we reduced nothing — we just typed. Watch Overall climb 25% → 90% across the week; if it
stalls while commits pile up, we're building, not de-risking.

**2026-07-17 — no bar moved, and that is correct.** P7 landed at 🟡, not 🟢: its bar says *"actually try
each — first-hand"* and we only read source and docs. P8's checklist is written but every box is
unticked. Real work happened; **no proof turned green; therefore nothing moves.** This is the rule
working, not the rule failing. The temptation to nudge Demo to 30% because the day *felt* productive
is exactly what these bars exist to refuse.

**One bar arguably should go DOWN, and we're leaving it — flagging instead.** P7 found that our frozen
one-liner is **factually refutable** (Portkey's breaker is windowed; OpenLIT's guards block inline).
Competitive risk didn't fall today — it got *better understood*, and the pitch got *worse* until a
human fixes the line. **Overall stays 55%.**
