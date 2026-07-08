# DEMO_RISK.md — War-game: 50 ways the demo dies (Day-6 exercise)

**Objective: a demo with ≥99% probability of succeeding.**

Assume it's Day 6. Everything is built. Tomorrow we present. Here is everything
that can go wrong, ranked, and how we neutralize it.

---

## The insight: 5 structural commitments kill ~40 of the 50 at once

A good demo isn't hardened risk-by-risk. A few architectural decisions neutralize
whole classes of failure. Make these five and most of the table below disappears.

1. **✈️ AIRPLANE MODE** — the demo runs **100% offline on one laptop**. Nothing on
   the critical path crosses the venue network. *(Kills all WiFi / DNS / firewall /
   bandwidth / OpenAI-availability / captive-portal failures — ~15 items.)*
   → Critical path uses a **deterministic replay LLM** (or local Ollama). The real
   model is an *optional live encore* only if conditions are perfect.

2. **🎬 DETERMINISM** — failures are **injected** (chaos), model responses are
   **replayed from golden fixtures**, recovery grounds against **fixed data**. The
   product *logic* is 100% real; only the *inputs* are deterministic. *(Kills LLM
   stochasticity, bad formats, filtered injections, false-positives — ~10 items.)*

3. **🔥 PRE-WARM & PRE-SEED** — stack booted **30+ min early**; image digests
   **pinned & pre-pulled**; ClickHouse **seeded with history**; anomaly baseline
   **warm**; ports pre-checked; volumes fresh. Dashboards are **never empty**.
   *(Kills startup lag, ingestion lag, empty-dashboard, port conflicts — ~12 items.)*

4. **🎛️ GOD-MODE CONTROL PANEL** — one linear **"Next beat"** driver, decoupled
   from live timing. The **LEARN** beat is driven by a fast `query_range` **poll**,
   **not** the alert scheduler. *(Kills races, alert-timing, wrong-button order — ~8.)*

5. **📹 BACKUP VIDEO + DRESS REHEARSAL ×2** — a recorded **flawless run** to cut to
   instantly, and **two timed rehearsals on the actual laptop + projector with wifi
   off**. *(Backstops ALL 50; kills human error, projector, over-time.)*

> Principle: **the critical path must not depend on anything we don't control in the room.**

---

## The ranked 50

Tier = probability × impact. 🔴 demo-ending & likely · 🟠 high · 🟡 medium · 🟢 low.

| # | Failure mode | Tier | Mitigation |
|---|--------------|------|-----------|
| 1 | Venue WiFi down / flaky | 🔴 | Airplane mode; nothing external on critical path |
| 2 | OpenAI outage / 503 | 🔴 | Replay LLM / local model; real API = encore only |
| 3 | Images not pre-pulled → GB download on venue wifi | 🔴 | Pin digests, pre-pull, `docker save` offline bundle |
| 4 | `compose up` fails on demo laptop (works-on-my-machine) | 🔴 | Rehearse full boot on the *actual* laptop ≥2×; commit known-good compose+.env |
| 5 | SigNoz slow to start → dashboards empty at showtime | 🔴 | Boot stack ≥30 min early; health-gate; never `up` live |
| 6 | ClickHouse ingestion lag → data invisible on cue | 🔴 | Pre-seed history; tune async_insert/flush; stream new data on top |
| 7 | LLM non-determinism → recovery returns a *wrong* answer live | 🔴 | Re-ground against golden fixture; replay; never trust live correctness |
| 8 | Exporter can't reach collector → empty dashboards | 🔴 | Pre-seed; localhost collector; verify E2E in pre-flight |
| 9 | No rehearsal on real hardware/projector | 🔴 | Full dress rehearsal on demo laptop+projector, wifi off, timed, ×2 |
| 10 | OpenAI latency spike → dead air | 🔴 | Offline; hard timeouts; pre-warmed responses |
| 11 | OpenAI rate limit 429 | 🟠 | Offline; if live, dedicated key + low concurrency + backoff |
| 12 | Firewall blocks OTLP 4317/4318 or OpenAI | 🟠 | All localhost; nothing crosses venue net |
| 13 | Captive portal hijacks requests | 🟠 | Disable wifi entirely during run |
| 14 | Model safety-filters the injection payload → beat won't fire | 🟠 | Benign deterministic injection fixture at tool-output layer (chaos), not a real jailbreak |
| 15 | Response format unexpected → agent crashes | 🟠 | Strict parse + fallback; replay guarantees format |
| 16 | API key expired / quota / billing | 🟠 | Offline; verify key+quota pre-flight; backup key |
| 17 | Alert doesn't fire within demo window | 🟠 | Drive LEARN via `query_range` poll (fast, deterministic); show anomaly panel already tripped |
| 18 | Anomaly baseline cold → no anomaly detected | 🟠 | Pre-warm baseline with seeded history |
| 19 | Port conflicts (4317/3301/8080 taken) | 🟠 | Pin ports; `lsof` pre-flight; unique compose project name |
| 20 | Laptop resource exhaustion (stack too heavy) | 🟠 | Cap container resources; close everything; 32GB machine; test headroom |
| 21 | Laptop sleeps / battery dies | 🟠 | Plugged in; disable sleep+screensaver (`caffeinate`) |
| 22 | Clock skew → data lands outside dashboard window | 🟠 | NTP-sync containers to host; relative window; verify timestamps pre-flight |
| 23 | Trace context not propagated → recovery span unlinked | 🟠 | Explicit `trace_id` propagation; assert linkage in a test |
| 24 | Guard false-positive blocks the *correct* answer on stage | 🟠 | Threshold tuned; fixtured inputs; test happy path 20× |
| 25 | MCP server down / auth (investigation beat) | 🟠 | Make MCP beat OPTIONAL; pre-capture output as fallback |
| 26 | Presenter clicks wrong button / order | 🟠 | God-mode linear "Next beat"; scripted runbook; driver+narrator split |
| 27 | Projector crops key panel / font too small | 🟠 | Pre-set resolution+zoom; design for projector; dry-run on it |
| 28 | Over 5 minutes → cut before payoff | 🟠 | Timed rehearsal; payoff front-loaded; per-beat time budget |
| 29 | Docker Desktop update/restart prompt | 🟠 | Disable auto-update; pre-open; freeze all updates demo week |
| 30 | Stale state from prior run pollutes demo | 🟠 | Fresh volumes per run; reset script; unique run-id/time window |
| 31 | Collector misconfig drops spans | 🟡 | Config committed+tested; pre-flight span-count assertion |
| 32 | Metric temporality mismatch → blank panels | 🟡 | Fix delta/cumulative; validate panels render with seeded data |
| 33 | Cardinality explosion mid-demo slows ClickHouse | 🟡 | Enforce low-cardinality labels; load-test before |
| 34 | Chaos toggle doesn't fire / double-fires | 🟡 | Idempotent, debounced injection; visible state; manual re-trigger |
| 35 | Cost-guard / loop-kill fires too early/late | 🟡 | Deterministic thresholds vs fixture; test 10× |
| 36 | Gateway crashes under the injected failure | 🟡 | Guard rails + try/catch; the immune system must not die of the disease; chaos-test it |
| 37 | Timeout too aggressive aborts recovery | 🟡 | Tune timeouts w/ margin; test under demo latency |
| 38 | Idempotency bug → quarantine flaps | 🟡 | Idempotency key + state machine; test re-fire |
| 39 | Race: dashboard queried before data lands | 🟡 | Pre-seed + buffer; god-mode advances only when data present; poll-retry |
| 40 | DNS failures on venue net | 🟡 | Offline/localhost; hosts entries; no external DNS on path |
| 41 | ARM/x86 image mismatch (Apple Silicon) | 🟡 | Multi-arch images; test on exact demo arch |
| 42 | SigNoz version drift from rehearsal | 🟡 | Pin image tag/digest; freeze versions demo week |
| 43 | Query API schema differs from expectation | 🟡 | Pin version; snapshot responses; contract test |
| 44 | Dashboard JSON import fails / blank panels | 🟡 | Import + screenshot-verify pre-flight; commit working JSON |
| 45 | Streaming stalls (huge TTFT) | 🟡 | Non-streaming for demo; or replay; timeout+fallback |
| 46 | VPN interferes with localhost routing | 🟢 | VPN off during demo |
| 47 | ClickHouse disk fills / OOM | 🟢 | Seed modest data; cap retention; monitor disk pre-flight |
| 48 | Bandwidth saturated by other teams | 🟢 | Offline removes the dependence entirely |
| 49 | Human forgets a step / nerves | 🟢 | Printed checklist + runbook; two people; rehearsed |
| 50 | Model deprecated / changed since build | 🟢 | Pin model version; offline replay is immune to provider changes |

---

## T-minus pre-flight checklist (run before you walk on stage)

**Night before**
- [ ] Image digests pinned; `docker save` offline bundle created & test-loaded
- [ ] Full offline dress rehearsal on the demo laptop (wifi OFF), timed < 5:00
- [ ] Backup video recorded of a flawless run; on the laptop + a USB stick + cloud
- [ ] Dashboard JSON imports clean; every panel renders from seeded data (screenshot)
- [ ] Model version pinned; replay fixtures frozen

**T-30 min**
- [ ] `caffeinate` on; sleep/screensaver disabled; laptop plugged in
- [ ] `lsof` ports clear (4317/4318/3301/8080); fresh volumes; unique run-id
- [ ] Full stack `up`; health-gate all containers green
- [ ] Seed script run; open each dashboard; confirm data + correct time window
- [ ] NTP sync verified; timestamps land in the relative window
- [ ] Run the god-mode "Next beat" flow once end-to-end, wifi OFF
- [ ] Projector: resolution + zoom set; key panel readable from the back row

**T-0**
- [ ] Wifi OFF. VPN OFF. Notifications OFF (Do Not Disturb).
- [ ] Backup video one keystroke away.
- [ ] Driver on god-mode; narrator on script. Breathe.

**If anything smells wrong in the first 20 seconds → cut to the video and narrate live. Nobody will know.**
