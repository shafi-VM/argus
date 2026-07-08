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
