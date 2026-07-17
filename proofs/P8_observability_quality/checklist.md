# P8 — Observability quality

**Question:** would a SigNoz engineer open our traces and think *"these people understand OpenTelemetry"*?

This is a SigNoz hackathon. Instrumentation *craft* is worth real judging points. Grade ourselves
as if a SigNoz maintainer is reviewing our telemetry.

**STATUS: 🔴 PENDING — checklist WRITTEN (2026-07-17), zero boxes ticked.**
Every box below requires looking at telemetry that does not exist yet. Ticking any of them before
`argusd` emits a real span would be fabricating a proof. The checklist is the *deliverable* of the
pre-hackathon window; the ticks are the deliverable of the build.

> **Pre-hackathon rule (Jul 17→20):** *"coding and design work should begin only after the hackathon
> starts."* Writing this checklist = notes. Ticking it = instrumenting. Do not cross that line.

---

## ⚠️ Read this before writing a single `gen_ai.*` attribute

**The GenAI conventions moved repos six weeks ago.** As of core semconv **v1.42.0 (2026-06-12)**, every
`gen_ai.*` signal was deprecated in `open-telemetry/semantic-conventions` and moved to
**`open-telemetry/semantic-conventions-genai`** (issue #3696). The old path now serves a stub reading
*"This page has moved and is no longer maintained in this repository."*

This matters more than it looks:

- **Every blog post, vendor doc, and LLM's training data points at the old paths.** If we copy a
  tutorial, we emit 2024-era attributes and a maintainer clocks it instantly.
- **The new repo is UNVERSIONED** — zero releases, zero tags, `Schema URL: TODO`. Last push
  2026-07-16 (*yesterday*). We cannot say "we target GenAI semconv v1.43.0" — v1.43.0 doesn't cover
  GenAI anymore.
- **Everything is `Development` stability** — the lowest tier. Zero Stable/RC/Beta. The only Stable
  attributes on a GenAI span are borrowed core ones (`error.type`, `server.address`, `server.port`).

**The sentence to say instead (and to put in the README):**
> *Targets `open-telemetry/semantic-conventions-genai` @ `<git-SHA>`, pinned against core semconv
> v1.43.0. All GenAI conventions are Development-stability and unversioned as of July 2026; we pin a
> SHA and expect breakage.*

Citing the repo split with a pinned SHA proves we read the source in the last six weeks, not a blog.
It is the cheapest credibility we can buy in this proof.

---

## Checklist

### Identity & naming
- [ ] **`gen_ai.provider.name`** used — **NOT `gen_ai.system`**, which was *removed* (renamed in core
      v1.37.0, PR #2046). Zero occurrences remain in the registry. Using it is the #1 stale-copy tell.
- [ ] **`gen_ai.operation.name`** set from the 17-value enum (`chat`, `invoke_agent`, `execute_tool`, …)
- [ ] **Span names** follow the spec formula, not our imagination:
      - inference → `{gen_ai.operation.name} {gen_ai.request.model}` → **`chat gpt-4o`** (not `gpt-4o`, not `handler_1`)
      - tool → **`execute_tool {gen_ai.tool.name}`**
      - agent → **`invoke_agent {gen_ai.agent.name}`**
- [ ] **Span kind** is correct — the #1 agent-instrumentation tell. The discriminator is *whether a
      process boundary is actually crossed*:
      - Go gateway → LLM provider = **`CLIENT`**
      - Python agent, in-process = **`INTERNAL`**
      - `execute_tool` = **`INTERNAL`**
- [ ] **Argus-native spans keep `argus.*` names** (`argus.recovery.reground`) — they are not GenAI
      operations and must NOT be forced into the `gen_ai.*` span taxonomy. See the reconciliation
      note below.

### Tokens & response
- [ ] **`gen_ai.usage.input_tokens` / `.output_tokens`** — NOT `prompt_tokens`/`completion_tokens`
      (renamed in core v1.27.0; ~2 years stale; zero occurrences remain)
- [ ] **No `gen_ai.usage.total_tokens`** — it does not exist in the spec. Emitting it is a tell.
      Derive it at query time.
- [ ] **`gen_ai.response.finish_reasons` is a string ARRAY**, not a scalar (common bug)
- [ ] Billable tokens reported when available (spec: MUST, when both available)

### Metrics
- [ ] **All 12 GenAI metrics are Histograms.** There are no Counters/UpDownCounters in GenAI semconv.
- [ ] **Explicit spec bucket boundaries set** (not SDK defaults) — cheap, visible competence signal:
      - `gen_ai.client.token.usage` `{token}`: `[1,4,16,64,256,1024,4096,16384,65536,262144,1048576,4194304,16777216,67108864]`
      - `gen_ai.client.operation.duration` `s`: `[0.01,0.02,0.04,0.08,0.16,0.32,0.64,1.28,2.56,5.12,10.24,20.48,40.96,81.92]`
- [ ] **`gen_ai.token.type` ∈ {`input`,`output`} ONLY** — NOT `prompt`/`completion`
- [ ] **Zero high-cardinality labels on metrics.** Spec: *"Metric attributes that may have high
      cardinality can only be defined with `Opt-In` level."* Never `conversation.id`, `response.id`,
      user IDs, or prompt text. (This is already CLAUDE.md law — the spec agrees with us.)
- [ ] **Metric temporality** correct (no double-counting; Delta→vanilla-Prometheus silently breaks
      counters). ⚠️ *GenAI semconv does NOT specify temporality* — it's SDK/exporter territory. Do not
      claim the spec mandates it.
- [ ] **`argus_*` metrics** (our own contract, locked in P4) stay `argus_*` — they measure Argus, not
      a GenAI operation. Do not rename them into `gen_ai.*`.

### Content & privacy
- [ ] **Content capture OFF by default** — off at all three layers (spec text, `Opt-In` level,
      `NO_CONTENT` default). Every content attr carries a PII warning.
- [ ] **`gen_ai.input.messages` / `gen_ai.output.messages` / `gen_ai.system_instructions`** used when
      opted in — NOT the flat `gen_ai.prompt.{i}.content` style (that's OpenLLMetry's, **never** official OTel)
- [ ] **JSON-serialized on spans** — structured span attributes are blocked on OTEP #4485; spec says
      serialize to JSON string on spans, structured form on events. *Knowing why is a strong signal.*
- [ ] **The opt-in coupling** understood: `OTEL_INSTRUMENTATION_GENAI_CAPTURE_MESSAGE_CONTENT` +
      `OTEL_SEMCONV_STABILITY_OPT_IN=gen_ai_latest_experimental`. **Without the latter, Python GenAI
      instrumentation pins to `Schemas.V1_26_0`** — the 2024-era span-event design — and
      `get_content_capturing_mode()` raises. Huge, non-obvious gotcha.
      ⚠️ **Env var names verified for PYTHON ONLY.** Go naming unverified → verify at build time.
- [ ] **`gen_ai.conversation.id` never fabricated.** Spec verbatim: *"a new UUID, a trace identifier,
      or a hash of request content SHOULD NOT be used as a fallback value."* Classic tell.

### Resource & errors
- [ ] **Resource attrs set**: `service.name`, `service.version`, `service.instance.id`,
      **`deployment.environment.name`** (⚠️ renamed from `deployment.environment`; now Stable)
- [ ] **No `unknown_service`** anywhere — instant tell
- [ ] **Span status left UNSET on success.** Spec: *"MUST be left unset if the instrumented operation
      has ended without any errors."* Do not set `Ok`.
- [ ] **`error.type` is low-cardinality** — an identifier, not a raw exception message. Success
      metrics SHOULD NOT carry `error.type`.
- [ ] **BatchSpanProcessor**, not SimpleSpanProcessor (blocking export per `span.end()`)

### Argus-specific
- [ ] **Metric → trace → logs** click-through works in SigNoz (exemplars / correlation)
- [ ] **Service boundaries** obvious (gateway ≠ agent ≠ recovery as distinct services)
- [ ] **Service map** tells the story on its own (gateway → LLM → tools → argus)
- [ ] **Exceptions** — bad responses surface as span events/exceptions, browsable in SigNoz
- [ ] **Recovery is trace-linked** (P3 🟢) so one trace narrates the whole incident
- [ ] **No orphan spans** across the Go↔Python boundary

## The Go↔Python seam (our specific risk)
Argus is Go gateway + Python agent. **W3C traceparent must propagate across that boundary** or the
trace fragments. Assert one continuous trace from gateway → agent → tools.

Specific breakages to test for:
- Go: `go func()` without passing the parent `ctx` → fragmented trace
- Go: raw `net/http` not wrapped with `otelhttp` → no propagation
- Python: `ProcessPoolExecutor` / manual threads do **not** inherit context (asyncio `contextvars` do)

## ⚠️ Reconciliation debt from P3 (found 2026-07-17, decide at build time)
P3 went 🟢 with spans named **`agent.request`** and **`argus.recovery.reground`** — chosen before this
semconv review existed. They are not wrong, but they need a *decision*, not a drift:

- `argus.recovery.reground` → **keep**. It's an Argus action, not a GenAI operation. No spec name fits.
- `agent.request` → **probably rename to `invoke_agent {gen_ai.agent.name}`**, which IS a spec
  operation. Leaving it as `agent.request` is the kind of thing a maintainer notices.

This does not invalidate P3 (it proved *parenting*, not naming). Log the decision in DECISIONS.md.

## What SigNoz actually does with this today — the honest answer
**Nothing lights up automatically.** Emitting perfect GenAI semconv gets us generic trace/metric/log
exploration where `gen_ai.*` is filterable like any attribute, plus manually-imported dashboards.

- Issue **#8865 "LLM Observability" is still OPEN** (created 2025-08-20)
- A real `/llm-observability` surface **is** being built — Overview (PR #12145), Model Pricing +
  "Unpriced Models" auto-detecting `gen_ai.request.model` (PR #11813), Attribute Mapping (#11778–11819)
- **PR #12123 is still OPEN**: the feature is gated behind **`enable_ai_observability`, disabled by default**
- SigNoz latest: v0.133.0 (2026-07-15) — note we are pinned to **v0.132** (P0)

**Do NOT claim** "SigNoz renders our GenAI semconv natively." **Claim:**
> *We emit spec-conformant `gen_ai.*`, so it's queryable in SigNoz today and forward-compatible with
> the AI-observability surface landing behind `enable_ai_observability`.*

Accurate, *and* it shows we read their PRs — which is worth more than either.

## Verdict
- All boxes checked → ☐ PASS / ☐ FAIL — **not yet attempted; no telemetry exists**
- A SigNoz maintainer would nod → ☐ yes / ☐ not yet

**The two lines most likely to make a maintainer look twice:** citing the repo split with a pinned
SHA (proves we read source, not blogs), and the `OTEL_SEMCONV_STABILITY_OPT_IN` → `Schemas.V1_26_0`
coupling (proves we read the implementation).

## Flagged uncertain — verify at build time, do not assert
- Env var names **verified for Python only**; Go/JS/Java unverified
- **Temporality is not specified** by GenAI semconv — don't claim the spec mandates it
- Structured span attributes blocked on **OTEP #4485** — in flight
- Streaming-chunks section of the spec is literally `TODO`; so is the production-recommended
  external-content-storage pattern
- Unreleased breaking changes already queued in `changelog.d/`
- `gen_ai.invoke_agent.inference_calls`/`tool_calls` landed **2026-07-16** (PR #336), replacing a
  removed `gen_ai.agent.steps` — *this spec is moving under us daily*

→ update `../SUMMARY.md` (stays 🔴 until boxes are ticked against real telemetry)
