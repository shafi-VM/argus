#!/usr/bin/env python3
"""
P4 — Build the 'Argus — Intelligence Health' hero dashboard as code and POST it to SigNoz.
Idempotent: deletes any existing dashboard with the same title first.
Also writes the created JSON to dashboard.json (vendored, reproducible for P9).

Every panel maps to a metric argusd ACTUALLY emits (verified live 2026-07-24):
  argus_infra_health_ratio         (gauge)  — 2xx/all, stays green while behavior rots
  argus_intelligence_health_ratio  (gauge)  — the cold-open RED number
  argus_cost_usd_per_request       (gauge)  — windowed avg cost
  argus_requests_total{decision}   (sum)    — the decision mix (pass/recovered/refused/...)
  argus_grounded_total             (sum)    — grounded first-try rate
The old mock-contract metrics (users_impacted, reasoning_loop, grounding_failed,
recoveries, a phantom infra metric) were removed — they never existed in real argusd and
rendered as dead 'No Data' panels. Trace panels are added by add_trace_panels.py.

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


def query_data(metric, mtype, time_agg, space_agg, reduce_to, group_by=None, legend=""):
    gb = []
    for g in (group_by or []):
        gb.append({"key": g, "dataType": "string", "type": "tag", "isColumn": False, "isJSON": False})
    return {
        "aggregateAttribute": {"dataType": "float64", "id": f"{metric}--float64--{mtype}--true",
                               "isColumn": True, "isJSON": False, "key": metric, "type": mtype},
        "aggregateOperator": time_agg, "dataSource": "metrics", "disabled": False,
        "expression": "A", "filters": {"items": [], "op": "AND"}, "functions": [],
        "groupBy": gb, "having": [], "legend": legend, "limit": None, "orderBy": [],
        "queryName": "A", "reduceTo": reduce_to, "spaceAggregation": space_agg,
        "stepInterval": 60, "timeAggregation": time_agg,
    }


def widget(title, metric, mtype, panel, time_agg, space_agg, reduce_to,
           unit="", group_by=None, legend="", stacked=False):
    return {
        "id": str(uuid.uuid4()), "title": title, "description": "",
        "panelTypes": panel, "timePreferance": "GLOBAL_TIME",
        "isStacked": stacked, "opacity": "1", "nullZeroValues": "zero", "yAxisUnit": unit,
        "query": {"queryType": "builder", "id": str(uuid.uuid4()),
                  "builder": {"queryData": [query_data(metric, mtype, time_agg, space_agg,
                                                        reduce_to, group_by, legend)],
                              "queryFormulas": []},
                  "clickhouse_sql": [{"name": "A", "legend": "", "disabled": False, "query": ""}],
                  "promql": [{"name": "A", "legend": "", "disabled": False, "query": ""}]},
    }


# Each entry: kwargs to widget(). All metrics below are emitted by real argusd.
PANELS = [
    dict(title="🟢 Infrastructure Health", metric="argus_infra_health_ratio", mtype="Gauge",
         panel="value", time_agg="avg", space_agg="avg", reduce_to="last"),
    dict(title="🔴 Intelligence Health", metric="argus_intelligence_health_ratio", mtype="Gauge",
         panel="value", time_agg="avg", space_agg="avg", reduce_to="last"),
    dict(title="Intelligence Health — trend", metric="argus_intelligence_health_ratio", mtype="Gauge",
         panel="graph", time_agg="avg", space_agg="avg", reduce_to="avg"),
    dict(title="Cost $/request", metric="argus_cost_usd_per_request", mtype="Gauge",
         panel="graph", time_agg="avg", space_agg="avg", reduce_to="avg", unit="none"),
    dict(title="Requests by decision (rate)", metric="argus_requests_total", mtype="Sum",
         panel="graph", time_agg="rate", space_agg="sum", reduce_to="sum",
         group_by=["decision"], legend="{{decision}}", stacked=True),
    dict(title="Grounded first-try (rate)", metric="argus_grounded_total", mtype="Sum",
         panel="graph", time_agg="rate", space_agg="sum", reduce_to="sum"),
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
    w = widget(**p)
    widgets.append(w)
    layout.append({"i": w["id"], "x": (idx % 2) * 6, "y": (idx // 2) * 6,
                   "w": 6, "h": 4 if p["panel"] == "value" else 6, "moved": False, "static": False})

dashboard = {"title": TITLE,
             "description": "Infra is green; intelligence is not. The stop-scrolling hero panel.",
             "tags": ["argus", "hero"], "layout": layout, "widgets": widgets}

out = api("POST", "/api/v1/dashboards", dashboard)
did = out["data"]["id"]
print(f"CREATED dashboard id={did}  panels={len(widgets)}")
print(f"OPEN: {URL}/dashboard/{did}")
with open(os.path.join(os.path.dirname(__file__), "dashboard.json"), "w") as f:
    json.dump(out["data"].get("data", out["data"]), f, indent=2)
print("wrote dashboard.json")
