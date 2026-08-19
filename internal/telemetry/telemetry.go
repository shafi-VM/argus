// Package telemetry wires argusd to SigNoz over OTLP and sets up W3C context
// propagation so cross-service traces (agent -> gateway) link into one waterfall.
package telemetry

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"os"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// instanceID is one stable id for this process, shared by the trace and metric
// resources so SigNoz correlates their signals to the same argusd instance
// (service.instance.id, per OTel resource semconv).
var instanceID = newInstanceID()

func newInstanceID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "argusd-unknown"
	}
	return hex.EncodeToString(b)
}

// newResource builds the OTel Resource shared by traces and metrics. A complete,
// identical resource on both signals is what a SigNoz engineer looks for first:
// service.name/version, a per-instance id, and a low-cardinality environment.
func newResource(ctx context.Context, serviceName string) (*resource.Resource, error) {
	return resource.New(ctx,
		resource.WithTelemetrySDK(), // telemetry.sdk.{name,language,version}
		resource.WithAttributes(
			attribute.String("service.name", serviceName),
			attribute.String("service.version", "0.1.0"),
			attribute.String("service.instance.id", instanceID),
			attribute.String("deployment.environment.name", getenv("ARGUS_ENV", "demo")),
		))
}

// otlpTarget returns the OTLP endpoint and whether export is enabled. Semantics:
// OTLP_ENDPOINT unset -> default localhost:4317 (local dev against host SigNoz);
// set but EMPTY -> export disabled (the zero-dependency demo tier — PREVENT and
// Mission Control run with no backend at all); set non-empty -> that endpoint.
func otlpTarget() (string, bool) {
	v, ok := os.LookupEnv("OTLP_ENDPOINT")
	if !ok {
		return "localhost:4317", true
	}
	return v, v != ""
}

// Init configures the global tracer provider + propagator. Returns a shutdown func.
func Init(ctx context.Context, serviceName string) (func(context.Context) error, error) {
	endpoint, export := otlpTarget()
	// Propagation always on: cross-service trace headers still parse/inject even
	// when we're not exporting, so nothing downstream changes behavior by tier.
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{}, propagation.Baggage{},
	))
	if !export {
		// No OTLP target: leave the global no-op TracerProvider in place. Spans
		// become cheap no-ops; argusd needs no SigNoz to run its PREVENT reflex.
		return func(context.Context) error { return nil }, nil
	}

	exp, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(endpoint),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		return nil, err
	}

	res, err := newResource(ctx, serviceName)
	if err != nil {
		return nil, err
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)
	return tp.Shutdown, nil
}

// InitMetrics configures the global MeterProvider with a periodic OTLP exporter, so
// argus_* metrics (incl. the observable Intelligence Health gauge) ship to SigNoz
// every ExportInterval. Short interval so the hero panel moves on demo cadence.
func InitMetrics(ctx context.Context, serviceName string) (func(context.Context) error, error) {
	endpoint, export := otlpTarget()
	if !export {
		// No OTLP target: leave the global no-op MeterProvider in place. The health
		// gauge and argus_* counters become no-ops instead of failing to export.
		return func(context.Context) error { return nil }, nil
	}
	exp, err := otlpmetricgrpc.New(ctx,
		otlpmetricgrpc.WithEndpoint(endpoint),
		otlpmetricgrpc.WithInsecure(),
	)
	if err != nil {
		return nil, err
	}
	res, err := newResource(ctx, serviceName)
	if err != nil {
		return nil, err
	}
	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exp, sdkmetric.WithInterval(5*time.Second))),
		sdkmetric.WithResource(res),
	)
	otel.SetMeterProvider(mp)
	return mp.Shutdown, nil
}

func getenv(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
