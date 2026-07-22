// Package learn is the LEARN control loop.
//
// It reads the windowed Intelligence Health FROM SigNoz — never computed locally —
// because "we read the windowed behavioral truth from SigNoz and act on it" is the
// entire point (ADR-0003). It feeds that value to the health.Governor and executes
// quarantine/reroute. Every step is a span, so the decision itself is observable
// (the observer is observed).
package learn

import (
	"context"
	"time"

	"github.com/shafi-VM/argus/internal/health"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// SigNozClient reads the current windowed Intelligence Health from SigNoz.
// age is how old the newest data point is, so the poller can refuse to act on
// stale truth (red-team R5).
type SigNozClient interface {
	LatestHealth(ctx context.Context) (value float64, age time.Duration, err error)
}

// Actuator applies LEARN's decision. Implementations must be idempotent.
type Actuator interface {
	Quarantine(model, fallback string)
	Recover(model string)
}

// Poller runs the LEARN loop. Construct with New, then Run (or drive Tick in tests).
type Poller struct {
	client   SigNozClient
	gov      *health.Governor
	act      Actuator
	model    string
	fallback string
	interval time.Duration
	maxAge   time.Duration
	tracer   trace.Tracer
}

// Config for the LEARN poller. Interval/MaxAge default to demo-tuned values.
type Config struct {
	Client   SigNozClient
	Governor *health.Governor
	Actuator Actuator
	Model    string
	Fallback string
	Interval time.Duration // poll cadence
	MaxAge   time.Duration // reject windows whose newest point is older than this
}

func New(c Config) *Poller {
	if c.Interval == 0 {
		c.Interval = 2 * time.Second
	}
	if c.MaxAge == 0 {
		// Normal trace ingestion lag is ~13s; the query window (lookback) is 30s.
		// MaxAge sits between them: hold when SigNoz falls >25s behind (genuine
		// staleness) without spuriously holding on normal-lag jitter. This guard was
		// inert while SigNoz.groundingRate hardcoded age=0 (review #1) — now live.
		c.MaxAge = 25 * time.Second
	}
	if c.Governor == nil {
		c.Governor = health.NewGovernor(0.5, 0.7, 2)
	}
	return &Poller{
		client: c.Client, gov: c.Governor, act: c.Actuator,
		model: c.Model, fallback: c.Fallback,
		interval: c.Interval, maxAge: c.MaxAge,
		tracer: otel.Tracer("argusd/learn"),
	}
}

// Run polls until ctx is cancelled.
func (p *Poller) Run(ctx context.Context) {
	t := time.NewTicker(p.interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			p.Tick(ctx)
		}
	}
}

// Tick performs one poll cycle: query SigNoz -> evaluate -> (maybe) act. Returns
// the action taken (usually None). Exported so the loop is unit-testable.
func (p *Poller) Tick(ctx context.Context) health.Action {
	ctx, span := p.tracer.Start(ctx, "argus.learn.evaluate", trace.WithSpanKind(trace.SpanKindInternal))
	defer span.End()

	value, age, err := p.query(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return health.None
	}

	// Never act on stale truth: a wrong action on old data is worse than no action.
	if age > p.maxAge {
		span.SetAttributes(
			attribute.String("argus.learn.skip", "stale"),
			attribute.Float64("argus.learn.age_s", age.Seconds()),
		)
		return health.None
	}

	span.SetAttributes(
		attribute.Float64("argus.intelligence_health", value),
		attribute.Float64("argus.learn.age_s", age.Seconds()),
		attribute.String("argus.learn.phase", p.gov.Phase().String()),
	)

	act := p.gov.Observe(value)
	span.SetAttributes(attribute.String("argus.learn.action", act.String()))
	switch act {
	case health.Quarantine:
		p.quarantine(ctx)
	case health.Recover:
		p.recover(ctx)
	}
	return act
}

func (p *Poller) query(ctx context.Context) (float64, time.Duration, error) {
	ctx, span := p.tracer.Start(ctx, "argus.learn.query_window", trace.WithSpanKind(trace.SpanKindClient))
	defer span.End()
	span.SetAttributes(
		attribute.String("db.system", "signoz"),
		// LEARN reads the windowed decision from TRACE spans (argus.decision), not the
		// intelligence_health metric gauge — the trace signal is ~13s fresh vs ~60s.
		attribute.String("argus.learn.source", "traces:argus.decision"),
	)
	v, age, err := p.client.LatestHealth(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
	return v, age, err
}

func (p *Poller) quarantine(ctx context.Context) {
	_, span := p.tracer.Start(ctx, "argus.learn.quarantine", trace.WithSpanKind(trace.SpanKindInternal))
	defer span.End()
	span.SetAttributes(
		attribute.String("argus.model", p.model),
		attribute.String("argus.fallback", p.fallback),
	)
	p.act.Quarantine(p.model, p.fallback)
}

func (p *Poller) recover(ctx context.Context) {
	_, span := p.tracer.Start(ctx, "argus.learn.recover", trace.WithSpanKind(trace.SpanKindInternal))
	defer span.End()
	span.SetAttributes(attribute.String("argus.model", p.model))
	p.act.Recover(p.model)
}
