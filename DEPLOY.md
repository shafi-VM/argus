# Deploy — reproduce our SigNoz with Foundry

Per the hackathon rules, this repo ships the **SigNoz Foundry** config so judges can reproduce our
exact deployment: [`casting.yaml`](casting.yaml) + [`casting.yaml.lock`](casting.yaml.lock).

We run SigNoz the SigNoz-native way — [Foundry](https://github.com/SigNoz/foundry), *"one config, one
command."* The `casting.yaml` declares the deployment (docker-compose flavor); `casting.yaml.lock` is
the fully-resolved, reproducible state (collectors, OTLP receivers, ClickHouse, all pinned).

## 1. Bring up SigNoz (reproducible)

```bash
# install foundryctl — https://github.com/SigNoz/foundry
foundryctl forge          # reads casting.yaml -> generates the deployment + writes casting.yaml.lock
docker compose up -d      # from the generated compose directory
```

The SigNoz query service/UI listens on container port `8080` (Foundry publishes it to host `:8081`);
OTLP is on `4317`/`4318`. Foundry installs SigNoz **and its MCP server** in one step. First boot runs
schema migrations — the OTLP ingester is ready ~2–3 min *after* the UI reports healthy (wait for a
`200` on an OTLP POST, not just the UI).

## 2. Run Argus against it

> **Just want the money moment?** You don't need any of this. `docker compose up --build` + `make demo`
> runs the PREVENT reflex + Mission Control with **no SigNoz at all** (see the README Quickstart). The
> steps below add the SigNoz-backed **LEARN** arc.

argusd is a drop-in OpenAI-compatible proxy that emits OTel to SigNoz and reads its LEARN decisions
back from SigNoz `query_range`.

### Option A — one command (Docker)

With SigNoz up on its `signoz-network` (step 1) and your key in `.env`:

```bash
cp .env.example .env        # set SIGNOZ_API_KEY
make demo-signoz            # layers compose.signoz.yaml, runs PREVENT + LEARN + provisions the dashboard
```

This joins Argus to `signoz-network` and talks to SigNoz **by service DNS** — telemetry goes straight to
`signoz-ingester-1:4317`, LEARN reads back from `signoz-signoz-0:8080`. No host round-trip, so no
host-port clash and no Host-header rejection. If your SigNoz uses different service names/ports, override
`SIGNOZ_URL` / `OTLP_ENDPOINT` in `.env` (a host-published SigNoz is `http://host.docker.internal:8081`).

### Option B — local processes (no containers)

The proven full-arc path used to generate the trace/dashboard artifacts — argusd on the host, exporting
to `localhost:4317`, reading `query_range` from the host-published UI. Full one-take runbook in
**[`demo/README.md`](demo/README.md)**:

```bash
python mockllm/mockllm.py                                   # replay engine
SIGNOZ_URL=… SIGNOZ_API_KEY=… go run ./cmd/argusd           # gateway + LEARN + Mission Control (:8088)
python agent/ada.py                                         # cross-service traffic
python demo/drive.py                                        # runs + measures the PREVENT / LEARN beats
```

Provision the hero dashboard from code (Option A does this automatically):
`proofs/P4_dashboard/build_dashboard.py` then `add_trace_panels.py` (both take `SIGNOZ_URL` +
`SIGNOZ_API_KEY`).
