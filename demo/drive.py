#!/usr/bin/env python3
"""
Argus demo driver — runs the deterministic system beats end-to-end and MEASURES each
one, so the backup video is a one-take and P5 (beat timing) is retired with real
numbers, not guesses.

It drives only the parts that are deterministic and scriptable:
  - PREVENT: a hallucination is caught + re-grounded inline; the caller sees the
    corrected answer, never the bad one (the money moment).
  - LEARN:   sustained drift -> Argus quarantines the bad model and reroutes ->
    recovers; every request is HTTP 200 the whole time (the competitive kill).

The UI beats (hero dashboard, incident trace waterfall, Mission Control, service map)
are shown live by the presenter — this driver only guarantees the state exists and
prints where to look.

    python demo/drive.py            # assumes the stack is up (see demo/README.md)

Budgets: PREVENT is a per-request reflex (<10s, really milliseconds). LEARN is a
WINDOWED loop — quarantine ~15-25s, recover ~35-50s — inherently NOT a <10s snap;
those beats are narrated over the live dashboard (perception budget), not waited on
in silence. This driver reports both honestly.
"""
import json
import os
import time
import urllib.request

ARGUS = os.getenv("ARGUS_URL", "http://localhost:8088")
MOCK = os.getenv("MOCK_URL", "http://localhost:9099")
# The retrieved context the answer is grounded against (AA42 is real; UA99 is not).
CONTEXT = '{"tool":"flight_search","results":[{"flight":"AA42","depart":"SFO 09:15","arrive":"JFK 17:40"}]}'


def chat(prompt):
    body = json.dumps({"model": "gpt-4o", "messages": [
        {"role": "system", "content": "Answer ONLY from this.\nRETRIEVED_CONTEXT: " + CONTEXT},
        {"role": "user", "content": prompt}]}).encode()
    t0 = time.time()
    r = urllib.request.urlopen(urllib.request.Request(
        ARGUS + "/v1/chat/completions", data=body, headers={"Content-Type": "application/json"}), timeout=15)
    dt = (time.time() - t0) * 1000
    d = json.load(r)
    return r.status, d.get("model"), d["choices"][0]["message"]["content"], dt


def set_drift(on):
    urllib.request.urlopen(urllib.request.Request(
        MOCK + "/admin/chaos", data=json.dumps({"mode": "hallucinated" if on else "grounded"}).encode()), timeout=5)


def preflight():
    try:
        urllib.request.urlopen(ARGUS + "/healthz", timeout=3)
        urllib.request.urlopen(MOCK + "/admin/chaos", timeout=3)
    except Exception as e:
        raise SystemExit("stack not up (%s). See demo/README.md — start mock + argusd first." % e)


def beat_prevent():
    print("\n== BEAT · PREVENT (the money moment) ==")
    set_drift(True)  # primary gpt-4o will hallucinate UA99
    time.sleep(0.2)
    status, model, answer, dt = chat("What is my flight number?")
    leaked = "UA99" in answer
    corrected = "AA42" in answer
    ok = status == 200 and corrected and not leaked
    print("  request under drift -> HTTP %d, %.1f ms round-trip (incl. re-ground)" % (status, dt))
    print("  caller received: %r" % answer)
    print("  hallucination (UA99) reached caller: %s   corrected (AA42): %s" % (leaked, corrected))
    print("  VERDICT: %s" % ("PASS — user never saw the hallucination" if ok else "FAIL"))
    return {"beat": "PREVENT", "ok": ok, "ms": round(dt, 1), "budget": "<10s (per-request reflex)"}


def beat_learn():
    print("\n== BEAT · LEARN (drift -> quarantine -> recover, HTTP 200 throughout) ==")
    set_drift(True)
    t0 = time.time()
    q = r = None
    non200 = 0
    drift_stopped = False
    while time.time() - t0 < 90:
        try:
            st, model, _, _ = chat("What is my flight number?")
        except Exception:
            st, model = 0, "ERR"
        if st != 200:
            non200 += 1
        if q is None and model == "gpt-4o-mini":
            q = time.time() - t0
            print("  >>> QUARANTINE at %.1fs (gpt-4o -> gpt-4o-mini reroute, HTTP %d)" % (q, st))
        if q is not None and not drift_stopped and (time.time() - t0) - q > 8:
            set_drift(False)
            drift_stopped = True
            print("  --- drift stopped; fallback healthy, watch recover ---")
        if drift_stopped and r is None and model == "gpt-4o":
            r = time.time() - t0
            print("  >>> RECOVER at %.1fs (back to gpt-4o, HTTP %d)" % (r, st))
            break
        time.sleep(0.6)
    ok = q is not None and r is not None and non200 == 0
    print("  non-200 responses during the whole arc: %d" % non200)
    print("  VERDICT: %s" % ("PASS — behavior recovered while HTTP stayed 200" if ok else "FAIL"))
    return {"beat": "LEARN", "ok": ok, "quarantine_s": round(q, 1) if q else None,
            "recover_s": round(r, 1) if r else None, "budget": "windowed (narrate over dashboard)"}


def main():
    preflight()
    print("Argus demo driver — measuring the deterministic beats")
    print("Mission Control: %s/mission" % ARGUS)
    results = [beat_prevent(), beat_learn()]
    set_drift(False)  # leave the stack clean/green

    print("\n================ MEASURED BEAT TIMINGS ================")
    for b in results:
        mark = "PASS" if b["ok"] else "FAIL"
        if b["beat"] == "PREVENT":
            print("  %-8s %s   %.1f ms   budget: %s" % (b["beat"], mark, b["ms"], b["budget"]))
        else:
            print("  %-8s %s   quarantine %ss / recover %ss   budget: %s" %
                  (b["beat"], mark, b["quarantine_s"], b["recover_s"], b["budget"]))
    print("======================================================")
    if not all(b["ok"] for b in results):
        raise SystemExit(1)
    print("All deterministic beats PASS. UI beats (dashboard / trace / service map) are presenter-driven.")


if __name__ == "__main__":
    main()
