#!/usr/bin/env python3
"""
P4 — Build the 'Argus — Intelligence Health' hero dashboard as code and POST it to SigNoz.
Idempotent: deletes any existing dashboard with the same title first.
Also writes the created JSON to dashboard.json (vendored, reproducible for P9).

    SIGNOZ_URL=http://localhost:8081 SIGNOZ_API_KEY=<key> python build_dashboard.py
"""
import os, json, uuid, urllib.request

URL = os.environ.get("SIGNOZ_URL", "http://localhost:8081").rstrip("/")
KEY = os.environ["SIGNOZ_API_KEY"]
TITLE = "Argus — Intelligence Health"
HDRS = {"Content-Type": "application/json", "SIGNOZ-API-KEY": KEY}


def api(method, path, body=None):
    data = json.dumps(body).encode() if body is not None else None
    req = urllib.request.Request(f"{URL}{path}", data=data, method=method, headers=HDRS)
    with urllib.request.urlopen(req) as r:
        raw = r.read()
        return json.loads(raw) if raw else {}


def query_data(metric, mtype, time_agg, space_agg, reduce_to):
    return {
        "aggregateAttribute": {"dataType": "float64", "id": f"{metric}--float64--{mtype}--true",
                               "isColumn": True, "isJSON": False, "key": metric, "type": mtype},
        "aggregateOperator": time_agg, "dataSource": "metrics", "disabled": False,
        "expression": "A", "filters": {"items": [], "op": "AND"}, "functions": [],
        "groupBy": [], "having": [], "legend": "", "limit": None, "orderBy": [],
        "queryName": "A", "reduceTo": reduce_to, "spaceAggregation": space_agg,
        "stepInterval": 60, "timeAggregation": time_agg,
    }


def widget(title, metric, mtype, panel, time_agg, space_agg, reduce_to, unit=""):
    return {
        "id": str(uuid.uuid4()), "title": title, "description": "",
        "panelTypes": panel, "timePreferance": "GLOBAL_TIME",
        "isStacked": False, "opacity": "1", "nullZeroValues": "zero", "yAxisUnit": unit,
        "query": {"queryType": "builder", "id": str(uuid.uuid4()),
                  "builder": {"queryData": [query_data(metric, mtype, time_agg, space_agg, reduce_to)],
                              "queryFormulas": []},
                  "clickhouse_sql": [{"name": "A", "legend": "", "disabled": False, "query": ""}],
                  "promql": [{"name": "A", "legend": "", "disabled": False, "query": ""}]},
    }


# (title, metric, mtype, panelType, timeAgg, spaceAgg, reduceTo, unit)
PANELS = [
    ("🟢 Infrastructure Health", "argus_infra_health_ratio", "Gauge", "value", "avg", "avg", "last", ""),
    ("🔴 Intelligence Health", "argus_intelligence_health_ratio", "Gauge", "value", "avg", "avg", "last", ""),
    ("Intelligence Health — trend", "argus_intelligence_health_ratio", "Gauge", "graph", "avg", "avg", "avg", ""),
    ("Cost $/request", "argus_cost_usd_per_request", "Gauge", "graph", "avg", "avg", "avg", "none"),
    ("Users impacted", "argus_users_impacted", "Gauge", "value", "avg", "avg", "last", ""),
    ("Grounding failures (rate)", "argus_grounding_failed_total", "Sum", "graph", "rate", "sum", "sum", ""),
    ("Reasoning loops (rate)", "argus_reasoning_loop_total", "Sum", "graph", "rate", "sum", "sum", ""),
    ("Recoveries (rate)", "argus_recoveries_total", "Sum", "graph", "rate", "sum", "sum", ""),
]

# idempotent: remove existing dashboards with this title
for d in api("GET", "/api/v1/dashboards").get("data", []):
    if (d.get("data") or {}).get("title") == TITLE or d.get("title") == TITLE:
        did = d.get("id") or d.get("uuid")
        try:
            api("DELETE", f"/api/v1/dashboards/{did}")
            print(f"deleted existing {did}")
        except Exception as e:
            print("delete skip:", e)

widgets, layout = [], []
for idx, p in enumerate(PANELS):
    w = widget(*p)
    widgets.append(w)
    layout.append({"i": w["id"], "x": (idx % 2) * 6, "y": (idx // 2) * 6,
                   "w": 6, "h": 4 if p[3] == "value" else 6, "moved": False, "static": False})

dashboard = {"title": TITLE,
             "description": "Infra is green; intelligence is not. The stop-scrolling hero panel.",
             "tags": ["argus", "hero"], "layout": layout, "widgets": widgets}

out = api("POST", "/api/v1/dashboards", dashboard)
did = out["data"]["id"]
print(f"CREATED dashboard id={did}")
print(f"OPEN: {URL}/dashboard/{did}")
with open(os.path.join(os.path.dirname(__file__), "dashboard.json"), "w") as f:
    json.dump(out["data"].get("data", out["data"]), f, indent=2)
print("wrote dashboard.json")
