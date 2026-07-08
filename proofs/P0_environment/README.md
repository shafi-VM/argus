# P0 — Environment

**Question:** Does the full SigNoz stack boot and run **100% offline** on the demo laptop?

This is the gate. If this fails, everything downstream is theoretical.

## Install method — VERIFIED 2026-07-08 (this CHANGED; see JOURNAL Day 1)
SigNoz **deprecated the bundled docker-compose** (`deploy/docker/`) as of ~v0.130 — it now ships
via **Foundry**. The old `deploy/docker/docker-compose.yaml` no longer exists upstream. Do not use it.

1. Install Foundry (one time):
   ```bash
   curl -fsSL https://signoz.io/foundry.sh | bash        # installs ~/.local/bin/foundryctl
   ```
2. Generate the real, version-locked compose (source of truth):
   ```bash
   foundryctl gen examples
   # -> docs/examples/docker/compose/pours/deployment/compose.yaml (+ pours/ config)
   ```
3. Bring it up FROM the deployment dir (relative volume paths require this cwd):
   ```bash
   cd docs/examples/docker/compose/pours/deployment
   docker compose -f compose.yaml up -d
   ```
   **Verified ports:** OTLP `4317`/`4318` (host) · SigNoz UI+query `8080` (host) · MCP `8000` · Postgres + ClickHouse **internal-only** (no host publish → `9000` is NOT a conflict).
   **Local override on this machine:** host `8080` was taken (Python dev server) → UI remapped to `8081` (`8081:8080`). Point the proof scripts at it:
   ```bash
   export SIGNOZ_URL=http://localhost:8081 SIGNOZ_UI=http://localhost:8081
   ```
4. Offline bundle (demo-day): `docker save $(docker compose -f compose.yaml config --images) -o signoz-images.tar`
5. **Turn wifi OFF.** Run `./verify_offline.sh`. Everything must still pass.

## Record (fill in)
- SigNoz version / image digests: `____`
- OTLP endpoints reachable: gRPC `localhost:4317` ☐  HTTP `localhost:4318` ☐
- SigNoz UI: `localhost:8080` (or `3301`) ☐  ← confirm the actual port for your build
- Query API base + auth (PAT header `SIGNOZ-API-KEY`): `____`
- Cold-boot → all-green time: `____` (target: boot ≥30 min before demo regardless)
- **Offline run passed:** ☐

## Baseline findings — 2026-07-07
- ✅ Docker 28.4.0 · Compose v2.39.2 · daemon UP · git 2.50.1 · python 3.13.7
- ✅ 32 GB RAM · 115 GB free disk · ports 4317 / 4318 / 3301 free
- 🔴 **Port 8080 IN USE** (local Python dev server, pid 59047) → SigNoz UI host-port conflict
- 🔴 **Port 9000 IN USE** (local node dev server, pid 68254) → collides with our webhook + ClickHouse native port

### Resolution (minimum change)
- Webhook sidecar moved **9000 → 9010** (`WEBHOOK_PORT`; see `docker-compose.yml` + `../P2_alert_pipeline/webhook_server.py`).
- SigNoz UI 8080: either **free port 8080** (stop pid 59047) OR map SigNoz's frontend to a free host port (e.g. 8081) and set `SIGNOZ_URL` / `SIGNOZ_UI` env so the proof scripts match.

**Verdict:** 🟡 baseline PASS; stack bring-up + offline verify PENDING → update `../SUMMARY.md`
