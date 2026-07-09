#!/usr/bin/env python3
"""
P4 — Mock 'argus_*' metrics emitter.

Produces the metric contract from dashboard.json so the Intelligence Health dashboard
has real data to render. Low-cardinality labels ONLY (model/agent/tool/decision/tenant_bucket).

Story it tells over the run: infra stays green (~99.99%) while INTELLIGENCE health degrades
(hallucinations/loops/cost climb), then recoveries kick in.

    pip install opentelemetry-sdk opentelemetry-exporter-otlp
    OTLP_ENDPOINT=localhost:4317 DURATION=60 STEP=3 python metrics_emit.py
"""
import os, time
from opentelemetry.sdk.resources import Resource
from opentelemetry.sdk.metrics import MeterProvider
from opentelemetry.sdk.metrics.export import PeriodicExportingMetricReader
from opentelemetry.exporter.otlp.proto.grpc.metric_exporter import OTLPMetricExporter
from opentelemetry import metrics

ENDPOINT = os.getenv("OTLP_ENDPOINT", "localhost:4317")
DURATION = int(os.getenv("DURATION", "60"))
STEP = int(os.getenv("STEP", "3"))

reader = PeriodicExportingMetricReader(
    OTLPMetricExporter(endpoint=ENDPOINT, insecure=True),
    export_interval_millis=STEP * 1000,
)
provider = MeterProvider(resource=Resource.create({"service.name": "argus"}),
                         metric_readers=[reader])
metrics.set_meter_provider(provider)
m = metrics.get_meter("argus.metrics")

# gauges (point-in-time ratios / values)
g_infra = m.create_gauge("argus_infra_health_ratio")
g_intel = m.create_gauge("argus_intelligence_health_ratio")
g_cost = m.create_gauge("argus_cost_usd_per_request")
g_users = m.create_gauge("argus_users_impacted")
# counters (cumulative)
c_ground = m.create_counter("argus_grounding_failed_total")
c_loop = m.create_counter("argus_reasoning_loop_total")
c_prevent = m.create_counter("argus_prevent_total")
c_recover = m.create_counter("argus_recoveries_total")

LBL = {"model": "gpt-4o", "agent": "booking-agent", "tenant_bucket": "t1"}

t0 = time.time()
i = 0
while time.time() - t0 < DURATION:
    i += 1
    frac = min(1.0, (time.time() - t0) / DURATION)
    g_infra.set(0.9999, LBL)                               # infra: flat green
    g_intel.set(round(max(0.12, 0.92 - 0.8 * frac), 3), LBL)  # intelligence: degrades
    g_cost.set(round(0.02 + 0.6 * frac, 3), LBL)              # $/req climbs
    g_users.set(int(300 * frac), LBL)
    c_ground.add(2, {**LBL, "decision": "blocked"})
    if frac > 0.3:
        c_loop.add(1, {**LBL, "tool": "flight-search"})
    c_prevent.add(2, {**LBL, "decision": "blocked"})
    c_recover.add(2, {**LBL, "decision": "recovered"})
    time.sleep(STEP)

provider.force_flush()
provider.shutdown()
print(f"emitted argus_* metrics for {DURATION}s ({i} points) to {ENDPOINT}")
