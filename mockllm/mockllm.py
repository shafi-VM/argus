#!/usr/bin/env python3
"""
Deterministic mock OpenAI /v1/chat/completions server.

Day-1: a stand-in upstream so argusd's trace pipeline can be proven with no API key.
Later: this IS the demo's replay LLM — deterministic answers make the demo
airplane-mode-safe (see DEMO_RISK.md). Chaos will make it return a wrong answer on cue.

    python mockllm.py            # listens on 127.0.0.1:9099
"""
import json
from http.server import BaseHTTPRequestHandler, HTTPServer

# The one canonical "correct, grounded" answer for the SFO->JFK booking flow.
GROUNDED = "Flight AA42 departs SFO 09:15, arrives JFK 17:40, nonstop."


class Handler(BaseHTTPRequestHandler):
    def do_POST(self):
        n = int(self.headers.get("Content-Length", 0))
        req = json.loads(self.rfile.read(n) or b"{}")
        model = req.get("model", "gpt-4o")
        resp = {
            "id": "chatcmpl-mock",
            "object": "chat.completion",
            "model": model,
            "choices": [{
                "index": 0,
                "message": {"role": "assistant", "content": GROUNDED},
                "finish_reason": "stop",
            }],
            "usage": {"prompt_tokens": 42, "completion_tokens": 18, "total_tokens": 60},
        }
        b = json.dumps(resp).encode()
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(b)))
        self.end_headers()
        self.wfile.write(b)

    def log_message(self, *a):
        pass


if __name__ == "__main__":
    print("mock LLM on 127.0.0.1:9099")
    HTTPServer(("127.0.0.1", 9099), Handler).serve_forever()
