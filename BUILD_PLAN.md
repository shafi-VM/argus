# BUILD_PLAN.md — the 7-day hackathon sprint (Jul 20–26)

**Rule lock:** nothing here starts before **Jul 20**. This is a *plan* (allowed), not code (locked).
**Scope is frozen:** the two superpowers (PREVENT, LEARN) + the demo. See `VISION` / `DEMO_SCRIPT` / `KILL_LIST`.
**Rigor rule (Abhishek's bar):** no milestone is "done" until it's been *watched working, adversarially.*
No "should work." No green off a happy path that ran once.

## The one rule of sequencing
Build **the demo's money moment first**, then widen. Everything serves the 30 seconds where a bad
answer gets caught and fixed while the user never notices.

## Ownership (per ADR-0001)
- **Eng A (Go):** `argusd` — gateway, PREVENT, LEARN poller, recovery, chaos, Mission Control page.
- **Eng B (Python):** the demo agent "Ada", OTel instrumentation, the P7 bake-off, demo/pitch assets.
- **Then swap as red-teamers:** each attacks the other's work before it's called green.

## Day-by-day

### Day 1 (Jul 20) — The spine
- **A:** `argusd` skeleton — proxies one LLM call, emits a real OTel span (CLIENT kind), propagates `traceparent`.
- **B:** "Ada" agent (OpenAI Agents SDK or LangGraph) → routes through `argusd` → emits INTERNAL spans.
- **Together:** one real request flows agent → gateway → LLM, and **one linked trace shows in SigNoz.**
- ✅ Retire first **P8** ticks (real span names, span kind, `gen_ai.provider.name`, resource attrs).
- 🎯 EOD gate: a **real** trace in SigNoz, not a mock.

### Day 2 (Jul 21) — PREVENT (the money moment)
- **A:** deterministic grounding check inline (<50 ms); block ungrounded answer → re-ground → retry.
- **A:** recovery span shares the request `trace_id` (P3 mechanic, now real).
- **B:** chaos toggle #1 — inject a deterministic hallucination on cue.
- 🎯 EOD gate: **agent hallucinates → caught → corrected → user never notices**, as one SigNoz trace.

### Day 3 (Jul 22) — LEARN + real metrics
- **A:** LEARN poller — `/api/v5/query_range` over a window → quarantine/reroute on drift/cost.
- **A/B:** emit real `argus_*` metrics from `argusd` (retire the mock emitter). Hero dashboard now **real**.
- **B:** chaos toggle #2 — subtle drift (quality decays; every call still returns 200).
- 🎯 EOD gate: **drift over a window → SigNoz catches → Argus quarantines**, dashboard shows it live.

### Day 4 (Jul 23) — Mission Control + one chaos button + security beat
- **A:** Mission Control page served by `argusd` (status + last action). **One** deterministic chaos button firing the 2–3 demo faults (hallucination, drift, injection) — per KILL_LIST, **no suite** (see #19).
- **B:** injection/exfil beat; the "vaccination" fast-forward run.
- 🎯 EOD gate: every beat in `DEMO_SCRIPT.md` exists and fires from the god-mode panel.

### Day 5 (Jul 24) — Demo timing + judge experience
- Measure every beat end-to-end (< 10 s); perception budget (**P6**).
- Record the **backup video** of a flawless run. First offline / airplane-mode run.
- ✅ Retire **P5, P6.**

### Day 6 (Jul 25) — Judge install + observability quality + hardening
- Vendor the compose; `docker compose up` → hero dashboard in < 10 min, zero manual steps (**P9**).
- Tick the **P8** semconv checklist against real telemetry; fix any tells.
- Dress rehearsal #1 on the real laptop, wifi off.
- ✅ Retire **P9, P8.**

### Day 7 (Jul 26) — Rehearse, launch, submit
- Dress rehearsal #2 (timed, wifi off). Pitch polish. Flip repo **public** (OSS launch).
- Submit. Final backup-video check.

## Proof retirement map
`P5, P6 → Day 5` · `P8, P9 → Day 6` · `P7 (hands-on) → before Jul 20 if we do it now, else Day 6`

## Pre-hackathon readiness checklist (Jul 18–19 — do NOW, all rule-legal)
- [ ] SigNoz stack green + one-command bring-up re-verified (Foundry).
- [ ] LLM API key(s) ready (OpenAI/Anthropic); spend cap set.
- [ ] Service-account key + **role** documented; `SIGNOZ_URL`/`SIGNOZ_API_KEY` in a gitignored `.env`.
- [ ] Offline demo-image bundle saved (`docker save`) for airplane-mode.
- [ ] **P7 hands-on bake-off done** (optional-but-ideal) → P7 🟢.
- [ ] Repo clean; issues/milestones created; both engineers know their Day-1 task cold.
