# KILL_LIST.md — Things we will NOT build

> Winning teams don't ask "what else can we build?"
> They ask "what can we delete?"

Every time an idea here resurfaces — from anyone, including the Tech Lead —
it is **rejected on sight**. No debate. It is already decided.

- ❌ Custom dashboard framework — **SigNoz's UI is our UI.**
- ❌ Multi-agent orchestration support
- ❌ 5 AI-framework integrations — **ship ONE end-to-end** (OpenAI Agents SDK or LangGraph) + "any OTel app."
- ❌ Fine-grained RBAC / auth / multi-tenant admin
- ❌ AI-generated RCA prose as a core feature
- ❌ Complex policy DSL / rules engine
- ❌ Plugin marketplace / extensibility API
- ❌ Kubernetes operator / Helm chart
- ❌ Fancy bespoke frontend (only a thin god-mode control strip)
- ❌ Tool-reputation engine (deferred; not in the two superpowers)
- ❌ Chaos *suite* — one deterministic **chaos button** only
- ❌ Semantic-diff / time-travel debugger / replay UI
- ❌ Self-hosted model training / fine-tuning
- ❌ Anything that requires forking or patching SigNoz internals

### The test
If a proposed task does not **retire an assumption** (see `ASSUMPTIONS.md`) or
**directly serve one of the two superpowers** (see `VISION.md`), it does not get built.
