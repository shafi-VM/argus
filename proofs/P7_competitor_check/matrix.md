# P7 — Competitor check

**Question:** can Portkey / OpenLIT / Langfuse / LangSmith / Helicone already do this?

**STATUS: 🟡 PARTIAL — desk research done 2026-07-17, hands-on NOT done.**

> **Read this before quoting anything below.** The pass bar for this proof is *"Actually try each —
> first-hand, not docs."* **We did not.** Every cell below is from vendor docs, **source code**,
> licenses, GitHub issues/PRs, and maintainer threads — not from running the tools against a
> hallucinating agent. That is stronger than marketing copy and weaker than the bar.
>
> **P7 goes 🟢 only when someone runs the bake-off in `## Hands-on TODO` below.** Until then, if a
> judge asks *"did you try Portkey?"* the honest answer is **"we read its source; we didn't run it"** —
> and that answer is fine. A fabricated "yes" is not.

**Verification legend:** `[src]` = read the actual source · `[doc]` = vendor docs · `[lic]` = license
file · `[iss]` = issue/PR/maintainer statement · `[?]` = uncertain, verify before asserting publicly

---

## ⛔ The headline: our claim was WRONG, and the fix makes it stronger

We claimed: *"no single competitor closes the loop from detection to action."*

**That is refutable in thirty seconds.** Three tools already take a **windowed** signal and
**automatically act on live traffic with no human**:

- **LiteLLM** — router counts failures per deployment in a **rolling 1-minute window**
  (`allowed_fails`, default 3; >50% failures in the current minute) → **automatically removes the
  deployment from the routing pool** for `cooldown_time`. That is quarantine + reroute, no human. `[doc]`
- **Portkey** — circuit breaker: `{failure_threshold_percentage: 20, minimum_requests: 10,
  cooldown_interval: 60000, failure_status_codes: [401,429,500]}` → unhealthy targets
  **automatically removed from routing**. `minimum_requests` is unambiguous cross-request accounting. `[doc]`
- **Helicone** — gateway fallback on windowed error rate `[?]` *(marketing copy only; the
  [error-handling docs](https://docs.helicone.ai/gateway/concepts/error-handling) document only
  billing-method fallback — treat the "10%" threshold as unverified)*

**But every one of those loops runs on TRANSPORT health** — HTTP status codes, latency, rate limits.
**None of them observes response *content* over a window and acts.**

That distinction is not a technicality. It is *exactly* the thesis in `VISION.md`:

> *AI infrastructure can be perfectly healthy while AI behavior is catastrophically wrong.*

Portkey's breaker fires on 5xx. Argus's fires while every call returns **200 OK** and the answers are
quietly getting worse. **A circuit breaker on HTTP codes is blind to precisely the failure we exist for.**

### The claim that survived every refutation attempt
> **Nobody closes the loop from a windowed CONTENT/SEMANTIC signal to an automatic action on live traffic.**

Ship that version. Do not ship the loose one.

---

## The matrix

**Columns:** **Detect** (see it) · **Prevent** (block inline, on *content*) · **Recover** (fix, no human) ·
**Investigate** (RCA) · **OSS** (open + self-host) · **🔁 Loop** (windowed signal → automatic action on live traffic)

| Tool | Detect | Prevent | Recover | Investigate | OSS | 🔁 Loop | The one-line truth |
|------|:------:|:-------:|:-------:|:-----------:|:---:|:------:|--------------------|
| **OpenLIT** | ✅ | ⚠️ | ❌ | ⚠️ | ✅ | ❌ | Regex guards block PII/injection inline — **never hallucination**. Detection and enforcement layers **are not connected**. `[src]` |
| **Langfuse** | ✅ | ❌ | ❌ | ✅ | ✅ | ❌ | Declined to enter the request path **on the record**. Hands you an event; **you** build the actuator. `[iss]` |
| **LangSmith** | ✅ | ⚠️ | ❌ | ✅ | ❌ | ❌ | Gateway blocks **spend/PII only, request-side**. Engine drafts PRs — *"you review and merge."* `[doc]` |
| **Helicone** | ⚠️ | ❌ | ⚠️ | ⚠️ | ✅ | ⚠️ | **Input-only** enforcement; its own docs concede output *"appears to bypass security inspection."* **Acquired, maintenance mode.** `[doc]` |
| **Portkey** | ⚠️ | ✅ | ⚠️ | ✅ | ⚠️ | ⚠️ | **Closest.** Real inline block (HTTP 446) + real breaker — but the breaker eats **HTTP codes**, not quality. `[src]` |
| **LiteLLM** | ⚠️ | ❌ | ✅ | ❌ | ✅ | ⚠️ | **The refuter.** Rolling-1-min failure window → auto-cooldown. Transport health only. `[doc]` |
| **Guardrails AI** | ⚠️ | ✅ | ⚠️ | ❌ | ✅ | ❌ | Real inline output validation + `reask` — but **stateless, per-request**. No window, no fallback model. `[doc]` |
| **NeMo Guardrails** | ⚠️ | ✅ | ❌ | ❌ | ✅ | ❌ | Inline output rails → canned refusal. Maintainer: rails *"inspect messages one at a time."* `[iss]` |
| **Arize Phoenix** | ✅ | ❌ | ❌ | ✅ | ❌ | ❌ | Own docs: *"Phoenix doesn't block requests — your app does."* **Drift was DELETED**; **Elastic License 2.0**. `[src]` |
| **W&B Weave** | ✅ | ❌ | ❌ | ✅ | ⚠️ | ❌ | Docs contradict marketing (below). App calls `.apply_scorer()`; **you** write the fallback. `[doc]` |
| **WhyLabs** | ✅ | ✅ | ❌ | ✅ | ⚠️ | ❌ | **Closest to our thesis.** Has **both halves and never joins them** — drift lives on the SaaS control plane and never feeds the guardrail. **Discontinuing.** `[src]` |
| **Argus** | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | **The only row where 🔁 runs on content, not HTTP codes.** ← *unbuilt; this row is a claim, not a result* |

⚠️ = partial/narrow. See per-tool detail below. **The `Argus` row is aspirational until the product
exists.** It is in the table to show the target shape, not to claim a result.

---

## Per-tool detail (the parts that matter)

### Portkey — the closest competitor, and the one that can hurt us
- **Refutes "guards one call" literally.** The circuit breaker does windowed→auto-action. Say
  *"Portkey guards one call"* on stage and a judge with the doc page open wins. `[doc]`
- **No native groundedness.** Portkey's own plugin does `moderateContent, language, pii, gibberish`.
  Groundedness is **outsourced to third-party paid APIs**; `plugins/patronus/retrievalHallucination.ts`
  defaults to a **15,000 ms timeout** on the critical path. **Our deterministic <50ms Grounding Check
  is a genuine technical differentiator — lead with it.** `[src]`
- **Guardrails default to log-only** — `async` defaults `true`, `deny` defaults `false`. The advertised
  "50+ guardrails" ship **non-blocking**. `[src]`
- **`on_fail` can only emit `feedback: {value, weight}`** — a logged score. It cannot mutate the
  response. Retry = same prompt, reroll. Fallback = different model, hope. **Nothing re-grounds.** `[src]`
- **Only evaluates the last message in the request body** — documented limitation, bad for multi-turn. `[doc]`
- **The OSS gateway can't run the breaker at all.** `handleCircuitBreakerResponse` is read via
  `c.get(...)` but **never `c.set(...)` anywhere in the repo**; `isOpen` is only ever read, never
  written; `cb_config` isn't in the OSS config validator. **The accounting lives in Portkey's closed
  control plane.** Self-hosters get the seam, not the engine. `[src]`
- **⚠️ The threat we could not fully close:** guardrail denial returns **446**, and
  `failure_status_codes` accepts arbitrary non-5xx codes (official example uses 401/429). The
  post-guardrail 446 **does** reach the breaker (`responseHandler` emits it in `tryPost`
  → returned as `mappedResponse` → breaker called on that response). So `failure_status_codes: [446]`
  **would plausibly assemble our thesis from Portkey parts.** Why it still doesn't land: (1)
  undocumented and untestable — closed control plane; (2) **impossible in OSS** (see above); (3) it
  counts **binary denials**, not drift — it cannot act on a declining quality score while every call
  returns 200, which is exactly LEARN's case. **Do not pretend this doesn't exist.** If asked, concede
  it and explain (3). `[src]` `[?]`
- **Do not fear "Gateway 2.0."** Claims of P99-latency breaking / probe-based recovery trace to
  **AI-generated aggregator sites** and are likely fabricated. HEAD is v1.15.2 — no `probe`, no `p99`,
  no half-open state. Two researchers flagged this independently. **Don't cite it either.** `[src]`

### OpenLIT — "watches" is FALSE as of v1.44.0
- Ships **genuine inline enforcement**: `setup_auto_guards()` does a second `wrapt` pass over
  OpenAI/Anthropic/Groq/Mistral/Cohere. Preflight `DENY` raises **before** the API call; postflight
  `DENY`/`REDACT` fires **before the response returns to the user**. Pure regex → **sub-ms, no extra
  LLM call**. Not offline analysis. `[src]`
- **But it cannot block a hallucination.** Guards are injection/PII/sensitive-topic/moderation/schema
  only. The 11 eval types that *do* catch hallucination are **not available as guards** — they run
  offline (the module is literally named `offline.py`) or on cron. `[src]`
- **Its alerts cannot fire on LLM behavior at all.** `AlertTriggerType` is exclusively **config-plane**:
  `access_update`, `invite`, `prompt_version_update`, `vault_secret_change`, `context_change`,
  `rule_engine_change`. **There is no trace/metric/eval-driven alert evaluator.** An eval scoring a
  hallucination into ClickHouse **triggers nothing.** `[src]`
- **No feedback path backend→guard.** Guards are statically configured at `openlit.init(guards=[...])`.
  A backend signal can never install or tighten one. `[src]`
- **Postflight guards are silently skipped when streaming** (extractors can't reassemble chunks) and
  **`fail_open=True` by default** — a throwing guard resolves to *allow*. `[src]`
- **The "Rule Engine" is retrieval, not enforcement** — *"When a rule matches, it returns references to
  linked entities."* `grep -iE "action|block|deny|route" sdk/go/rule_engine.go` → **zero hits**. `[src]`
- **Docs drift cuts against us:** `guardrails.mdx` documents an older LLM-based `.detect()` API; the
  shipped code is regex `evaluate()` + auto-guards, and the public guardrails URL **404s**. **Cite the
  code, not the docs** — if we claim guards are slow LLM calls, the source contradicts us. `[src]`
- **SigNoz relation:** integration only — SigNoz is one of ~20 destinations. **No acquisition** (searched
  specifically). Quietly **semi-competitive**: OpenLIT's own backend is ClickHouse-based with its own
  dashboards — architecturally a mini-SigNoz for LLMs — while also supporting SigNoz as a destination.
  **Argus's slot — the action layer on top of SigNoz's OTel data — is one neither product fills.** `[doc]`

### Langfuse — passive, and says so itself
- Own security page: Langfuse is for *"the **ex-post evaluation**"* of guardrails; it points at
  **third-party** libs (LLM Guard, Prompt Armor, NeMo) for actual blocking. `[doc]`
- **Declined the request path on the record.** Discussion #4939 asked for proxy mode; maintainer
  deflected to LiteLLM. The self-host "LLM API/Gateway" is **not** a production proxy — it exists so
  Langfuse can call models for its **own** Playground/judge/experiments. *"Langfuse tracing does not
  need access to the LLM API as traces are captured client-side."* `[iss]` `[doc]`
- **Has real windowed drift** — Monitors, 1h/1d/1w lookbacks, thresholds, `> ≥ < ≤ = ≠`. So *"Langfuse
  only sees one call"* is **wrong**. Every Monitor terminates in a **notification** (Slack/webhook/GH
  Action). **Langfuse hands you an event; you build the actuator** — and it's out-of-band, so the bad
  response already reached the user. That line survives someone citing the webhook docs at us. `[doc]`
- **Monitors are Cloud-only** — a real seam in their OSS story `[?]` *(stated in current docs, not in
  `ee/`; Langfuse ships fast — re-verify before publishing)*
- **Open-core, and the gating is genuinely peripheral — don't overstate it.** MIT except `ee/`
  (admin-api, billing, SSO, audit-log, data-retention). **Tracing, evals, and LLM-as-judge are MIT and
  fully self-hostable.** `[lic]`
- **Don't say "Langfuse touches nothing at runtime"** — Prompt Management is a real runtime hook
  (relabel a version → live apps pick it up, no redeploy). It's human-triggered and operates on
  **config, not responses**: a deploy channel, not a control plane. `[doc]`

### LangSmith — "never in the request path" is now FALSE
- Shipped an **LLM Gateway** that *"sits between your agents and the LLM providers"* — genuinely
  inline, hard-blocks (**HTTP 402** on spend cap), Presidio PII/secret redaction. `[doc]`
- **The saving distinction:** it blocks on **spend, PII, and secrets — never response quality**, and
  blocking is documented **request-side only**. No bad *response* is stopped. Also **private beta**
  (*"APIs and features may change"*); self-hosted availability undocumented. `[doc]`
- **LangSmith Engine** scans every 6h, clusters failures, diagnoses RCA, **drafts PRs** — explicitly
  *"**You** review and merge."* Autonomous detection; **human-gated mutation**. `[doc]`
- **Not OSS.** Self-host is an **Enterprise add-on** with a sales-issued license key. LangChain/LangGraph
  being MIT does **not** extend to the platform. Our claim holds. `[doc]`
- **OTel coexistence: yes** — accepts OTLP; collector fan-out to LangSmith **and** SigNoz simultaneously
  is documented. ⚠️ Do **not** cite `docs/langsmith/export-backend` for this — it's about exporting a
  self-hosted instance's own k8s infra logs, not app traces. `[doc]`

### Helicone — confirmed passive on content, and arguably dead
- **Both enforcement features are INPUT-ONLY.** Moderations: *"the user's latest message is prepared
  for moderation before any chat completion request."* LLM Security: *"Analyzes each **user message**"*
  → and the docs concede: **"The LLM's generated output appears to bypass security inspection."** `[doc]`
- **Structurally incapable of blocking responses:** webhooks fire *"when LLM requests complete"*
  (async, 2-min timeout); online evaluator scores carry a **10-minute processing delay**. `[doc]`
- **Acquired by Mintlify 2026-03-03; in maintenance mode** — *"working closely with every customer to
  support a smooth migration to another platform."* **Arguably no longer a live competitor** — worth a
  sentence, not a slide. `[doc]`
- Apache-2.0, clean, self-hostable. `[lic]`

### The guardrail specialists
- **Guardrails AI** — real inline output validation, `on_fail`: `reask`/`fix`/`filter`/`refrain`/
  `exception`. `reask` re-prompts the **same** model; no fallback. Apache-2.0. **Stateless,
  per-request**; OTel is **export-only**. No window. `[doc]`
- **NeMo Guardrails** — inline output rails (Colang `bot refuse to respond`) → canned refusal, no
  reask. Apache-2.0. **Maintainer admission** ([issue #2028](https://github.com/NVIDIA-NeMo/Guardrails/issues/2028)):
  rails *"inspect messages one at a time"*; windowed cross-turn drift is an **unimplemented proposal**. `[iss]`
- **Arize Phoenix** — **two premise corrections we had wrong.** (1) Own docs: *"Phoenix doesn't block
  requests — your app does."* (2) **Drift was DELETED, not deprecated** — PR #11589, **−42,860 lines**,
  v13.3.0, Feb 2026 (search engines still surface stale drift docs — don't cite them). (3) It is
  **Elastic License 2.0 — NOT OSI open source.** No alert engine at all. `[src]` `[lic]`
- **W&B Weave** — not inline; the app calls `.apply_scorer()` and **the developer writes the fallback**:
  *"guardrails require code changes because they affect your application's control flow."*
  **Their marketing contradicts their docs** — [wandb.ai/site/guardrails](https://wandb.ai/site/guardrails/)
  claims Weave *"automatically routes... to an alternative flow."* **Usable ammunition, but use it
  carefully — it's a fair fight to quote their docs, not a gotcha to quote their marketing.** Apache-2.0,
  license-gated self-host. `[doc]`
- **WhyLabs — the most interesting case, and the closest anyone gets to our thesis.** WhyLabs Secure
  **genuinely blocks responses inline** (verified in `openllmtelemetry` source) **AND** has real
  windowed drift (Hellinger/KL/JS/PSI vs reference profiles). **It has both halves and never joins
  them:** drift computes on the **SaaS control plane over batch profiles** and never feeds back into
  the guardrail container's policy. Also: async path doesn't block, streaming unguarded, **fails open**.
  **Discontinuing operations** (Apple acquisition); repos ~20 months stale. **Do not dismiss its
  per-request blocking as "just monitoring" — it's real.** `[src]`

---

## Objections we must be able to answer

1. **"Portkey's circuit breaker already does this."** → *It does — on HTTP status codes. Ours fires
   while every call returns 200 and the answers are getting worse. And in the OSS gateway the breaker
   doesn't compute state at all.*
2. **"LiteLLM cools down bad deployments automatically."** → *On transport failures in a 1-minute
   window. It cannot see that the model is confidently wrong.*
3. **"Couldn't you just set `failure_status_codes: [446]` in Portkey?"** → *Plausibly, and we're not
   going to pretend otherwise — but it's undocumented, impossible in the OSS gateway, and it counts
   **binary denials**, not a drifting quality score. It can't act while everything returns 200.*
4. **"OpenLIT has guardrails."** → *It does — regex, at the door. It cannot block a hallucination, and
   its alert engine can't fire on LLM behavior at all: `AlertTriggerType` is config-plane only.*
5. **"Isn't this just CI/CD eval gates with auto-rollback?"** → *Those gate a **deploy**. We gate **live
   traffic**. Adjacent, not overlapping.* **← expect this one; it's the sharpest.**
6. **"Did you actually try these?"** → **"We read their source. We didn't run them."** *(Until the
   hands-on TODO below is done — then say what we ran.)*

## The one-liner — ⚠️ EVIDENCE CONTRADICTS THE FROZEN TEXT (human decision required)

`TERMINOLOGY.md` freezes: *"Portkey guards one call. OpenLIT watches. **Argus closes the loop.**"*

**Two of the three claims are now factually refutable:** Portkey does **not** only guard one call (the
breaker is windowed), and OpenLIT does **not** only watch (auto-guards block inline).

`JOURNAL.md`'s own rule: *"Did reality disagree? → **change the ADR, not the code**."* This is that.
**Not changed unilaterally — VISION/TERMINOLOGY are frozen and this is Shafi's call.** Options:

- **Minimal** (removes the falsifiable word, costs nothing rhetorically):
  > Portkey guards **the** call. OpenLIT watches. Argus closes the loop.
- **Strongest** (concedes the breaker and wins anyway) — **recommended**:
  > **Portkey breaks the circuit on errors. OpenLIT watches. Argus closes the loop on behavior.**
- **The full defensible claim, for the competitive slide:**
  > *Portkey's only closed loop from windowed signal to automatic action runs on HTTP error rates.
  > Nothing closes the loop on behavioral quality — no eval score, guardrail failure rate, or output
  > drift over a window can trip a breaker, reroute traffic, or quarantine a model. And in the
  > open-source gateway, the breaker doesn't compute state at all.*

## Hands-on TODO — what P7 needs to actually go 🟢
Run each against **the same** scripted agent that hallucinates a fact, loops, and overspends:
- [ ] **Portkey** — does a guardrail `deny` fire inline? measure added latency. **Does
      `failure_status_codes: [446]` actually trip the breaker?** ← the open threat
- [ ] **OpenLIT** — confirm auto-guards block; confirm no eval-driven alert exists
- [ ] **LiteLLM** — confirm cooldown fires on transport errors and **not** on bad content
- [ ] **Langfuse** — confirm a Monitor fires and terminates in a notification
- [ ] **Guardrails AI** — confirm `reask` re-prompts the same model
- [ ] Screenshot each. *No product needed — these are competitor tools. **But this is still "trying
      tools," not building Argus: it is rule-legal before Jul 20.*** ⚠️ *Time-box it; it is not on the
      critical path, and P5/P6/P9 are.*

## Verdict
- Confirmed **first-hand** that no competitor does Detect+Prevent+Recover+Investigate → ☐ PASS / ☐ FAIL
  → **NOT YET — desk research only. This is the box that keeps P7 at 🟡.**
- Confirmed **from source/docs** that nobody closes a **content-signal** loop → ☑ **yes** (11 tools,
  adversarially tested, incl. 3 genuine partial refutations found and folded in)
- We are NOT about to demo something judges have already seen → ☑ **confirmed** — *narrower than we
  thought, and better-defended for it*

→ `../SUMMARY.md`: P7 **🟡 PARTIAL**
