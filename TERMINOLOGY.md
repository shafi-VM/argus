# TERMINOLOGY — frozen vocabulary

We use the same words **everywhere**: code, commits, dashboards, README, pitch, Devpost. Consistent
vocabulary is part of the product (Kubernetes, Docker, Terraform, Prometheus all did this). The
"NOT" synonyms are **banned** — don't let them creep in.

## Capabilities — the two superpowers (verbs)
- **PREVENT** — the inline reflex. NOT: blocking, filtering, guarding-generic.
- **LEARN** — the windowed adaptation. NOT: analytics, monitoring, observability-generic.

## Components — what delivers them (nouns)
- **Behavior Guard** — delivers PREVENT (inline, in `argusd`). NOT: gateway, middleware, interceptor, filter, proxy.
- **Behavior Drift** — delivers LEARN (windowed, reads SigNoz). NOT: analytics engine, trend engine, monitor.
- **Recovery** — reground + retry + reroute. NOT: retry logic, fallback, error handling.
- **Investigation** — the SigNoz-MCP RCA beat. NOT: debugging, log search.
- **Chaos** — deterministic fault injection (the "vaccination"). NOT: testing, fuzzing.

## Signals & surfaces
- **Intelligence Health** — the headline metric/dashboard (behavioral vs infra health). NOT: LLM metrics, model stats.
- **Grounding Check** — the deterministic PREVENT detector (claims vs provided context). NOT: hallucination detector, groundedness model.
- **Quarantine** — isolating a bad model/agent. NOT: disable. (*Kill* is reserved for the loop-breaker only.)

## Intelligence Health — the composite, defined (resolves #21)

The cold-open number (🔴 12%). A bounded score in **[0, 1]** (rendered as %) of the agent's **raw
behavioral quality** over a rolling window. Emitted by `argusd` as the gauge
**`argus_intelligence_health_ratio`** (low-cardinality labels: `model`, `agent`, `tenant_bucket`).
**One source of truth:** the hero panel renders it; the LEARN poller reads it to decide quarantine.

```
intelligence_health = clamp( grounding_rate − loop_penalty − cost_penalty , 0, 1 )

  grounding_rate = grounded / total           # raw model answers that passed the grounding check
  loop_penalty   = min(0.30, 0.10 × loops_per_request)
  cost_penalty   = min(0.20, 0.20 × max(0, cost_per_req / budget − 1))   # penalize overruns only
```

- **Inputs are only the low-cardinality `argus_*` metrics** argusd already emits (#10):
  `argus_grounded_total` / `argus_requests_total`, `argus_reasoning_loop_total`,
  `argus_cost_usd_per_request`. `argusd` computes the gauge from its own in-process counters — **no
  SigNoz read on the request path** (ADR-0003 intact).
- **It measures RAW model quality, not delivered quality.** PREVENT keeps what the *user* sees correct
  the whole time; `intelligence_health` reflects what the *model* is producing — which is exactly what
  LEARN fixes by quarantining.
- **Demo trajectory (must be REAL, from real chaos — not a scripted gauge):** cold-open ~12% (bad
  persona, grounding failing) → dips further when drift fires (2:00) → **recovers** after LEARN
  quarantines the bad model and reroutes (2:30).
- **LEARN threshold:** quarantine/reroute when `intelligence_health < 0.5` sustained over the window
  (tunable). The same number the hero panel shows.

Weights/caps are tunable; grounding is the primary driver, loops and cost are bounded penalties so no
single component dominates. If a judge asks *"what is that number?"* — this is the answer.

### Window mechanics — specified (gap found in review of #23)
The formula alone is not enough: **where the window lives decides whether the hero number moves.**

- **Window: rolling 30 seconds** of request outcomes. Chosen for demo cadence — drift fires at 2:00 and
  the number must visibly dip by the LEARN beat at 2:30. **Lifetime counters would be sticky and would
  NOT move** — that was the gap.
- **Windowing happens in exactly ONE place: `argusd`.** It keeps time-bucketed counters
  (6 × 5 s buckets, oldest evicted) and emits an **already-windowed** gauge — never a lifetime ratio.
- **The hero panel plots the gauge RAW** — no second aggregation. The LEARN poller reads **the same
  gauge** and thresholds it raw. One home for the window ⇒ panel and LEARN can never disagree, and
  nothing is double-windowed.
- **Warm-up:** with fewer than 3 requests in the window, hold the previous value (cold start `1.0`) so a
  single request cannot slam the gauge to 0 or 1.
- **Emission cadence: every 5 s**, so the line visibly moves on demo timescales.
- **Anti-flap for LEARN:** quarantine when `intelligence_health < 0.5` across **2 consecutive windows**,
  not on a single dip.

`budget` in `cost_penalty` is a **configured constant** (`ARGUS_COST_BUDGET_USD`), **not** a metric —
#10 must not go looking for an `argus_budget` series.

## Framing (marketing skin — sparingly, NEVER in code)
"**Immune System for AI Agents**" is the *marketing* line; the immune metaphor (antibody,
vaccination…) opens the door in a pitch. The engineering names above are canonical.
Never name a function `antibody()`.

## The one-liner (verbatim, everywhere)
> **Portkey breaks the circuit on errors. OpenLIT watches. Argus closes the loop on behavior.**

_Updated 2026-07-18 (Shafi's call) per **P7** evidence: the old "Portkey guards one call" was factually refutable — Portkey ships a **windowed** circuit breaker. What survives every refutation: **everyone else's loop fires on transport health (HTTP codes); ours fires on behavioral quality while every call returns 200.** See `proofs/P7_competitor_check/matrix.md`._
