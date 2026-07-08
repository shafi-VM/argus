# DECISIONS.md — Architecture Decision Records

Short, dated, irreversible-by-default. Supersede an ADR only with a newer ADR.

---

## ADR-0001 — Tech stack · 2026-07-07 · Accepted

- **Go** (`cmd/argusd` + `internal/*`) — Gateway/PREVENT, LEARN, recovery, webhook,
  and the Mission-Control page. **One binary.** Owner: **Eng A**.
- **Python** — demo agent, AI integrations, MCP investigation agent, Day-1 proofs.
  Owner: **Eng B**.
- **UI** — SigNoz native. Mission Control = a **static page served by `argusd`**. No React/Next build.
- **Infra** — Docker Compose; ClickHouse comes with SigNoz; GitHub Actions minimal (`build + vet + smoke`).

**Rejected:** Rust (velocity, no real gain in 7 days). Separate JS frontend (SigNoz is the UI).
**Fallback:** Python (FastAPI) gateway, kept in back pocket if we're behind by **Day 3**.
**Gate:** no Go gateway code until **A8 chosen + P1 green**.

**Why the two-language split pays off:** ownership divides cleanly along it
(Eng A = Go/infra = PREVENT+LEARN; Eng B = Python = agent+proofs) → parallel work, near-zero merge contention.
Closing pitch line it earns: *"Everything critical — including the control surface — is one Go binary. Python is used only where the AI ecosystem lives."*

---

## ADR-0002 — PREVENT detector · 2026-07-07 · Accepted (design; measurement = A8)

**Decision:** The inline PREVENT signal is a **deterministic grounding check** — compare the
agent's asserted facts against the tool/context payload it was *actually handed*.
**No LLM and no local ML model in the hot path.**

**Why it wins twice:**
- **Go-native** → keeps the critical path a single Go binary (serves ADR-0001).
- **Deterministic** → the demo can't flake; no false-positive blocking the *correct* answer on stage.
- **Verifiable** → "we checked the claim against its source," not "an LLM felt it was ungrounded."

**Consequences:**
- LLM-as-judge is allowed **only OFF the always-on path** (LEARN / investigation beat), never inline.
- A8's latency bar tightens to **< 50 ms p95** (local string/structured match, not a 300 ms model call).

**Non-goal:** general hallucination detection. We detect *ungrounded-vs-provided-context* — the
honest, demoable subset. We will say exactly that to judges.

---

## ADR-0003 — The SigNoz boundary (inviolable) · 2026-07-07 · Accepted

The **inline request path** and the **SigNoz-dependent path** are separated by an async
boundary that is **never crossed**:

- **PREVENT (Behavior Guard, inline):** decides locally, depends on **nothing external**.
  It EMITS to SigNoz fire-and-forget; it never BLOCKS on a read from SigNoz.
- **LEARN (Behavior Drift):** ALWAYS sources truth from SigNoz (`query_range` poll / alert),
  running **async, off the request path** — even though it lives in the same `argusd` binary.
- **Investigation:** ALWAYS uses SigNoz (MCP).

**Rule:** if any code makes a synchronous response wait on a SigNoz/ClickHouse read, it is a
bug — full stop. Revert it.

**Why this ADR exists:** on Day 4 someone will say *"let's just query SigNoz from the gateway
to decide inline."* That reintroduces the physics-lie we already killed (alert/ingestion
latency ≫ inline budget). This boundary is the guardrail.

**Refinement note:** the proposed wording was "the gateway never queries SigNoz." Corrected —
the `argusd` binary DOES host LEARN, which DOES read SigNoz. The invariant is about the
**inline path**, not the process. Phrased so no one can "technically comply" while breaking it.
