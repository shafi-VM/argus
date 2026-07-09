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
- **P3 → 🟢:** recovery span (`argus.recovery.reground`) lands as a **child** of `agent.request` in the SAME trace (`parent_span_id` matches `span_id`). "Postmortem writes itself" mechanic works — one waterfall in SigNoz. Ingestion ~1.5s this run (fast end of P1 range).
