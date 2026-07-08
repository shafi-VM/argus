#!/usr/bin/env python3
"""
P2 — Minimal webhook catcher (stdlib only). Records the wall-clock time SigNoz
delivers an alert. Combine with the alert trigger time to measure alert->webhook
latency (A3).

    python webhook_server.py          # listens on 0.0.0.0:9010, POST /webhook

Port 9010 (not 9000): 9000 is ClickHouse's native port and was also held by a local
dev server in the P0 baseline. Override with WEBHOOK_PORT if needed.
Point a SigNoz webhook channel at http://host.docker.internal:9010/webhook.
"""
import json, os, time
from http.server import BaseHTTPRequestHandler, HTTPServer


class Handler(BaseHTTPRequestHandler):
    def do_POST(self):
        n = int(self.headers.get("Content-Length", 0))
        body = self.rfile.read(n).decode("utf-8", "replace")
        recv_ms = int(time.time() * 1000)
        print(json.dumps({"received_ms": recv_ms, "path": self.path, "body": body[:1500]}))
        self.send_response(200); self.end_headers(); self.wfile.write(b"ok")

    def log_message(self, *a):  # silence default logging
        pass


if __name__ == "__main__":
    port = int(os.getenv("WEBHOOK_PORT", "9010"))
    print(f"listening on 0.0.0.0:{port}  (POST /webhook)")
    HTTPServer(("0.0.0.0", port), Handler).serve_forever()
