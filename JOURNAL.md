# ENGINEERING JOURNAL

One entry per day. Three questions only. This is **Devpost fuel** and it stops us re-litigating
settled decisions.

**Template — the 5PM four questions (objective, not emotional)**
```
## Day N — YYYY-MM-DD
1. What assumption went 🔴→🟢 today?
2. What stayed 🔴?
3. Did reality disagree with any ADR? → if yes, change the ADR, not the code.
4. Can we still win? (objective)
- Surprises / screenshots:
```

---

## Day 0 — 2026-07-07 (pre-build)
- **Proofs 🔴→🟢:** none yet — all P0–P9 pending. Proof harness scaffolded and runnable.
- **What surprised us:** the flagship "SigNoz blocks the response in real time" was a *physics
  lie* — alert/ingestion latency is orders of magnitude above the inline budget. Forced the
  PREVENT/LEARN split, which turned out to be a *stronger*, more honest architecture.
- **Decision changed by evidence (reasoning, pre-measurement):** PREVENT detector set to a
  deterministic **grounding check** (ADR-0002) so the hot path stays a single Go binary and the
  demo can't flake. Awaiting P1/P8 numbers to confirm the latency budget.

## Day 1 — 2026-07-07 (P0 baseline probe)
1. **🔴→🟢 today:** environment baseline — Docker 28.4.0 / Compose v2.39.2 / daemon UP / git / python 3.13.7 / 32 GB RAM / 115 GB free. Machine exceeds requirements.
2. **Stayed 🔴:** P0 not yet PASS — stack not up, offline not verified. P1–P9 untouched.
3. **ADR disagree?** No. Port conflicts are an environment issue, not architecture. ADRs intact.
4. **Can we still win?** Yes — only obstacle is two local dev-server port conflicts (8080 Python, 9000 node), both trivially resolved.
- **Surprise:** 9000 already taken by a node dev server — but SigNoz's ClickHouse `9000` is container-internal (not host-published), so it was never a real conflict. Only our webhook sidecar was → remapped 9000→9010 with `WEBHOOK_PORT`.
- **Bigger surprise (real P0 catch):** SigNoz **deprecated the bundled docker-compose** (`deploy/docker/`) ~v0.130 — install is now via **Foundry** (`foundryctl`). Our P0 runbook was invalid. Corrected via `foundryctl gen examples` → real compose. Verified host ports: OTLP 4317/4318, UI 8080 (→8081 locally), MCP 8000; Postgres/ClickHouse internal. Exactly the assumption that would've detonated on Day 6 — caught Day 1 for the cost of an edit. No ADR affected (install mechanism, not architecture).
- **P0 RESULT → 🟢\*:** Foundry-generated compose UP; all 6 containers healthy; `verify_offline.sh` PASS (OTLP 4318=200, UI 8081=200). Marker span emitted & accepted (ingestion works, no auth). Admin provisioned via `/api/v1/register`.
- **Readiness learning:** `signoz-0` reports *healthy* ~2.5 min BEFORE the OTLP ingester accepts data (collector gated behind first-boot schema migrations). **True readiness = OTLP accepting a POST, not UI health.** Feeds DEMO_RISK (boot early) + P9 (judges must wait for OTLP, not the UI).
- **Open (not architecture):** P1 query-half needs a PAT; v0.132 has no scriptable JSON login (all `/login` paths serve the SPA) and no OSS PAT-create route found → PAT is a 30-sec UI step. Wifi-off toggle also pending to close P0's offline clause.

## Day 2 — 2026-07-08/09 (stack up, repo, offline)
1. **P0 → 🟢:** SigNoz (Foundry v0.132) boots/serves/ingests; serves via internal container DNS (`{"status":"ok"}` container-to-container); live OTLP ingest 200.
2. **Repo:** github.com/shafi-VM/argus (private) — initial commit pushed (35 files, clean).
3. **Reality check:** this dev box is behind NAT (`en0` stays up when "wifi off"), so the ping-based offline heuristic can't confirm isolation. And **Claude is cloud-hosted → can't operate during a true-offline run.** ⇒ Definitive airplane-mode proof = 60s HUMAN step on the demo laptop; not worth tearing down the running stack we need for P1.
4. **Still red:** P1+ blocked on a SigNoz PAT (30s UI step).

## Day 3 — 2026-07-09 (P1 GREEN — the big de-risk)
1. **🔴→🟢 P1 done.** A1 query RTT **~30ms** (bar <2s); A4 ingestion lag **median ~4.7s / max 5.4s** (bar <10s). **LEARN loop is latency-viable** — the whole "close the loop through SigNoz" thesis holds.
2. **Auth cracked:** v0.132 = service-account keys via `SIGNOZ-API-KEY` header + the account **needs a role** (`signoz-viewer`+); missing role → 403.
3. **Query-API churn:** v4 list needs `aggregateOperator:noop`+`selectColumns`; v5 exists. Measured lag via **ClickHouse directly** (robust) rather than fighting the JSON — will wire the exact query when building the real LEARN poller.
4. **ADR check:** confirms **ADR-0003** — PREVENT can't wait ~5s (stays inline), LEARN tolerates it. Nothing changed.
- **Repo:** github.com/shafi-VM/argus (private) — P0+P1 milestone committed & pushed (`d12e322`).

---

## Day 4 — 2026-07-17 (competitor recon — we refuted ourselves)
1. **🔴→🟡 P7 (not 🟢).** 11 tools researched from **source code, licenses, issues, maintainer threads** —
   not marketing. Held at 🟡 on purpose: P7's bar says *"actually try each, first-hand"* and we didn't
   run them. **P8 checklist written, zero boxes ticked** (nothing to check against yet). No bar moved.
2. **Stayed 🔴:** P5, P6, P9 — and they're not "pending," they're **unreachable**: all three measure a
   product that the hackathon rules forbid us from building until **Jul 20**. Same for A2/A8/A10.
   **A8 was nearly started today by mistake** — caught by re-reading the rules page. *"Coding and
   design work should begin only after the hackathon starts."* A disqualified project retires no
   assumptions at all.
3. **Did reality disagree?** — **YES, and this is the day's real finding. Reality disagreed with the
   frozen one-liner.** Both halves of *"Portkey guards one call. OpenLIT watches"* are **factually
   refutable from public sources**:
   - **Portkey** ships a **circuit breaker** — `minimum_requests: 10`, `failure_threshold_percentage`,
     `cooldown_interval` → auto-removes targets from routing. **That is windowed→auto-action: the
     loop we claimed to own.**
   - **OpenLIT v1.44.0** ships **inline blocking guards** (`setup_auto_guards()`, regex, sub-ms,
     preflight raise + postflight deny/redact). It does **not** merely watch.
   - **LiteLLM** (not even on our radar) auto-cools-down deployments on a **rolling 1-min failure
     window**. A reviewer produces this in ~30 seconds.
   → Per our own rule (*change the ADR, not the code*), **the line must change — flagged for Shafi,
   not edited unilaterally** (VISION/TERMINOLOGY are frozen; that's a human call).
4. **Can we still win? Yes — and the pitch got *more* honest, which is the same thing here.** The
   wedge survived every refutation attempt, just **narrower and far better defended**: *every* one of
   those loops runs on **transport health — HTTP codes, latency, rate limits. None observes response
   CONTENT over a window and acts.* Portkey's breaker fires on 5xx. Ours fires while **every call
   returns 200 OK** and the answers are quietly getting worse. **That is literally the sentence
   VISION.md opens with** — *"infrastructure can be perfectly healthy while AI behavior is
   catastrophically wrong"* — and we now have 11 tools' worth of source-read evidence that the
   industry has built breakers for the healthy half and nothing for the wrong half.

- **Surprises:**
  - **WhyLabs has both halves and never joins them** — real inline response blocking **and** real
    windowed drift (Hellinger/KL/JS/PSI)… computed on the SaaS control plane, never fed back to the
    guardrail. The closest anyone got. (They're also discontinuing — Apple acquisition.)
  - **Portkey has no native groundedness check.** Its own plugin does moderation/PII/gibberish;
    groundedness is outsourced to paid third-party APIs — `patronus/retrievalHallucination.ts`
    defaults to a **15,000 ms timeout** on the hot path. **ADR-0002's deterministic <50ms Grounding
    Check is a real differentiator, not just a demo-safety choice.** Best news of the day.
  - **The threat we could NOT close:** Portkey guardrail denial returns **446**, and
    `failure_status_codes` takes arbitrary codes → `[446]` would plausibly assemble our thesis from
    Portkey parts. Undocumented, **impossible in the OSS gateway** (`c.set()` for breaker state exists
    nowhere in the repo — the accounting is in their closed cloud), and it counts **binary denials,
    not drift**. Logged in the matrix rather than hidden. If asked, concede it.
  - **"Gateway 2.0 / P99 breaking / probe-based recovery" appears to be AI-slop** from aggregator
    sites — HEAD is v1.15.2, no `probe`, no `p99`. Two researchers flagged it independently. Don't
    fear it; don't cite it either.
  - **OTel GenAI semconv moved repos six weeks ago** (core v1.42.0 → `semantic-conventions-genai`,
    **zero releases, unversioned**), and **`gen_ai.system` was removed** → `gen_ai.provider.name`.
    Every blog and most model training data is stale. **P3's `agent.request` span is now semconv debt**
    (`invoke_agent {agent}` is a real spec name) — logged in P8, decide at build time. Doesn't
    invalidate P3: it proved *parenting*, not naming.
  - **SigNoz has no GenAI view yet** — issue #8865 still OPEN; `/llm-observability` is being built
    behind **`enable_ai_observability`, disabled by default** (PR #12123, still open). So: *"queryable
    today, forward-compatible with the surface landing behind their flag"* — accurate, and it shows we
    read their PRs.
  - **`ASSUMPTIONS.md` — "the engineering bible" — was the last file to hear the news.** It still read
    all-🔴 while five proofs had been green for a **week**. Resynced; added a sync rule. A stale bible
    invites re-proving what's proven and hides what isn't.
  - **We are 2 days from missing the Pre-Event blog track** (deadline **Jul 19**, AirPods + SigNoz
    interviews for top blogs). It's *writing* → rule-legal now. We're sitting on the material: the
    physics lie, the Foundry deprecation, the healthy-but-not-ingesting race. **Nobody has started it.**
- **P3 → 🟢:** recovery span (`argus.recovery.reground`) lands as a **child** of `agent.request` in the SAME trace (`parent_span_id` matches `span_id`). "Postmortem writes itself" mechanic works — one waterfall in SigNoz. Ingestion ~1.5s this run (fast end of P1 range).
- **P4 → 🟢 (hero dashboard as-code):** built "Argus — Intelligence Health" (8 panels) entirely via the SigNoz API — no manual clicking. All panels return live data (gauges avg/avg; counters rate/sum). `argus_*` metric contract locked & flowing; clean `dashboard.json` vendored (importable, P9). **v5 query schema fully cracked** — `/api/v5/query_range` (~85ms) is the LEARN read path (closes ADR-0003's poll question, P1 follow-up).
- **P2 → 🟢 (decided by evidence):** LEARN polls query_range (~30ms + ~5s freshness); alerts = optional demo visual, not the action path.
- **🚪 GATE:** P0–P4 all green → **cleared to build the actual Argus product.** ⚠️ *Corrected 2026-07-18: cleared in principle, but **rule-locked until Jul 20** — no product code before the hackathon starts.*

## Day 4 — 2026-07-18 (Shafi + Claude: one-liner locked, demo script drafted)
- **One-liner LOCKED** (Shafi's call, per **P7** evidence): *"Portkey breaks the circuit on errors. OpenLIT watches. Argus closes the loop on behavior."* Updated in VISION / TERMINOLOGY / README / CLAUDE with a dated rationale note. The frozen text changed because **reality disagreed** (Portkey's breaker is windowed) — the process working, not failing.
- **`DEMO_SCRIPT.md` drafted** — 5-min storyboard, a wow every ~30 s, beats tagged PROVEN vs TO-BUILD, P7 objection-proofing baked in. Written notes only → rule-legal.
- **⚠️ URGENT (Abhishek flagged Jul 17): the Pre-Event blog track deadline is Jul 19 — TOMORROW.** Writing → rule-legal now; material is sitting ready (physics lie, Foundry deprecation, healthy-but-not-ingesting race); nobody has started it. Highest-urgency legal work left in the pre-hackathon window.

## Hackathon Day 1 — 2026-07-20 (build UNLOCKED)
- **🟢 Day-1 GATE (issues #2–#4): the spine is real.** `argusd` (Go, OTel-instrumented) + Ada (Python agent) + a deterministic mock LLM → **one linked cross-service trace** in SigNoz: `invoke_agent ada` (ada-agent, root) → `chat gpt-4o` (argusd, child), verified via `parent_span_id`. W3C traceparent crosses the Python→Go boundary cleanly; **no orphans**.
- **First real product telemetry** — ticks P8: `gen_ai.provider.name` (not the removed `gen_ai.system`), `invoke_agent {name}` + `chat {model}` span names, CLIENT span kind, cross-boundary propagation.
- **The mock LLM is deliberate, not a shortcut:** a deterministic upstream IS the demo's airplane-mode replay engine (DEMO_RISK). The real LLM key plugs in for grounding on Day 2. Chaos will make it return a wrong answer on cue.
- **Architecture note:** argusd is a drop-in OpenAI-compatible proxy (agent points `base_url` at it) — framework-agnostic by construction, which is exactly the "works with any OTel-emitting agent" claim.
- **Next (Day 2):** PREVENT — inline grounding check → block → re-ground → recover. The money moment.

## Hackathon Day 2 — 2026-07-21 (PREVENT works)
- **🟢 Day-2 GATE (#5 #6 #7 #8): the money moment is REAL.** With the mock lying on cue (chaos=`hallucinated` → cites `UA99`, absent from the retrieved context), Ada's caller still received the grounded answer (`AA42`). **The user never saw the hallucination.**
- **Grounding Check measured at 0.138 ms** — budget was <50 ms, so **~360× under**. Portkey outsources the same groundedness check to a third-party API with a **15,000 ms** default timeout. That gap is now a *measured number*, not a claim. Lead with it.
- **One waterfall, three levels:** `invoke_agent ada` (ada-agent) → `chat gpt-4o` (argusd, `decision=recovered`) → `argus.recovery.reground` (argusd). The incident narrates itself.
- **Applied my own PR-review note on #22:** argusd grounds against the **in-band `RETRIEVED_CONTEXT` in the request**, never `fixtures/booking.json`. Corrected the misleading fixture comment. This is what keeps "drop-in proxy, works with any agent" honest instead of a demo trick.
- **Fail-OPEN by design:** no context → skip the check, never block. A false block (refusing a *correct* answer) on stage is the one failure that would sink the demo. Unit-tested.
- **Honest scope:** this is ENTITY-PRESENCE grounding (catches "cited something not in context"), which is exactly ADR-0002's stated non-goal boundary — not general hallucination detection. Say it that way on stage.
- **Surprise:** Docker died overnight (machine restart) → the first e2e ran with **no telemetry**. Behavior was provable; telemetry wasn't. Did **not** call it green until the stack was back and the trace captured. Restart→OTLP-ready took ~9 s (vs ~2.5 min cold boot — migrations already done).
- **Next (Day 3):** LEARN — windowed drift → quarantine/reroute, plus real `argus_*` metrics including the Intelligence Health composite defined in #21.
