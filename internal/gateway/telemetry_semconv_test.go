package gateway

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/shafi-VM/argus/internal/health"
	"github.com/shafi-VM/argus/internal/metrics"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

// This is the P8 evidence, made reproducible and regression-guarded: it asserts the
// exact semconv the checklist ticks. Break the token mapping / span kind / provider
// name / metric-label set tomorrow and this test goes red — which is the point.

func spanAttrs(s sdktrace.ReadOnlySpan) map[string]string {
	m := map[string]string{}
	for _, kv := range s.Attributes() {
		m[string(kv.Key)] = kv.Value.Emit()
	}
	return m
}

func findSpan(t *testing.T, sr *tracetest.SpanRecorder, name string, want int) []sdktrace.ReadOnlySpan {
	t.Helper()
	var out []sdktrace.ReadOnlySpan
	for _, s := range sr.Ended() {
		if s.Name() == name {
			out = append(out, s)
		}
	}
	if len(out) < want {
		t.Fatalf("want >=%d span(s) named %q, got %d", want, name, len(out))
	}
	return out
}

// TestGenAISemconvOnSpans pins the gen_ai.* attributes, span names, kinds and status
// that P8 claims — measured off real emitted spans, not read from source.
func TestGenAISemconvOnSpans(t *testing.T) {
	sr := tracetest.NewSpanRecorder()
	otel.SetTracerProvider(sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr)))

	// pass path
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"Flight AA42 departs."}}],` +
			`"usage":{"prompt_tokens":9,"completion_tokens":5,"total_tokens":14}}`))
	}))
	defer up.Close()
	post(New(up.URL))

	chat := findSpan(t, sr, "chat gpt-4o", 1)[0]
	if chat.SpanKind() != trace.SpanKindClient {
		t.Errorf("chat span kind = %v, want CLIENT", chat.SpanKind())
	}
	if chat.Status().Code != codes.Unset {
		t.Errorf("success span status = %v, want UNSET (not Ok)", chat.Status().Code)
	}
	a := spanAttrs(chat)
	checks := map[string]string{
		"gen_ai.provider.name":       "openai", // NOT the removed gen_ai.system
		"gen_ai.operation.name":      "chat",
		"gen_ai.request.model":       "gpt-4o",
		"gen_ai.usage.input_tokens":  "9", // NOT prompt_tokens
		"gen_ai.usage.output_tokens": "5", // NOT completion_tokens
	}
	for k, want := range checks {
		if a[k] != want {
			t.Errorf("span attr %s = %q, want %q", k, a[k], want)
		}
	}
	if _, ok := a["gen_ai.system"]; ok {
		t.Error("emitted gen_ai.system — it was removed from semconv; use gen_ai.provider.name")
	}
	if _, ok := a["gen_ai.usage.total_tokens"]; ok {
		t.Error("emitted gen_ai.usage.total_tokens — no such attribute in the spec")
	}
}

// TestErrorSemconv pins the upstream-error span shape (#25 + P8 error section).
func TestErrorSemconv(t *testing.T) {
	sr := tracetest.NewSpanRecorder()
	otel.SetTracerProvider(sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr)))
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(500)
		_, _ = w.Write([]byte(`{"error":{"message":"overloaded"}}`))
	}))
	defer up.Close()
	post(New(up.URL))

	s := findSpan(t, sr, "chat gpt-4o", 1)[0]
	if s.Status().Code != codes.Error {
		t.Errorf("upstream-error span status = %v, want Error", s.Status().Code)
	}
	if a := spanAttrs(s); a["error.type"] != "upstream_5xx" {
		t.Errorf("error.type = %q, want upstream_5xx (low-cardinality class)", a["error.type"])
	}
}

// TestRecoverySpanKind pins the recovery span as INTERNAL (an Argus action, not a
// GenAI operation) — the P8 identity/kind claim.
func TestRecoverySpanKind(t *testing.T) {
	sr := tracetest.NewSpanRecorder()
	otel.SetTracerProvider(sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr)))
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := readAll(r)
		c := "Flight UA99 departs."
		if strings.Contains(b, "REGROUND") {
			c = "Flight AA42 departs."
		}
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"` + c + `"}}]}`))
	}))
	defer up.Close()
	post(New(up.URL))

	rec := findSpan(t, sr, "argus.recovery.reground", 1)[0]
	if rec.SpanKind() != trace.SpanKindInternal {
		t.Errorf("recovery span kind = %v, want INTERNAL", rec.SpanKind())
	}
}

// TestMetricLabelsAreLowCardinality pins the argus_* metric label sets — the single
// most important metric-hygiene claim (no prompt/user/trace/request id on metrics).
func TestMetricLabelsAreLowCardinality(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	otel.SetMeterProvider(sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader)))
	m, err := metrics.New(health.NewWindow(health.Config{}))
	if err != nil {
		t.Fatal(err)
	}
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"Flight AA42."}}],"usage":{"prompt_tokens":9,"completion_tokens":5}}`))
	}))
	defer up.Close()
	g := New(up.URL)
	g.SetMetrics(m)
	post(g)

	allowed := map[string]map[string]bool{
		"argus_requests_total": {"model": true, "decision": true, "status_class": true},
		"argus_grounded_total": {"model": true},
	}
	var rm metricdata.ResourceMetrics
	if err := reader.Collect(context.Background(), &rm); err != nil {
		t.Fatal(err)
	}
	for _, sm := range rm.ScopeMetrics {
		for _, mm := range sm.Metrics {
			ok, tracked := allowed[mm.Name]
			if !tracked {
				continue
			}
			sum, isSum := mm.Data.(metricdata.Sum[int64])
			if !isSum {
				t.Errorf("%s is not a Sum (counter); got %T", mm.Name, mm.Data)
				continue
			}
			for _, dp := range sum.DataPoints {
				for _, kv := range dp.Attributes.ToSlice() {
					if !ok[string(kv.Key)] {
						t.Errorf("%s carries disallowed (high-cardinality?) label %q", mm.Name, kv.Key)
					}
				}
			}
		}
	}
}

// helpers ---------------------------------------------------------------------

func readAll(r *http.Request) (string, error) {
	b := make([]byte, r.ContentLength)
	_, err := r.Body.Read(b)
	return string(b), err
}
