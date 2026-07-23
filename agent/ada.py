#!/usr/bin/env python3
"""
Ada — the demo agent.

Calls the LLM *through* argusd (drop-in OpenAI base URL), emits its own OTel spans,
and propagates W3C traceparent so the agent span and the gateway's CLIENT span
render as ONE linked trace in SigNoz.

    pip install opentelemetry-sdk opentelemetry-exporter-otlp
    ARGUS_URL=http://localhost:8088 python ada.py
"""
import os
import json
import uuid
import urllib.request

from opentelemetry import trace, propagate
from opentelemetry.sdk.resources import Resource
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor
from opentelemetry.exporter.otlp.proto.grpc.trace_exporter import OTLPSpanExporter

ARGUS = os.getenv("ARGUS_URL", "http://localhost:8088")

# Single source of truth, shared with the mock LLM and (next) argusd's grounding check.
_FIXTURE = os.path.join(os.path.dirname(__file__), "..", "fixtures", "booking.json")
with open(_FIXTURE, encoding="utf-8") as _f:
    FIXTURE = json.load(_f)

# Full resource attrs (P8): a SigNoz engineer looks for service.instance.id + a
# low-cardinality environment, not just name/version. Matches argusd's resource shape.
_provider = TracerProvider(resource=Resource.create({
    "service.name": "ada-agent",
    "service.version": "0.1.0",
    "service.instance.id": uuid.uuid4().hex,
    "deployment.environment.name": os.getenv("ARGUS_ENV", "demo"),
}))
# BatchSpanProcessor, not Simple: Simple exports (blocking) on every span.end(). Ada
# force_flush()es before exit, so batching loses nothing and matches production shape.
_provider.add_span_processor(BatchSpanProcessor(
    OTLPSpanExporter(endpoint="localhost:4317", insecure=True)))
trace.set_tracer_provider(_provider)
tracer = trace.get_tracer("ada")


def chat(prompt: str) -> str:
    # span name follows OTel GenAI semconv: "invoke_agent {agent.name}"
    with tracer.start_as_current_span("invoke_agent ada") as span:
        span.set_attribute("gen_ai.operation.name", "invoke_agent")
        span.set_attribute("gen_ai.agent.name", "ada")

        headers = {"Content-Type": "application/json"}
        propagate.inject(headers)  # inject traceparent -> argusd parents its span on ours

        # Carry the retrieved tool context IN the request. This is the "provided
        # context" argusd's grounding check (#5) compares the answer against — the
        # detector grounds claims-vs-context, so the context must travel in-band.
        context = json.dumps(FIXTURE["tool_context"])
        body = json.dumps({
            "model": "gpt-4o",
            "messages": [
                {"role": "system", "content":
                    "You are a booking assistant. Answer ONLY using this retrieved "
                    f"flight data.\nRETRIEVED_CONTEXT: {context}"},
                {"role": "user", "content": prompt},
            ],
        }).encode()
        req = urllib.request.Request(
            f"{ARGUS}/v1/chat/completions", data=body, headers=headers, method="POST")
        with urllib.request.urlopen(req, timeout=30) as r:
            resp = json.load(r)

        answer = resp["choices"][0]["message"]["content"]
        span.set_attribute("argus.answer_chars", len(answer))
        return answer


if __name__ == "__main__":
    print("Ada: booking SFO -> JFK ...")
    print("Ada answer:", chat(FIXTURE["user_prompt"]))
    _provider.force_flush()
