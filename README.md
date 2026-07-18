# Argus — The Immune System for AI Agents

> AI infrastructure can be perfectly healthy while AI behavior is catastrophically wrong.
> Argus detects the behavioral failures infra monitoring can't see — hallucinations, loops,
> runaway cost, poisoned context — and recovers before users notice.
>
> Built on OpenTelemetry. Powered by [SigNoz](https://signoz.io).

**Portkey breaks the circuit on errors. OpenLIT watches. Argus closes the loop on behavior.**

---

## Two capabilities

- **PREVENT** — inline, in the gateway, milliseconds. A bad response is intercepted and recovered
  before it reaches the user. Depends on nothing external.
- **LEARN** — windowed, through SigNoz, seconds. Behavioral drift (quality falling while infra
  stays green) is detected from SigNoz telemetry, and the offending model is quarantined / rerouted.

Fast reflexes. Slow adaptation. An immune system.

## Status

🚧 Early and **proof-driven**. Before writing product code we retire the architectural risks as
measurable proofs — see [`proofs/SUMMARY.md`](proofs/SUMMARY.md) for the live scoreboard.

- ✅ **P0** — current SigNoz (Foundry, v0.132) boots, serves, and ingests OTLP on a clean machine.
- ⏳ **P1–P9** — ingestion/query latency, trace correlation, the hero dashboard, demo timing,
  competitor check, and a <10-minute judge install.

## Stack

Go gateway (`cmd/argusd`, one binary) · Python demo agent (`agent/`) · SigNoz (traces, metrics,
alerts, MCP) · OpenTelemetry everywhere · Docker.

## Why now

See [`WHY_NOW.md`](WHY_NOW.md). Built for the **Agents of SigNoz** hackathon.

## Repo map

`VISION.md` · `WHY_NOW.md` · `DECISIONS.md` (ADRs) · `KILL_LIST.md` · `TERMINOLOGY.md` ·
`DEMO_RISK.md` · `ASSUMPTIONS.md` · `proofs/` (the de-risking harness) · `JOURNAL.md`

## License

MIT — see [`LICENSE`](LICENSE).
