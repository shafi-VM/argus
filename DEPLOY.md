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

SigNoz UI comes up on `:8080` (OTLP on `4317`/`4318`). Foundry installs SigNoz **and its MCP server**
in one step. First boot runs schema migrations — the OTLP ingester is ready ~2–3 min *after* the UI
reports healthy (wait for a `200` on an OTLP POST, not just the UI).

## 2. Run Argus against it

argusd is a drop-in OpenAI-compatible proxy that emits OTel to SigNoz and reads its LEARN decisions
back from SigNoz `query_range`. Full one-take runbook — bring up the gateway + agent + replay engine,
then drive and measure the demo beats — is in **[`demo/README.md`](demo/README.md)**:

```bash
python mockllm/mockllm.py                                   # replay engine
SIGNOZ_URL=… SIGNOZ_API_KEY=… go run ./cmd/argusd           # gateway + LEARN + Mission Control (:8088)
python agent/ada.py                                         # cross-service traffic
python demo/drive.py                                        # runs + measures the PREVENT / LEARN beats
```

Provision the hero dashboard from code: `proofs/P4_dashboard/build_dashboard.py` then
`add_trace_panels.py` (both take `SIGNOZ_URL` + `SIGNOZ_API_KEY`).

> A fully one-command `docker compose up` that bootstraps the SigNoz account/key and auto-provisions
> the dashboard is tracked in [#15](https://github.com/shafi-VM/argus/issues/15).
