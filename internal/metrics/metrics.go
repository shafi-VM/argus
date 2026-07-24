// Package metrics emits argusd's low-cardinality argus_* series and owns the
// health.Window that produces the Intelligence Health gauge. Labels are restricted
// to model/decision/status_class — never prompt, user, trace, or request ids.
package metrics

import (
	"context"

	"github.com/shafi-VM/argus/internal/health"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// Metrics is the argus_* emitter. Zero value is safe: a nil *Metrics no-ops, so the
// gateway can run un-instrumented in tests.
type Metrics struct {
	window   *health.Window
	requests metric.Int64Counter
	grounded metric.Int64Counter
}

// New registers the argus_* instruments and an observable gauge that reports the
// windowed Intelligence Health every export cycle (so the hero panel moves even
// between requests).
func New(w *health.Window) (*Metrics, error) {
	m := otel.Meter("argusd")
	requests, err := m.Int64Counter("argus_requests_total",
		metric.WithDescription("requests through argusd, by decision and HTTP status class"))
	if err != nil {
		return nil, err
	}
	grounded, err := m.Int64Counter("argus_grounded_total",
		metric.WithDescription("raw model answers that passed the Grounding Check"))
	if err != nil {
		return nil, err
	}
	gauge, err := m.Float64ObservableGauge("argus_intelligence_health_ratio",
		metric.WithDescription("windowed behavioral health in [0,1]; the cold-open RED number"))
	if err != nil {
		return nil, err
	}
	infra, err := m.Float64ObservableGauge("argus_infra_health_ratio",
		metric.WithDescription("windowed 2xx/all in [0,1]; the cold-open GREEN number (stays green while behavior rots)"))
	if err != nil {
		return nil, err
	}
	cost, err := m.Float64ObservableGauge("argus_cost_usd_per_request",
		metric.WithDescription("windowed average USD per behavioral request"))
	if err != nil {
		return nil, err
	}
	// One callback observes all three window-derived gauges each export cycle, so the
	// hero panels move even between requests.
	if _, err = m.RegisterCallback(func(_ context.Context, o metric.Observer) error {
		v, _ := w.Score(1.0)
		o.ObserveFloat64(gauge, v)
		o.ObserveFloat64(infra, w.InfraRatio(1.0))
		o.ObserveFloat64(cost, w.CostAvg())
		return nil
	}, gauge, infra, cost); err != nil {
		return nil, err
	}
	return &Metrics{window: w, requests: requests, grounded: grounded}, nil
}

// Record logs one request outcome.
//
// ctx MUST carry the active request span: the counter increments are recorded with
// it so the SDK can attach EXEMPLARS (metric datapoint -> trace), which is the whole
// basis of metric->trace click-through in SigNoz. Recording on context.Background()
// (as this did before) makes exemplars structurally impossible.
//
// behavioral=true iff the primary response was 2xx: an upstream error is an
// INFRASTRUCTURE failure, not behavioral drift, so it must never move the health
// score (red-team R2). primaryGrounded is the grounding result of the FIRST answer
// (pre-recovery) — Intelligence Health measures raw model quality.
func (m *Metrics) Record(ctx context.Context, model, decision, statusClass string, primaryGrounded, behavioral bool, costUSD float64) {
	if m == nil {
		return
	}
	m.requests.Add(ctx, 1, metric.WithAttributes(
		attribute.String("model", model),
		attribute.String("decision", decision),
		attribute.String("status_class", statusClass),
	))
	if behavioral && primaryGrounded {
		m.grounded.Add(ctx, 1, metric.WithAttributes(attribute.String("model", model)))
	}
	// ALWAYS record to the window (even on upstream errors) so infra health has a real
	// denominator; the window itself keeps behavioral vs infra separate (R2).
	m.window.Record(behavioral, primaryGrounded, 0, costUSD)
}
