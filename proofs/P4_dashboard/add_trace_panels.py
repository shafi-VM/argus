#!/usr/bin/env python3
"""
Day 4 — Add FAST (trace-derived) moving panels to the hero dashboard.

Why: the metric gauges (build_dashboard.py) lag ~60s on query_range — fine for the
FLAT infra gauge, fatal for the MOVING intelligence signal in a live 5-min demo.
Traces are ~13s fresh (measured 2026-07-22). These panels read argus.decision spans,
the same signal LEARN's control loop reads, so the dashboard and the loop agree.

Idempotent: strips any previously-added trace panels (title tag "[traces]") first,
then appends fresh ones and PUTs the whole dashboard back. Existing metric widgets
are read back from the server in canonical v5 form and re-sent untouched.

    SIGNOZ_URL=http://localhost:8081 SIGNOZ_API_KEY=<key> python add_trace_panels.py
"""
import os, json, uuid, urllib.request

URL = os.environ.get("SIGNOZ_URL", "http://localhost:8081").rstrip("/")
KEY = os.environ["SIGNOZ_API_KEY"]
TITLE = "Argus — Intelligence Health"
HDRS = {"Content-Type": "application/json", "SIGNOZ-API-KEY": KEY}
TAG = "[traces]"  # marks panels this script owns, for idempotent replace

# chat spans only; excludes LEARN/internal spans so counts are per-request decisions
CHAT_FILTER = "name = 'chat gpt-4o' OR name = 'chat gpt-4o-mini'"
# behavioral = grounded outcomes; upstream_error/transport_error excluded (red-team R2)
BEHAVIORAL_FILTER = ("(name = 'chat gpt-4o' OR name = 'chat gpt-4o-mini') AND "
                     "(argus.decision = 'pass' OR argus.decision = 'recovered' OR argus.decision = 'refused')")
PASS_FILTER = ("(name = 'chat gpt-4o' OR name = 'chat gpt-4o-mini') AND argus.decision = 'pass'")


def api(method, path, body=None):
    data = json.dumps(body).encode() if body is not None else None
    req = urllib.request.Request(f"{URL}{path}", data=data, method=method, headers=HDRS)
    with urllib.request.urlopen(req) as r:
        raw = r.read()
        return json.loads(raw) if raw else {}


def trace_query(name, filter_expr, group_by=None, legend=""):
    """One v5 builder trace query — mirrors the query_range spec proven to return data."""
    q = {
        "aggregations": [{"expression": "count()"}],
        "dataSource": "traces", "disabled": False, "expression": name,
        "functions": [], "filter": {"expression": filter_expr},
        "groupBy": group_by or [], "having": {"expression": ""},
        "legend": legend, "limit": None, "orderBy": [],
        "queryName": name, "stepInterval": 30,
    }
    return q


def widget(title, panel, query_data, formulas=None, unit="", stacked=False):
    return {
        "id": str(uuid.uuid4()), "title": f"{title} {TAG}", "description": "",
        "panelTypes": panel, "timePreferance": "GLOBAL_TIME",
        "isStacked": stacked, "opacity": "1", "nullZeroValues": "zero", "yAxisUnit": unit,
        "query": {"queryType": "builder", "id": str(uuid.uuid4()),
                  "builder": {"queryData": query_data, "queryFormulas": formulas or []},
                  "clickhouse_sql": [{"name": "A", "legend": "", "disabled": False, "query": ""}],
                  "promql": [{"name": "A", "legend": "", "disabled": False, "query": ""}]},
    }


def build_trace_widgets():
    # NOTE (verified live 2026-07-24 on SigNoz v0.134): a GROUPED trace count as a dashboard
    # GRAPH widget renders as an error (red ✕) even though the identical query returns data
    # via /api/v5/query_range. The formula panel below renders fine. So the grouped
    # "decisions over time" trace panel is dropped here — the metric panel
    # "Requests by decision (rate)" (from build_dashboard.py) shows the same decision mix and
    # renders, and Mission Control is the 0-lag live surface. This keeps every panel working.
    #
    # Live Intelligence Health = pass / behavioral, from traces (~13s fresh — the fast twin of
    # the ~60s-lagged metric gauge). B is disabled (compute-only for the A/B formula).
    health = widget(
        "Intelligence Health — live", "graph",
        [trace_query("A", PASS_FILTER, legend="pass"),
         dict(trace_query("B", BEHAVIORAL_FILTER, legend="behavioral"), disabled=True)],
        formulas=[{"expression": "A/B", "queryName": "F1", "legend": "grounding rate",
                   "disabled": False}],
    )
    return [health]


def main():
    ds = api("GET", "/api/v1/dashboards").get("data", [])
    did = None
    for d in ds:
        if ((d.get("data") or {}).get("title") or d.get("title")) == TITLE:
            did = d.get("id") or d.get("uuid")
            break
    if not did:
        raise SystemExit("hero dashboard not found — run build_dashboard.py first")

    data = api("GET", f"/api/v1/dashboards/{did}").get("data", {}).get("data", {})
    widgets = [w for w in data.get("widgets", []) if TAG not in (w.get("title") or "")]
    layout = [l for l in data.get("layout", [])
              if l.get("i") in {w["id"] for w in widgets}]

    new = build_trace_widgets()
    # place trace panels in a fresh row below existing content
    base_y = (max([l["y"] + l["h"] for l in layout], default=0))
    for idx, w in enumerate(new):
        widgets.append(w)
        layout.append({"i": w["id"], "x": (idx % 2) * 6, "y": base_y, "w": 6, "h": 6,
                       "moved": False, "static": False})

    data["widgets"] = widgets
    data["layout"] = layout
    out = api("PUT", f"/api/v1/dashboards/{did}", data)
    print(f"PUT ok. dashboard id={did}  widgets now={len(widgets)}")

    # verify our trace widgets survived server normalization
    back = api("GET", f"/api/v1/dashboards/{did}").get("data", {}).get("data", {})
    ours = [w for w in back.get("widgets", []) if TAG in (w.get("title") or "")]
    print(f"trace panels stored: {len(ours)}")
    for w in ours:
        qd = w["query"]["builder"]["queryData"]
        fm = w["query"]["builder"]["queryFormulas"]
        print(f"  - {w['title']!r}: dataSource={qd[0].get('dataSource')} "
              f"groupBy={qd[0].get('groupBy')} formulas={[f.get('expression') for f in fm]}")

    # vendor the final (metrics + traces) dashboard for reproducible judge-install (P9)
    with open(os.path.join(os.path.dirname(__file__), "dashboard.json"), "w") as f:
        json.dump(back, f, indent=2)
    print("wrote dashboard.json (metrics + trace panels)")
    print(f"OPEN: {URL}/dashboard/{did}")


if __name__ == "__main__":
    main()
