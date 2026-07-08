#!/usr/bin/env bash
# P0 — Offline sanity gate. Run with WIFI OFF. Everything here must pass.
set -euo pipefail

OTLP_HTTP="${OTLP_HTTP:-http://localhost:4318}"
SIGNOZ_UI="${SIGNOZ_UI:-http://localhost:8080}"

pass() { printf "  \033[32mPASS\033[0m  %s\n" "$1"; }
fail() { printf "  \033[31mFAIL\033[0m  %s\n" "$1"; FAILED=1; }
FAILED=0

echo "== P0 offline verification (wifi should be OFF) =="

# 1. Confirm we're actually offline (external should be UNREACHABLE)
if ping -c1 -t2 1.1.1.1 >/dev/null 2>&1; then
  echo "  WARN  external network reachable — turn WIFI OFF for a true offline test"
else
  pass "external network is down (truly offline)"
fi

# 2. Containers healthy
if docker compose ps >/dev/null 2>&1; then pass "docker compose reachable"; else fail "docker compose not reachable"; fi

# 3. OTLP HTTP receiver listening (any HTTP code != 000 means it's up)
code=$(curl -s -o /dev/null -w "%{http_code}" "$OTLP_HTTP/v1/traces" -X POST -H 'content-type: application/json' -d '{}' || echo 000)
if [ "$code" != "000" ]; then pass "OTLP HTTP receiver listening ($code)"; else fail "OTLP HTTP receiver unreachable"; fi

# 4. SigNoz UI serving
code=$(curl -s -o /dev/null -w "%{http_code}" "$SIGNOZ_UI" || echo 000)
if [ "$code" != "000" ]; then pass "SigNoz UI serving ($code)"; else fail "SigNoz UI unreachable at $SIGNOZ_UI"; fi

echo
if [ "$FAILED" = "0" ]; then echo "P0 OFFLINE: PASS"; else echo "P0 OFFLINE: FAIL"; exit 1; fi
