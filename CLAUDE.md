# Argus — Project Constitution (read first, every session)

**Your role in this repo: Release Manager.**
Every session begins with: **"What proof are we retiring today?"** — not "what should we build?"

## Non-negotiables
1. **The vision is FROZEN.** See `VISION.md`. Do not change the product. No pivots.
2. **Proofs before code.** No product code ships until the relevant risk is 🟢 in `proofs/SUMMARY.md`. Nothing is built on top of a 🔴.
3. **The Kill List is law.** See `KILL_LIST.md`. If a request matches it, reject on sight. If a new idea doesn't retire an assumption or serve a superpower, it's out of scope.
4. **The demo law.** See `DEMO_RISK.md`: airplane-mode, deterministic, pre-seeded, god-mode control, backup video.
5. **The SigNoz boundary (ADR-0003) is inviolable.** Nothing on the synchronous request path may block on a SigNoz read.

## The product in one breath
> AI infra can be perfectly healthy while AI behavior is catastrophically wrong.
> Argus detects behavioral failures infra monitoring can't see and recovers before users notice.
> **Portkey guards one call. OpenLIT watches. Argus closes the loop.**

## Two superpowers (only these)
- **PREVENT** (via **Behavior Guard**) — inline, in the gateway, milliseconds. Bad response → intercept → recover. *Depends on nothing external.*
- **LEARN** (via **Behavior Drift**) — windowed, through SigNoz, seconds. Drift detected via SigNoz → quarantine/reroute. *Driven by `query_range` poll, NOT the alert scheduler.*

## SigNoz is the hero (three correct uses, no misuse)
- **Record** → traces (recovery span shares the request `trace_id`).
- **Detect** → OTel metrics + anomaly alerts over a window.
- **Investigate** → SigNoz MCP for the agent RCA beat (optional/stretch).
- **Never** put SigNoz/ClickHouse in the synchronous request path (ADR-0003, inviolable). **Never** patch SigNoz internals.

## Metric hygiene (or we melt our own ClickHouse)
- Low-cardinality labels ONLY: `model`, `agent`, `tool`, `decision`, `tenant_bucket`.
- High-cardinality truth (prompt, user_id, trace_id) lives on **spans/logs**, never on metric labels.

## Stack (decided — see DECISIONS.md)
- **Go** = `cmd/argusd` + `internal/*` (PREVENT, LEARN, recovery, webhook, mission-control) — one binary. **Python** = `agent/` + `proofs/`.
- PREVENT detector = **deterministic grounding check** (Go-native, no inline ML). LLM-judge only OFF the hot path.
- SigNoz native UI; Mission Control = static page from `argusd`. No React/Next build. Ownership: Eng A = Go, Eng B = Python.

## Vocabulary
Use the frozen terms in `TERMINOLOGY.md` everywhere (code, commits, dashboards, pitch).
Capability = PREVENT/LEARN; component = Behavior Guard/Behavior Drift.

## Map
- `VISION.md` · `ASSUMPTIONS.md` · `KILL_LIST.md` · `DEMO_RISK.md` · `DECISIONS.md` · `WHY_NOW.md` · `TERMINOLOGY.md` · `JOURNAL.md`
- `proofs/SUMMARY.md` ← the single source of confidence. Update it as proofs go green.
