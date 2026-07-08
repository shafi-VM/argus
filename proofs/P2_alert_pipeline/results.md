# P2 — Results

**Question (A3):** alert-trigger → webhook latency, and the minimum evaluation interval.

| Metric | Value |
|--------|-------|
| Min alert evaluation interval SigNoz allows | ____ s |
| trigger → webhook latency (median of 3) | ____ s |

## Decision
- [ ] alert→webhook is fast & fine-grained enough (< ~10 s) → alerts CAN drive LEARN
- [ ] too slow / too coarse → **LEARN runs on `query_range` polling (P1)**; alerts are notify-only

**Reminder:** PREVENT never touches this pipeline — it's inline in the gateway. This proof is LEARN-only.

→ update `../SUMMARY.md`
