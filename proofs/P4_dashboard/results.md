# P4 — Results

**Question (A5):** does the hero dashboard provision from JSON and render from data?

**The dashboard is a contract** (`dashboard.json`): every panel ↔ exactly one metric, low-cardinality labels only. Build the panels to those metric names so the mock emitter AND the real Go/Python instrumentation target one schema — zero rework later.

## Steps
1. Build the "Intelligence Health" dashboard in the SigNoz UI (panels listed in `dashboard.json`).
2. Feed it **mocked OTel metrics** (low-cardinality) so panels populate.
3. **Export** the dashboard JSON → overwrite `dashboard.json` here.
4. Import it into a **fresh** SigNoz instance → confirm every panel renders.
5. Screenshot each panel → `screenshots/`.

## Verdict
- Exported JSON re-imports cleanly on a fresh instance → ☐ PASS / ☐ FAIL
- All panels render from seeded metrics (no blank panels) → ☐ PASS / ☐ FAIL
- The 🟢 Infra vs 🔴 Intelligence split reads clearly on a projector → ☐ yes / ☐ no

**This is the pitch artifact.** Build it Day 1 from mocks; it de-risks *and* becomes the demo.

→ update `../SUMMARY.md`
