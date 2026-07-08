# P8 — Observability quality

**Question:** would a SigNoz engineer open our traces and think *"these people understand OpenTelemetry"*?

This is a SigNoz hackathon. Instrumentation *craft* is worth real judging points. Grade ourselves
as if a SigNoz maintainer is reviewing our telemetry.

## Checklist
- [ ] **Span names** are meaningful & low-cardinality (`argus.recovery.reground`, not `handler_1`)
- [ ] **Semantic conventions** — use OTel GenAI conventions where they exist (gen_ai.* attrs)
- [ ] **Attributes** are low-cardinality on metrics; prompt/user_id/trace_id live on spans/logs only
- [ ] **Metric → trace → logs** click-through works in SigNoz (exemplars / correlation)
- [ ] **Service boundaries** are obvious (gateway ≠ agent ≠ recovery as distinct services)
- [ ] **Service map** tells the story on its own (gateway → LLM → tools → argus)
- [ ] **Exceptions** — bad responses surface as span events/exceptions, browsable in SigNoz
- [ ] **Recovery is trace-linked** (P3) so one trace narrates the whole incident
- [ ] **Metric temporality** correct (counters as sums, no double-counting)
- [ ] **Resource attributes** set (`service.name`, `service.version`, `deployment.environment`)
- [ ] **No orphan spans / broken context** across the Go↔Python boundary (propagate traceparent)

## The Go↔Python seam (our specific risk)
Argus is Go gateway + Python agent. **W3C traceparent must propagate across that boundary** or the
trace fragments. Assert one continuous trace from gateway → agent → tools.

## Verdict
- All boxes checked → ☐ PASS / ☐ FAIL
- A SigNoz maintainer would nod → ☐ yes / ☐ not yet

→ update `../SUMMARY.md`
