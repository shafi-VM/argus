// Package telemetry wires argusd to SigNoz over OTLP and sets up W3C context
// propagation so cross-service traces (agent -> gateway) link into one waterfall.
package telemetry

import (
	"context"
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

// Init configures the global tracer provider + propagator. Returns a shutdown func.
func Init(ctx context.Context, serviceName string) (func(context.Context) error, error) {
	exp, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(getenv("OTLP_ENDPOINT", "localhost:4317")),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		return nil, err
	}

	res, err := resource.New(ctx, resource.WithAttributes(
		attribute.String("service.name", serviceName),
		attribute.String("service.version", "0.1.0"),
	))
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
	res, err := resource.New(ctx, resource.WithAttributes(
		attribute.String("service.name", serviceName),
		attribute.String("service.version", "0.1.0"),
	))
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
