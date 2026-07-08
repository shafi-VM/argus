# P9 — Judge Install (clean VM)

**Question:** if a SigNoz engineer clones this repo cold, do they have Argus running in
**< 10 minutes with zero manual edits**?

This is the experience every OSS judge — and every post-hackathon visitor — actually has.
Optimize for it.

## Procedure (run on a FRESH VM / clean machine)
1. Fresh Ubuntu/macOS VM, Docker installed, nothing else.
2. `git clone <repo>`
3. `cd argus && docker compose up`
4. Wait. Watch. Touch nothing.

## Success criteria (all must be true)
- [ ] `docker compose up` is the ONLY command needed
- [ ] No manual file edits; no secret required for the demo path (offline/replay)
- [ ] SigNoz comes up healthy
- [ ] The **Intelligence Health** hero dashboard appears, populated
- [ ] The PREVENT + LEARN beats run via the Mission Control page
- [ ] Total time from `clone` → "hero dashboard visible" < **10 minutes**

**If any step needs a human, it's a bug.** Log it in `issues.md` and fix it before we ever say
"everything you saw is open source."
