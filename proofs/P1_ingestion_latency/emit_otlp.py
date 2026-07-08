#!/usr/bin/env python3
"""
P1 — Emit a uniquely-tagged marker span to SigNoz via OTLP; print the emit time.
Pair with query_latency.py to measure ingestion lag (A4) and query latency (A1).

    pip install opentelemetry-sdk opentelemetry-exporter-otlp
    OTLP_ENDPOINT=localhost:4317 python emit_otlp.py

Prints JSON with run_id + emit_ms. Feed those to query_latency.py.
"""
import os, time, uuid, json
from opentelemetry import trace
from opentelemetry.sdk.resources import Resource
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import SimpleSpanProcessor
from opentelemetry.exporter.otlp.proto.grpc.trace_exporter import OTLPSpanExporter

ENDPOINT = os.getenv("OTLP_ENDPOINT", "localhost:4317")
RUN_ID = os.getenv("RUN_ID", uuid.uuid4().hex[:12])

provider = TracerProvider(resource=Resource.create({"service.name": "argus-proof-emitter"}))
# SimpleSpanProcessor exports immediately, so emit_ms is honest (no batch delay).
provider.add_span_processor(SimpleSpanProcessor(OTLPSpanExporter(endpoint=ENDPOINT, insecure=True)))
trace.set_tracer_provider(provider)
tracer = trace.get_tracer("argus.proof")

emit_ms = int(time.time() * 1000)
with tracer.start_as_current_span("argus.proof.marker") as span:
    span.set_attribute("argus.run_id", RUN_ID)
    span.set_attribute("argus.emit_ms", emit_ms)

provider.force_flush()
print(json.dumps({"run_id": RUN_ID, "emit_ms": emit_ms, "endpoint": ENDPOINT}))
