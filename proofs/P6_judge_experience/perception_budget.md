# P6 — Judge experience (perception budget)

**Question:** does the demo *feel* fast? We optimize human perception, not just system latency.

A technically-correct beat that takes 18 s reads as "broken." Budget every beat against how
a judge *perceives* it, and pad the gaps with narration so there is never dead air.

| Beat | Perceived target |
|------|-----------------:|
| Hallucination injected | 0 s (instant, on click) |
| User notices something wrong | < 2 s |
| SigNoz visibly changes | < 5 s |
| Recovery starts | < 6 s |
| Recovery complete | < 10 s |

## Rules
- **Never** let a beat exceed ~10 s of silent waiting. If a real step takes longer, cover it
  with a scripted sentence ("...and here's Argus catching it —") so the wait *feels* intentional.
- Pre-seed history so SigNoz panels are already populated → "SigNoz visibly changes" is a delta on
  existing data, not a cold load.
- If any beat's P5 median > its perception target → shorten the path (pre-warm, poll faster,
  cache the fixture) BEFORE demo day.

## Verdict
- Every beat within its perceived target → ☐ PASS / ☐ FAIL
- No silent gap > 10 s anywhere in the 5 minutes → ☐ PASS / ☐ FAIL

→ update `../SUMMARY.md`
