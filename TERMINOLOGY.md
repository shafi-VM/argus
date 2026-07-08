# TERMINOLOGY — frozen vocabulary

We use the same words **everywhere**: code, commits, dashboards, README, pitch, Devpost. Consistent
vocabulary is part of the product (Kubernetes, Docker, Terraform, Prometheus all did this). The
"NOT" synonyms are **banned** — don't let them creep in.

## Capabilities — the two superpowers (verbs)
- **PREVENT** — the inline reflex. NOT: blocking, filtering, guarding-generic.
- **LEARN** — the windowed adaptation. NOT: analytics, monitoring, observability-generic.

## Components — what delivers them (nouns)
- **Behavior Guard** — delivers PREVENT (inline, in `argusd`). NOT: gateway, middleware, interceptor, filter, proxy.
- **Behavior Drift** — delivers LEARN (windowed, reads SigNoz). NOT: analytics engine, trend engine, monitor.
- **Recovery** — reground + retry + reroute. NOT: retry logic, fallback, error handling.
- **Investigation** — the SigNoz-MCP RCA beat. NOT: debugging, log search.
- **Chaos** — deterministic fault injection (the "vaccination"). NOT: testing, fuzzing.

## Signals & surfaces
- **Intelligence Health** — the headline metric/dashboard (behavioral vs infra health). NOT: LLM metrics, model stats.
- **Grounding Check** — the deterministic PREVENT detector (claims vs provided context). NOT: hallucination detector, groundedness model.
- **Quarantine** — isolating a bad model/agent. NOT: disable. (*Kill* is reserved for the loop-breaker only.)

## Framing (marketing skin — sparingly, NEVER in code)
"**Immune System for AI Agents**" is the *marketing* line; the immune metaphor (antibody,
vaccination…) opens the door in a pitch. The engineering names above are canonical.
Never name a function `antibody()`.

## The one-liner (verbatim, everywhere)
> Portkey guards one call. OpenLIT watches. **Argus closes the loop.**
