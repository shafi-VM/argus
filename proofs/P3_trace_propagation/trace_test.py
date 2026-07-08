#!/usr/bin/env python3
"""
P3 — Prove the recovery span shares the request's trace_id (A6).
Emits: request span -> (bad-response event) -> recovery child span, ALL one trace.
Prints trace_id; open it in SigNoz Traces and confirm ONE waterfall.

    pip install opentelemetry-sdk opentelemetry-exporter-otlp
    OTLP_ENDPOINT=localhost:4317 python trace_test.py
"""
import os, time, json
from opentelemetry import trace
from opentelemetry.sdk.resources import Resource
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import SimpleSpanProcessor
from opentelemetry.exporter.otlp.proto.grpc.trace_exporter import OTLPSpanExporter

ENDPOINT = os.getenv("OTLP_ENDPOINT", "localhost:4317")
provider = TracerProvider(resource=Resource.create({"service.name": "argus-gateway"}))
provider.add_span_processor(SimpleSpanProcessor(OTLPSpanExporter(endpoint=ENDPOINT, insecure=True)))
trace.set_tracer_provider(provider)
tracer = trace.get_tracer("argus.proof")

with tracer.start_as_current_span("agent.request") as req:
    tid = format(req.get_span_context().trace_id, "032x")
    req.set_attribute("argus.decision", "PREVENT")
    # bad response detected -> recorded as a span event (shows in SigNoz events/exceptions)
    req.add_event("argus.behavior.blocked",
                  {"argus.signal": "grounding", "argus.grounded": False})
    with tracer.start_as_current_span("argus.recovery.reground") as rec:
        rec.set_attribute("argus.action", "reground+retry")
        time.sleep(0.05)

provider.force_flush()
print(json.dumps({"trace_id": tid,
                  "expect": "one trace: agent.request > argus.recovery.reground",
                  "verify": "open this trace_id in SigNoz -> single linked waterfall = A6 PASS"}))
