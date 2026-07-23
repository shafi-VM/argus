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
	return resource.New(ctx, resource.WithAttributes(
		attribute.String("service.name", serviceName),
		attribute.String("service.version", "0.1.0"),
		attribute.String("service.instance.id", instanceID),
		attribute.String("deployment.environment.name", getenv("ARGUS_ENV", "demo")),
	))
}

// Init configures the global tracer provider + propagator. Returns a shutdown func.
func Init(ctx context.Context, serviceName string) (func(context.Context) error, error) {
	exp, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(getenv("OTLP_ENDPOINT", "localhost:4317")),
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
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{}, propagation.Baggage{},
	))
	return tp.Shutdown, nil
}

// InitMetrics configures the global MeterProvider with a periodic OTLP exporter, so
// argus_* metrics (incl. the observable Intelligence Health gauge) ship to SigNoz
// every ExportInterval. Short interval so the hero panel moves on demo cadence.
func InitMetrics(ctx context.Context, serviceName string) (func(context.Context) error, error) {
	exp, err := otlpmetricgrpc.New(ctx,
		otlpmetricgrpc.WithEndpoint(getenv("OTLP_ENDPOINT", "localhost:4317")),
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
