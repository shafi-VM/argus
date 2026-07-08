#!/usr/bin/env python3
"""
P1 — Poll SigNoz query API for the marker span from emit_otlp.py.
Measures:
  ingestion_lag_s : emit -> first time the span is queryable   (A4, bar < 10 s)
  query_rtt_ms    : round-trip latency of the query API call    (A1, bar < 2 s)

    pip install requests
    SIGNOZ_URL=http://localhost:8080 SIGNOZ_API_KEY=<PAT> \
    RUN_ID=<from emit> EMIT_MS=<from emit> python query_latency.py

VERIFY FIRST (in P0): the exact query_range version (v3 vs v4) and payload shape
for YOUR SigNoz build. The timing harness is the deliverable; tweak the payload to fit.
"""
import os, time, json, requests

URL = os.getenv("SIGNOZ_URL", "http://localhost:8080").rstrip("/")
API_KEY = os.getenv("SIGNOZ_API_KEY", "")
RUN_ID = os.environ["RUN_ID"]
EMIT_MS = int(os.environ["EMIT_MS"])
TIMEOUT_S = int(os.getenv("TIMEOUT_S", "60"))

ENDPOINT = f"{URL}/api/v4/query_range"
HEADERS = {"Content-Type": "application/json"}
if API_KEY:
    HEADERS["SIGNOZ-API-KEY"] = API_KEY


def payload(now_ms):
    # Builder query: list trace spans where argus.run_id == RUN_ID.
    return {
        "start": EMIT_MS - 60_000,
        "end": now_ms + 5_000,
        "step": 60,
        "compositeQuery": {
            "queryType": "builder",
            "panelType": "list",
            "builderQueries": {
                "A": {
                    "dataSource": "traces",
                    "queryName": "A",
                    "expression": "A",
                    "filters": {"op": "AND", "items": [
                        {"key": {"key": "argus.run_id", "type": "tag"}, "op": "=", "value": RUN_ID}
                    ]},
                    "limit": 10,
                }
            },
        },
    }


deadline = time.time() + TIMEOUT_S
while time.time() < deadline:
    now_ms = int(time.time() * 1000)
    t0 = time.time()
    try:
        r = requests.post(ENDPOINT, headers=HEADERS, data=json.dumps(payload(now_ms)), timeout=10)
        rtt_ms = (time.time() - t0) * 1000
        hit = r.ok and RUN_ID in r.text
    except Exception as e:
        print(f"query error: {e}"); time.sleep(1); continue
    print(f"query rtt={rtt_ms:.0f}ms status={r.status_code} hit={hit}")
    if hit:
        lag = time.time() - (EMIT_MS / 1000)
        print(json.dumps({
            "run_id": RUN_ID,
            "ingestion_lag_s": round(lag, 2),
            "query_rtt_ms": round(rtt_ms),
            "verdict_A1_query": "PASS" if rtt_ms < 2000 else "FAIL",
            "verdict_A4_ingestion": "PASS" if lag < 10 else "FAIL",
        }, indent=2))
        break
    time.sleep(1)
else:
    print(json.dumps({"run_id": RUN_ID, "result": "NOT VISIBLE within timeout", "verdict": "FAIL"}))
