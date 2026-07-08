# P7 — Competitor check (hands-on, not docs)

**Question:** can Portkey / OpenLIT / Langfuse / LangSmith / Helicone already do this?

Don't read marketing. **Actually try each** on the same task: an agent that hallucinates a fact,
loops, and overspends. Record what each can do — first-hand — so when a judge asks
*"how is this different from OpenLIT?"* you answer from experience, not a slide.

Columns: **Detect** (see it happened) · **Prevent** (stop it inline) · **Recover** (fix without a human) · **Investigate** (RCA) · **Open + self-hosted**

| Tool | Detect | Prevent | Recover | Investigate | OSS/self-host | Notes (first-hand) |
|------|:------:|:-------:|:-------:|:-----------:|:-------------:|--------------------|
| **OpenLIT** | | | | | | OTel-native GenAI obs → SigNoz. Expect: watches, doesn't act. |
| **Langfuse** | | | | | | Tracing + evals. Expect: passive. |
| **LangSmith** | | | | | | Evals/tracing, proprietary cloud. |
| **Helicone** | | | | | | Gateway obs + cache/cost. |
| **Portkey** | | | | | | Gateway guardrails + fallbacks + budgets. Closest on Prevent. |
| **Argus** | ✅ | ✅ | ✅ | ✅ | ✅ | The only row with all five. |

## The one-liner we must be able to defend from experience
> "Portkey guards one call. OpenLIT watches. **Neither connects behavioral detection to an
> observability backend that gives you the window to catch loops/cost/drift, then acts on it.**
> Argus closes the loop."

## Verdict
- Confirmed first-hand that no single competitor does Detect+Prevent+Recover+Investigate → ☐ PASS / ☐ FAIL
- We are NOT about to demo something judges have already seen → ☐ confirmed

→ update `../SUMMARY.md`
