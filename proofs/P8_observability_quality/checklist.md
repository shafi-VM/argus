# P8 — Observability quality

**Question:** would a SigNoz engineer open our traces and think *"these people understand OpenTelemetry"*?

This is a SigNoz hackathon. Instrumentation *craft* is worth real judging points. Grade ourselves
as if a SigNoz maintainer is reviewing our telemetry.

**STATUS: 🟢 PASS (verified live against SigNoz v0.134, 2026-07-24).**
Every emit-side box ticked against real spans/metrics (after fixing two tells: missing resource attrs;
Ada Simple→Batch). Live run then confirmed the **service map** (`ada-agent → argusd`, real cross-service
trace) and resolved the last item honestly: **metric→trace exemplars are not stored by SigNoz v0.134**
(no exemplar table, no `trace_id` on samples — verified in ClickHouse), so that is a documented platform
limitation, not a gap in our telemetry. A SigNoz maintainer reading these traces would nod.

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

## Grading — 2026-07-23 (#16), against REAL emitted telemetry

Method: drove every gateway decision path (pass / recovered / refused / upstream_error) through
an in-memory OTel `SpanRecorder` + `ManualReader` and asserted the actual spans, attributes, and
metrics; verified the resource by constructing it and reading its attributes. **This is emitted
telemetry, not code-reading — and it is now committed and regression-guarded**, not a throwaway:
- `internal/gateway/telemetry_semconv_test.go` — span names/kinds, `gen_ai.*` attrs, tokens (input/output,
  no total), status codes, `error.type`, and the low-cardinality metric-label sets.
- `internal/telemetry/resource_test.go` — the complete resource + shared `service.instance.id`.
Break any of these tomorrow (e.g. the token mapping) and the tick goes red with the test.

- **✅ ticked** = proven from emitted telemetry on this machine.
- **🖥️ demo-box** = correct on the emit side, but the *confirmation* is a SigNoz-UI view (service
  map, metric→trace click-through) that needs the live demo stack. **Not fake-ticked.**
- **➖ N/A** = does not apply to what argusd emits by design (we emit `argus_*` behavioral metrics,
  not `gen_ai.client.*` histograms; we capture no message content).

### Identity & naming
- [x] **`gen_ai.provider.name = openai`** used — **NOT** the removed `gen_ai.system` (grepped: zero occurrences).
- [x] **`gen_ai.operation.name`** from the enum — `chat` (gateway), `invoke_agent` (Ada).
- [x] **Span names** follow the spec formula: **`chat gpt-4o`** and **`invoke_agent ada`**.
      ➖ tool span (`execute_tool {tool}`) N/A — the demo agent makes no tool calls, so no tool spans exist.
- [x] **Span kind** correct: gateway→LLM = **`CLIENT`**, `argus.recovery.reground` = **`INTERNAL`**,
      Ada in-process = **`INTERNAL`**. (execute_tool N/A.)
- [x] **Argus-native spans keep `argus.*` names** — `argus.recovery.reground`, not forced into `gen_ai.*`.

### Tokens & response
- [x] **`gen_ai.usage.input_tokens` / `.output_tokens`** — confirmed in the dump; NOT prompt/completion.
- [x] **No `gen_ai.usage.total_tokens`** — confirmed absent from span attrs (mock returns it on the wire; gateway drops it).
- [x] Billable tokens reported when the usage block is present (input + output both emitted).
- ➖ **`gen_ai.response.finish_reasons`** — argusd emits no response.* attrs. Absence of an optional attr
      is not a tell; if we later add it, it MUST be a string array. Left as a build-time note.

### Metrics
- [x] **Zero high-cardinality labels.** `argus_requests_total{model,decision,status_class}`,
      `argus_grounded_total{model}`, `argus_intelligence_health_ratio{}` — all low-cardinality; no
      prompt/user/trace/request id on any metric. (CLAUDE.md law; verified in the dump.)
- [x] **Instrument types correct** — the two `argus_*_total` are monotonic **Sums** (counters),
      Intelligence Health is an **observable Gauge**. No double-counting; no counter-as-gauge.
      ⚠️ **`model` label is caller-supplied** (the request's `model` field) — bounded by our
      primary/fallback convention (`gpt-4o`/`gpt-4o-mini`), so low-cardinality in practice. A
      malicious/misconfigured caller could inflate it; acceptable for the demo, worth an allowlist later.
- [x] **`argus_*` metrics stay `argus_*`** — behavioral metrics, not renamed into `gen_ai.*`.
- ➖ **GenAI client metrics** (12 histograms, spec buckets, `gen_ai.token.type`) N/A **by design**:
      argusd emits *behavioral* metrics about Argus decisions, not raw LLM-client token/duration
      histograms. Those would come from instrumenting the LLM client — a possible enhancement, not
      required for "do they get OTel." Flagged, not faked.
- ⚠️ **Temporality**: cumulative Sums (OTLP default) — fine for SigNoz/ClickHouse. GenAI semconv does
      not specify temporality; we do not claim it does.

### Content & privacy
- [x] **Content capture OFF** — argusd/Ada emit **no** message-content attrs at all (verified absent).
      Compliant with the spec default (`NO_CONTENT`) by construction.
- [x] **No `gen_ai.conversation.id` fabricated** — we don't emit it, so no UUID/trace-id/hash fallback tell.
- ➖ **`gen_ai.input.messages` / opt-in coupling** N/A — no content-capture path exists (off by default is
      the whole point). If one is ever added, use `input.messages`/`output.messages` (not the OpenLLMetry
      flat style), JSON-serialized on spans (OTEP #4485), gated on
      `OTEL_INSTRUMENTATION_GENAI_CAPTURE_MESSAGE_CONTENT` + `OTEL_SEMCONV_STABILITY_OPT_IN=gen_ai_latest_experimental`.

### Resource & errors
- [x] **Resource attrs complete** — `service.name`, `service.version`, **`service.instance.id`** (128-bit,
      shared by trace+metric providers), **`deployment.environment.name=demo`**. *Fixed in this PR — the
      last two were missing.* Verified by constructing the resource and reading its attributes.
- [x] **No `unknown_service`** — `service.name` set to `argusd` / `ada-agent`.
- [x] **Span status UNSET on success** — `pass` and `recovered` spans have status `Unset` (not `Ok`).
- [x] **`error.type` low-cardinality** — `upstream_5xx` / `upstream_4xx` (a class, not a message) on the
      error span; success spans carry none.
- [x] **BatchSpanProcessor** — Go uses `WithBatcher`; **Ada switched Simple→Batch in this PR** (it
      `force_flush`es before exit, so batching loses nothing and matches production shape).

### Argus-specific
- [x] **Recovery is trace-linked** (P3 🟢, re-confirmed live) — `argus.recovery.reground` is a child span
      in the same trace; one waterfall narrates the incident.
- [x] **Behavioral events surface as span events** — a blocked answer emits `argus.behavior.blocked`
      (with the unsupported entities); transport failures `RecordError`; upstream 5xx set status + `error.type`.
- [x] **Service boundaries obvious** — two services (`ada-agent`, `argusd`); recovery is correctly a
      *child span within* argusd, not a bogus third service.
- [x] **No orphan spans across Go↔Python** — traceparent injected by Ada, extracted by argusd; verified
      end-to-end in the live smoke (one linked trace, correct parenting).
- ➖ **Metric → trace click-through (exemplars)** — **NOT achievable on SigNoz v0.134, verified live
      2026-07-24.** argusd now emits exemplar-capable measurements (request `ctx` threaded into `Record`),
      but SigNoz v0.134 **stores no metric exemplars** — there is no exemplar table in `signoz_metrics`
      and `samples_v4` has no `trace_id` column (checked directly in ClickHouse). So exemplar-based
      click-through cannot render regardless of what we emit — a platform limitation, not an argusd gap.
      The available correlation path is attribute+time (filter traces by the same `model`/`decision`),
      not true exemplars. Reclassified from "pending UI tick" to documented limitation.
- [x] **Service map tells the story** — **data confirmed live 2026-07-24:** two services in SigNoz,
      `ada-agent` (invoke_agent) → `argusd` (chat) as a linked cross-service trace; recovery is an
      INTERNAL span *inside* argusd, not a third node. ⚠️ Demo note: the map edge needs traffic driven
      **through Ada**, not raw `curl` to argusd (curl calls have no `ada-agent` parent) — runbook item.

## The Go↔Python seam (our specific risk)
Argus is Go gateway + Python agent. **W3C traceparent must propagate across that boundary** or the
trace fragments. Assert one continuous trace from gateway → agent → tools.

Specific breakages to test for:
- Go: `go func()` without passing the parent `ctx` → fragmented trace
- Go: raw `net/http` not wrapped with `otelhttp` → no propagation
- Python: `ProcessPoolExecutor` / manual threads do **not** inherit context (asyncio `contextvars` do)

## ✅ Reconciliation debt from P3 — RESOLVED (2026-07-23)
The Day-1 spine already fixed the naming flagged here: Ada emits **`invoke_agent ada`** (a real spec
operation) and the gateway emits **`chat gpt-4o`**; the old `agent.request` name is gone.
`argus.recovery.reground` was kept (an Argus action, no spec name fits). No debt remaining.

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

## Verdict — 🟡 emit-side PASS; two UI confirmations pending the demo box
- **Every emit-side box is ✅** against real telemetry, after fixing the two tells found (missing
  `service.instance.id` + `deployment.environment.name`; Ada `Simple`→`Batch`). N/A items are N/A
  **by design** and stated as such.
- **Two boxes stay open (🖥️)** — metric→trace click-through and the service map are SigNoz-UI
  confirmations, not emittable facts; they need the live demo stack. **We did not fake-tick them.**
- **So P8 is 🟡, not 🟢.** It goes 🟢 the first time we open the running SigNoz on the demo laptop and
  confirm those two views (a ~5-minute step during the Day-6 install / rehearsal). A SigNoz engineer
  reading our *spans* today would nod: correct semconv, complete resource, honest scope.

## Evidence — captured spans + metrics (2026-07-23, in-memory OTel recorder)
```
SPAN "chat gpt-4o"             kind=client   status=Unset   decision=pass
     gen_ai.provider.name=openai  gen_ai.operation.name=chat  gen_ai.request.model=gpt-4o
     gen_ai.usage.input_tokens=9  gen_ai.usage.output_tokens=5   (no total_tokens)
SPAN "argus.recovery.reground" kind=internal status=Unset   recovery.grounded=true
SPAN "chat gpt-4o"             kind=client   status=Error   decision=upstream_error
     error.type=upstream_5xx  argus.upstream.status=500
METRIC argus_requests_total   {model,decision,status_class}   (low-cardinality)
METRIC argus_grounded_total   {model}
METRIC argus_intelligence_health_ratio  (observable gauge)
RESOURCE  service.name=argusd  service.version=0.1.0
          service.instance.id=<128-bit>  deployment.environment.name=demo
```

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
