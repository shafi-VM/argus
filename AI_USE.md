# AI assistant use — disclosure

Per the Agents of SigNoz rules (*"Disclose AI assistant use or face disqualification"*), here is
exactly how AI was used to build Argus.

## Tool
- **Claude (Anthropic), via Claude Code** — used throughout for engineering, code review, writing
  tests, documentation, and verifying telemetry against live SigNoz.

## Human ownership (Shafi · [`shafi-VM`](https://github.com/shafi-VM), and Abhishek · [`Abhishekreddy31`](https://github.com/Abhishekreddy31))
The team directed the work and owns every decision. Concretely, and verifiable in the git history:

- **Every product and architecture decision was made by the team** — the PREVENT/LEARN split, the
  deterministic grounding check (ADR-0002), the "never block the hot path on SigNoz" boundary
  (ADR-0003), the KILL_LIST scope, and the frozen one-liner.
- **Every change went through branch → Pull Request → human review → merge.** No code was merged
  unreviewed. The PR threads contain real, adversarial review — e.g. reviews that caught an inert
  staleness guard, a metric→trace claim that couldn't render, and a live-only LEARN bug — each fixed
  before merge.
- **The live verifications were run and checked by the team** — the demo arc, the LEARN
  quarantine/recover timings, and the P8 telemetry grading were measured against real SigNoz, not
  assumed.

## In short
AI was a force multiplier for implementation and review under human direction. The ideas, the
decisions, the reviews, and the accountability are the team's. The `JOURNAL.md` and the PR history
are the honest record of how it was built.
