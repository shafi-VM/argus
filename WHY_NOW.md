# WHY NOW — the case for Argus

_The narrative spine. README, demo narration, Devpost, and blog all derive from this page._

## Why does AI need this now?
Agents crossed from demos into production in the last ~18 months. They no longer just generate
text — they take **autonomous actions**: calling tools, spending money, writing to systems. A
wrong action now has real cost and real blast radius. And the failures are **behavioral, not
infrastructural**: the server is healthy, CPU is flat, every HTTP call returns 200 — while the
agent confidently invents a fact, loops forever, or gets manipulated into leaking data. The last
decade of monitoring was built to watch **infrastructure**. It is structurally blind to this new
failure class. Today the recovery mechanism for behavioral failure is *a human noticing* — which
does not scale to fleets of agents.

## Why didn't this exist two years ago?
Three things weren't true yet:
1. **The pain wasn't acute.** Agents were toys; nobody had one in the loop of real money or real
   user data at scale.
2. **The signals weren't standard.** There was no agreed, vendor-neutral way to say "this LLM call
   cost $X, used N tokens, called tool T, was grounded/ungrounded." You can't build a control loop
   on signals that don't exist.
3. **The substrate was proprietary.** Early LLM tooling was closed SaaS. There was no open backend
   you could put in your own VPC and build a control loop on.

## Why is OpenTelemetry the enabler?
OTel's **GenAI semantic conventions** turned agent behavior into standard, portable telemetry for
the first time — "the agent hallucinated / looped / spent" is now a span and a metric *anyone* can
read. OTel is already **in the request path** (SDKs, collectors), making it the natural place to
both *sense* behavior and — through a gateway — *act* on it. And because it's vendor-neutral and
self-hostable, the prompts (which carry user/PHI data) never have to leave your infrastructure.

## Why is SigNoz the right foundation?
SigNoz is **OTel-native and open-source**: no proprietary silo, self-hostable, your prompt data
stays in your VPC. It **unifies traces, metrics, logs, and alerts in one backend** — exactly the
substrate a control loop needs: **record** (traces), **detect over a window** (metrics + anomaly
alerts), **investigate** (MCP). The SigNoz MCP even makes telemetry **queryable by agents**,
turning the observability backend from a wall of dashboards into a *decision surface*. We don't
compete with SigNoz — we make it the nervous system of a system that finally **acts**.

## The inevitability
Every wave of software got a control plane once it hit production scale: servers got Kubernetes,
networks got service meshes, data pipelines got orchestration. **Agents are hitting production
right now, and they have no control plane.** Argus is that layer — and it is only buildable *now*,
because OpenTelemetry finally made agent behavior observable in the open.

> If you can't observe your AI agents, you don't own them. **Argus is how you own them.**
