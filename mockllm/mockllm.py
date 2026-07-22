#!/usr/bin/env python3
"""
Deterministic mock OpenAI /v1/chat/completions server — the demo's replay engine.

Day-1: a stand-in upstream so argusd's trace pipeline can be proven with no API key.
Day-2: this grew two personas + a runtime chaos toggle so it can produce the money
moment on cue (see DEMO_RISK.md — deterministic, airplane-mode-safe).

    python mockllm.py                       # listens on 127.0.0.1:9099, mode=grounded

Chaos toggle (god-mode "Next beat" driver, no restart):
    curl -X POST 127.0.0.1:9099/admin/chaos -d '{"mode":"hallucinated"}'
    curl -X POST 127.0.0.1:9099/admin/chaos -d '{"mode":"grounded"}'
    curl 127.0.0.1:9099/admin/chaos          # -> {"mode": "..."}

Both answers come from fixtures/booking.json — the single source of truth shared with
Ada and (next) argusd's grounding check. "grounded" cites a flight that IS in the
retrieved context; "hallucinated" cites one that is NOT.
"""
import json
import os
from http.server import BaseHTTPRequestHandler, HTTPServer

_FIXTURE = os.path.join(os.path.dirname(__file__), "..", "fixtures", "booking.json")
with open(_FIXTURE, encoding="utf-8") as _f:
    FIXTURE = json.load(_f)

RESPONSES = FIXTURE["responses"]          # {"grounded": ..., "hallucinated": ...}
MODES = tuple(RESPONSES.keys())

# Runtime persona. Flipped via POST /admin/chaos; defaults to the safe answer so a
# fresh boot is never mid-incident.
MODE = "grounded"


class Handler(BaseHTTPRequestHandler):
    def do_POST(self):
        if self.path.rstrip("/") == "/admin/chaos":
            return self._set_chaos()
        return self._chat()

    def do_GET(self):
        if self.path.rstrip("/") == "/admin/chaos":
            return self._json(200, {"mode": MODE})
        return self._json(404, {"error": "not found"})

    def _chat(self):
        n = int(self.headers.get("Content-Length", 0))
        req = json.loads(self.rfile.read(n) or b"{}")
        model = req.get("model", "gpt-4o")
        # argusd appends a REGROUND instruction when an answer failed the Grounding
        # Check. A real model corrects itself when told its claim was unsupported;
        # the replay engine simulates that, deterministically.
        regrounded = any("REGROUND" in (m.get("content") or "")
                         for m in req.get("messages", []))
        # Drift is model-scoped: only the PRIMARY model (gpt-4o) degrades under chaos.
        # The fallback (anything else, e.g. gpt-4o-mini) is always healthy — so when
        # LEARN quarantines gpt-4o and reroutes to the fallback, health recovers.
        primary = os.getenv("MOCK_PRIMARY_MODEL", "gpt-4o")
        if regrounded or model != primary:
            mode = "grounded"
        else:
            mode = MODE
        self._json(200, {
            "id": "chatcmpl-mock",
            "object": "chat.completion",
            "model": model,
            "choices": [{
                "index": 0,
                "message": {"role": "assistant", "content": RESPONSES[mode]},
                "finish_reason": "stop",
            }],
            # wire-faithful: real OpenAI returns total_tokens too. argusd deliberately
            # does not map it to a span attribute (no gen_ai.usage.total_tokens exists).
            "usage": {"prompt_tokens": 42, "completion_tokens": 18, "total_tokens": 60},
        })

    def _set_chaos(self):
        global MODE
        n = int(self.headers.get("Content-Length", 0))
        try:
            body = json.loads(self.rfile.read(n) or b"{}")
            mode = body["mode"]
        except (ValueError, KeyError):
            return self._json(400, {"error": 'body must be {"mode": "..."}'})
        if mode not in MODES:
            return self._json(400, {"error": f"mode must be one of {list(MODES)}"})
        MODE = mode
        return self._json(200, {"mode": MODE})

    def _json(self, status, obj):
        b = json.dumps(obj).encode()
        self.send_response(status)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(b)))
        self.end_headers()
        self.wfile.write(b)

    def log_message(self, *a):
        pass


if __name__ == "__main__":
    print(f"mock LLM on 127.0.0.1:9099 (mode={MODE}, personas={list(MODES)})")
    HTTPServer(("127.0.0.1", 9099), Handler).serve_forever()
