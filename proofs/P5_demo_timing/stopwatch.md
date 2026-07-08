# P5 — Demo timing

**Question:** does every demo beat complete end-to-end fast enough to feel live?

Time each beat with a stopwatch (or log timestamps in the god-mode panel). Record into
`timings.csv`. Run the full sequence **3×** and keep the medians. Pass bar: **every beat < 10 s**.

## The beats (must match DEMO_RISK cold-open + PREVENT/LEARN)
1. Chaos injected (hallucination)
2. Bad response intercepted (PREVENT)
3. Re-ground + retry starts
4. Correct answer returned (user-never-notices)
5. Loop/cost chaos injected
6. SigNoz window shows the drift (LEARN)
7. Quarantine/reroute fires
8. Cost flatlines

Anything **15–20 s feels broken even if correct.** Optimize perception (see P6).

→ record numbers in `timings.csv`, verdict in `../SUMMARY.md`
