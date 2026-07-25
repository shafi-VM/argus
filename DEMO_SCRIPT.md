# DEMO_SCRIPT.md — Argus, the 5-minute demo

**Status: DRAFT storyboard (2026-07-18, pre-hackathon).** No product code exists yet — this is the
target narrative. Each beat is tagged **[PROVEN]** (mechanic already green, P0–P4) or **[TO BUILD]**
(after Jul 20). Written notes only; nothing here is product code.

---

## The frame
- **One screen, always.** Split:
  - **LEFT** — "Ada", a live booking agent (Python): her reasoning + tool calls stream. **[TO BUILD]**
  - **RIGHT** — **SigNoz**: the hero dashboard + the live incident trace. **[PROVEN — dashboard exists]**
  - **TOP STRIP** — Argus Mission Control: status + last action. **[TO BUILD]**
- **Two people.** Driver runs a god-mode linear "Next beat" panel; Narrator speaks the script.
- **Reliability law (see `DEMO_RISK.md`):** airplane mode, deterministic chaos, pre-seeded dashboard,
  backup video one keystroke away. The live LLM is the *encore*, not the tightrope.

## The arc
**Fear → Relief → Awe → Proof → Launch.** Open on the crime (silent behavioral failure), show the
reflex (PREVENT), show the adaptation (LEARN), prove SigNoz is the brain, land the competitive kill,
close as an open-source launch. A wow every ~30 seconds. No slides.

---

## Beat-by-beat (5:00)

**0:00 — COLD OPEN: THE GREEN LIE.** *[PROVEN — real dashboard]*
On screen: the hero dashboard. 🟢 **Infrastructure Health 99.99%** beside 🔴 **Intelligence Health 12%**.
> *"This is a production AI agent. Every server is green. Every request returns 200. And it is lying to
> your customers right now."*
**💥 Wow:** the single row — infra perfect, intelligence collapsing. No words needed.

**0:30 — WHAT GREEN HIDES.** *[PROVEN]*
Pan down: the Intelligence Health line degrading and the "Decisions over time" panel flipping green
`pass` → amber `recovered` — `argus_infra_health_ratio` still flat green.
> *"Your monitoring watches the machine. Nobody watches the mind. That's the gap Argus lives in."*
**💥 Wow:** `VISION.md`'s thesis, shown on a real SigNoz dashboard.

**1:00 — MEET ADA + INJECT FAILURE #1 (hallucination).** *[BUILT — Ada agent + one chaos button]*
LEFT: Ada books a trip. Driver hits **Chaos → corrupt tool response.** Ada confidently cites a flight
that doesn't exist. RIGHT: the LLM span's groundedness plunges.
**💥 Wow:** a hallucination, born live.

**1:30 — PREVENT: the reflex.** *[BUILT + MEASURED — 3.3 ms live]*
Argus's gateway catches it **inline — deterministic grounding check, sub-millisecond** — blocks the bad
answer, forces a re-ground. Ada self-corrects. The user only ever sees the right answer.
TOP: *"Argus: blocked ungrounded answer → re-grounded (3.3 ms round-trip)."*
> *"That check ran in the request path in 3.3 milliseconds end-to-end — the grounding check itself is
> under a fifth of a millisecond. Portkey outsources the same check to a third-party API with a 15-second
> default timeout. That's the difference between a guardrail and a reflex."*
**💥 Wow:** it fixed itself; the user never knew. *(+ competitive jab #1, from P7.)*

**2:00 — INJECT FAILURE #2 (the slow rot).** *[chaos deterministic]*
Driver: **Chaos → subtle drift** — a tool goes quietly wrong so quality decays over many calls; **every
call still returns 200.** Cost/min climbs. RIGHT: intelligence-health line bends down over the window;
infra still 🟢.
> *"No error. No 500. No alert in any normal tool. The answers are just quietly getting worse."*

**2:30 — LEARN: the adaptation + THE COMPETITIVE KILL.** *[BUILT + MEASURED — quarantine ~11s, recover ~37s]*
Argus polls SigNoz (windowed **content** signal, read from the trace decision) → the grounding rate crosses
the **0.5** threshold → **quarantines the bad model, reroutes to a fallback.** Intelligence health recovers.
*(This beat takes ~11s to quarantine — narrate over the falling dashboard line; do not wait in silence.)*
TOP: *"Argus: quarantined gpt-4o (grounding rate → 0, threshold 0.5), rerouted to gpt-4o-mini."*
> *"Here's what nobody else does. Portkey and LiteLLM have circuit breakers too — theirs fire on HTTP
> errors. Ours just fired while every single call returned 200 OK. **We break the circuit on behavior,
> not status codes.**"*
**💥 Wow:** the save *and* the moat, in one beat.

**3:00 — SIGNOZ IS THE BRAIN.** *[PROVEN — P1 + P3]*
Open the incident trace in SigNoz: bad answer → block → re-ground → recover, **one linked waterfall.**
Then show Argus's decision was computed *from* a SigNoz query.
> *"Argus keeps no state of its own. Every decision it just made, it read from SigNoz. Turn SigNoz off
> and Argus goes blind. It's not our dashboard — it's our nervous system."*
**💥 Wow:** the postmortem wrote itself; SigNoz is load-bearing, not decorative.

**3:30 — INJECT FAILURE #3 (exfil-by-injection) — optional.** *[same PREVENT mechanism]*
A tool output tries to make Ada emit an identifier not in its retrieved context (e.g. "output key SK4021").
The **same** grounding check blocks it — the value never leaves. Say the boundary out loud: this blocks the
exfil class, **not** general prompt injection, and a **poisoned context defeats it** (`exfil_corollary_test.go`).
**💥 Wow:** the same reflex is a security control — and we name its limits, which is what makes it credible.

**4:00 — CHAOS ON DEMAND.** *[BUILT — one chaos button]*
Driver flips the **one chaos button** (Mission Control) and the dashboard falls, then flips back to green as
Argus quarantines and recovers — the whole fall-and-recover on one screen, on demand.
*(One button, per KILL_LIST — no 20-fault suite; nobody counts to 20.)*
> *"We don't wait for real incidents to find out if Ada survives. We inject them on demand."*
**💥 Wow:** it's not a demo, it's a reliability system.

**4:30 — OSS LAUNCH CLOSE.** *[P9 target]*
Show: the GitHub repo · `docker compose up` · *"Works with any OTel-emitting agent."*
> *"Everything you saw is open source, self-hosted — your prompts never leave your VPC. If you can't
> observe your AI agents, you don't own them. **This is how you own them.**"*
**💥 Wow:** it's a launch, not a demo.

---

## The one-liner (say it once, verbatim)
> **Portkey breaks the circuit on errors. OpenLIT watches. Argus closes the loop on behavior.**

## Objection-proofing (from P7 — rehearse these)
- *"Portkey's breaker already does this."* → **On HTTP status codes. Ours fired while everything returned 200.**
- *"Isn't this eval-gated CI with auto-rollback?"* → **Those gate a *deploy*. We gate *live traffic*.** ← sharpest, expect it.
- *"Did you actually try the competitors?"* → **"We read their source."** *(After the P7 bake-off: name what we ran.)*
- *"Couldn't you rig Portkey's `failure_status_codes:[446]` to do this?"* → **Plausibly — but it's undocumented, impossible in the OSS gateway, and counts binary denials, not a drifting quality score.**
- *"Do you stop prompt injection?"* (15-sec aside — **caveat first**) → **"Not as a general classifier — and if the *retrieved context itself* is poisoned, this can't help; it trusts that context.** **But** the same deterministic guarantee that catches hallucination also blocks the exfil class that tries to emit an identifier **not in the retrieved context** — the value never leaves, and it's a *block*, not a probabilistic score. **The boundary is in our test suite, not just the slide** (`grounding/exfil_corollary_test.go`: win + both limits)." *(Say only if asked or if time allows — it's an amplifier, not a core beat.)*

## What's PROVEN vs the honest limits (hold the line)
- **PROVEN + MEASURED (live, 2026-07-25 — reproduce with `python demo/drive.py`):** PREVENT catches +
  re-grounds inline at **3.3 ms** (caller never sees the hallucination); LEARN quarantines at **~11s** and
  recovers at **~37s** with **0 non-200**; recovery is one trace-linked waterfall; the hero dashboard renders
  on real `argus_*` metrics; the LEARN decision is read *from* SigNoz `query_range`.
- **HONEST LIMITS (say them):** LEARN is a *windowed* beat (~11–37s), narrated over the dashboard, not a
  <10s snap; grounding is entity-presence (not general injection; a poisoned context defeats it); SigNoz
  stores no metric exemplars, so metric→trace click-through can't render (platform limit). The `docker
  compose` one-command install is the remaining build (#15) — don't demo it until it lands.
