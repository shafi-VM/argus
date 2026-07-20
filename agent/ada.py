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
import urllib.request

from opentelemetry import trace, propagate
from opentelemetry.sdk.resources import Resource
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import SimpleSpanProcessor
from opentelemetry.exporter.otlp.proto.grpc.trace_exporter import OTLPSpanExporter

ARGUS = os.getenv("ARGUS_URL", "http://localhost:8088")

_provider = TracerProvider(resource=Resource.create(
    {"service.name": "ada-agent", "service.version": "0.1.0"}))
_provider.add_span_processor(SimpleSpanProcessor(
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

        body = json.dumps({
            "model": "gpt-4o",
            "messages": [{"role": "user", "content": prompt}],
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
    print("Ada answer:", chat("Book me a flight from SFO to JFK next Tuesday."))
    _provider.force_flush()
