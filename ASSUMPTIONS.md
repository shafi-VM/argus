# ASSUMPTIONS.md — the engineering bible

No opinions. Only experiments. An assumption is not "true" until it has a
**green result and a number next to it**. Until then it is a risk, not a fact.

Prove the ground before pouring concrete. Nothing gets built on top of a 🔴.

| # | Assumption | Status | How to prove | Pass bar |
|---|-----------|--------|--------------|----------|
| A1 | SigNoz **Query API (`query_range`)** latency is acceptable for LEARN decisions | 🔴 Unproven | Measure round-trip on a windowed behavioral query | < 2 s |
| A2 | **SigNoz MCP** is suitable for the agent "investigation" beat | 🔴 Unproven | Prototype an MCP query, time it | Works; seconds OK (beat is optional) |
| A3 | **Alert → webhook** latency + min eval interval | 🔴 Unproven | Create a threshold alert, measure fire→webhook | Known number; decide alert-vs-poll |
| A4 | **ClickHouse ingestion lag** (OTLP emit → queryable) | 🔴 Unproven | Emit a marker span/metric, poll until visible | Single-digit seconds |
| A5 | **Dashboard provisioning** from JSON works on `compose up` | 🔴 Unproven | Import dashboard JSON, screenshot-verify panels | Panels render from seeded data |
| A6 | **Trace correlation** end-to-end (recovery span shares `trace_id`) | 🔴 Unproven | Propagate context, assert one linked trace | One waterfall: bad→block→reground→ok |
| A7 | **OTel metric cardinality** is manageable | 🔴 Unproven | Review label sets; load a burst; watch CH | No prompt/user_id/trace_id on metric labels |
| A8 | **PREVENT** deterministic grounding check (Go-native, no inline ML — ADR-0002) meets latency budget | 🔴 Unproven | Time the grounding check inline in the gateway | Adds < 50 ms p95 |
| A9 | The demo can run **100% offline** (no venue network on critical path) | 🔴 Unproven | Boot + full run with wifi OFF | Complete run, wifi off |
| A10 | **Deterministic replay** LLM produces the scripted bad→good behavior | 🔴 Unproven | Build replay/mock LLM, run 20× | Identical every run |

### Rule
Every engineering session picks a 🔴 and turns it green with a measurement.
We do not build features. We retire assumptions.
