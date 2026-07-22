package gateway

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

// recorder installs a GLOBAL tracer provider. Safe because these tests run
// sequentially and New() is called after the provider is set — do NOT add
// t.Parallel() to gateway tests without giving each its own provider.
func recorder(t *testing.T) *tracetest.SpanRecorder {
	t.Helper()
	sr := tracetest.NewSpanRecorder()
	otel.SetTracerProvider(sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr)))
	return sr
}

func spanNamed(t *testing.T, sr *tracetest.SpanRecorder, name string) sdktrace.ReadOnlySpan {
	t.Helper()
	for _, s := range sr.Ended() {
		if s.Name() == name {
			return s
		}
	}
	t.Fatalf("no span named %q (recorded %d spans)", name, len(sr.Ended()))
	return nil
}

func attrs(s sdktrace.ReadOnlySpan) map[string]string {
	m := map[string]string{}
	for _, kv := range s.Attributes() {
		m[string(kv.Key)] = kv.Value.Emit()
	}
	return m
}

const reqBody = `{"model":"gpt-4o","messages":[` +
	`{"role":"system","content":"RETRIEVED_CONTEXT: {\"flight\":\"AA42\"}"},` +
	`{"role":"user","content":"book it"}]}`

func post(g *Gateway) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(reqBody))
	rec := httptest.NewRecorder()
	g.ChatCompletions(rec, req)
	return rec
}

// Regression for #25: an upstream 5xx must never read as a healthy, grounded pass.
// Telemetry correctness is product correctness — a failure that looks green is P0.
func TestUpstreamErrorIsNotAHealthyPass(t *testing.T) {
	sr := recorder(t)
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":{"message":"upstream model overloaded"}}`))
	}))
	defer up.Close()

	rec := post(New(up.URL))
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("caller status = %d, want 500 passed through", rec.Code)
	}

	s := spanNamed(t, sr, "chat gpt-4o")
	a := attrs(s)
	if got := a["argus.decision"]; got != "upstream_error" {
		t.Errorf("argus.decision = %q, want upstream_error", got)
	}
	if s.Status().Code != codes.Error {
		t.Errorf("span status = %v, want Error", s.Status().Code)
	}
	if got := a["error.type"]; got != "upstream_5xx" {
		t.Errorf("error.type = %q, want upstream_5xx", got)
	}
	if _, ran := a["argus.grounding.claims"]; ran {
		t.Error("Grounding Check ran on an error body; it must be skipped")
	}
}

// Guards the pass-through path.
func TestGroundedAnswerPasses(t *testing.T) {
	sr := recorder(t)
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant",` +
			`"content":"Flight AA42 departs SFO 09:15."}}],` +
			`"usage":{"prompt_tokens":9,"completion_tokens":5}}`))
	}))
	defer up.Close()

	rec := post(New(up.URL))
	if rec.Code != http.StatusOK {
		t.Errorf("caller status = %d, want 200", rec.Code)
	}
	if got := attrs(spanNamed(t, sr, "chat gpt-4o"))["argus.decision"]; got != "pass" {
		t.Errorf("argus.decision = %q, want pass", got)
	}
}

// Guards THE money moment: an unsupported claim is blocked, re-grounded, and the
// caller never sees the hallucination.
func TestUngroundedIsBlockedAndRecovered(t *testing.T) {
	sr := recorder(t)
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		content := "Flight UA99 departs SFO 06:00." // unsupported by the context
		if strings.Contains(string(b), "REGROUND") {
			content = "Flight AA42 departs SFO 09:15." // corrected on re-ground
		}
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"` +
			content + `"}}]}`))
	}))
	defer up.Close()

	rec := post(New(up.URL))
	if strings.Contains(rec.Body.String(), "UA99") {
		t.Error("caller received the hallucination — PREVENT failed")
	}
	if !strings.Contains(rec.Body.String(), "AA42") {
		t.Error("caller did not receive the corrected answer")
	}
	if got := attrs(spanNamed(t, sr, "chat gpt-4o"))["argus.decision"]; got != "recovered" {
		t.Errorf("argus.decision = %q, want recovered", got)
	}
	spanNamed(t, sr, "argus.recovery.reground") // recovery must be observable
}

// Regression (Abhishek's #26 finding): a 5xx on the RE-GROUND retry must NOT read
// as "recovered", and the caller must not receive the raw upstream error.
func TestRecovery5xxIsNotAHealthyRecovered(t *testing.T) {
	sr := recorder(t)
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		if strings.Contains(string(b), "REGROUND") {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error":{"message":"overloaded during re-ground"}}`))
			return
		}
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant",` +
			`"content":"Flight UA99 departs SFO 06:00."}}]}`)) // primary: ungrounded
	}))
	defer up.Close()

	rec := post(New(up.URL))
	body := rec.Body.String()
	if strings.Contains(body, "overloaded") || strings.Contains(body, "UA99") {
		t.Errorf("caller received raw upstream error or hallucination: %q", body)
	}
	s := attrs(spanNamed(t, sr, "chat gpt-4o"))
	if s["argus.decision"] == "recovered" {
		t.Error("recovery 5xx labelled as recovered — the #25 class on the money path")
	}
	if s["argus.decision"] != "upstream_error" {
		t.Errorf("argus.decision = %q, want upstream_error", s["argus.decision"])
	}
	if spanNamed(t, sr, "chat gpt-4o").Status().Code != codes.Error {
		t.Error("main span not Error on recovery 5xx")
	}
}

// The refusal path: primary AND re-ground both ungrounded -> refuse, never serve UA99.
func TestUnrecoverableIsRefusedNotServed(t *testing.T) {
	sr := recorder(t)
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant",` +
			`"content":"Flight UA99 departs SFO 06:00."}}]}`)) // always ungrounded
	}))
	defer up.Close()

	rec := post(New(up.URL))
	if strings.Contains(rec.Body.String(), "UA99") {
		t.Error("served the hallucination when recovery failed")
	}
	if got := attrs(spanNamed(t, sr, "chat gpt-4o"))["argus.decision"]; got != "refused" {
		t.Errorf("argus.decision = %q, want refused", got)
	}
}
