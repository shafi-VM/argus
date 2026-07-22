// Package gateway is the Behavior Guard — argusd's inline PREVENT path.
//
// It proxies OpenAI-compatible chat completions and, before the answer reaches the
// caller, runs a deterministic Grounding Check. An ungrounded answer is BLOCKED and
// re-grounded (retried with an explicit instruction), so the caller only ever sees
// a supported answer.
//
// ADR-0003: nothing here blocks on SigNoz. The decision is local and in-process;
// telemetry is emitted fire-and-forget.
package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/shafi-VM/argus/internal/grounding"
	"github.com/shafi-VM/argus/internal/metrics"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

const (
	// regroundInstruction mirrors what an operator would do: tell the model its
	// claim was unsupported and make it answer from the retrieved context only.
	regroundInstruction = "REGROUND: your previous answer cited data that is not present in " +
		"RETRIEVED_CONTEXT. Answer again using ONLY the retrieved context."

	// refusalText is served if even a re-grounded answer stays unsupported.
	// Refusing beats serving a hallucination.
	refusalText = "I couldn't find that in the retrieved booking data, so I'd rather not guess."
)

// Gateway proxies OpenAI-compatible chat completions to an upstream. It also
// implements learn.Actuator: LEARN quarantines a model by installing a reroute here,
// which is how a windowed decision changes live behavior on the request path.
type Gateway struct {
	upstream string
	client   *http.Client
	tracer   trace.Tracer
	metrics  *metrics.Metrics // nil-safe: no-op when unset (tests)

	mu      sync.RWMutex
	reroute map[string]string // quarantined model -> fallback model
}

// New returns a Gateway forwarding to upstream (e.g. http://127.0.0.1:9099).
func New(upstream string) *Gateway {
	return &Gateway{
		upstream: upstream,
		client:   &http.Client{Timeout: 30 * time.Second},
		tracer:   otel.Tracer("argusd/gateway"),
		reroute:  map[string]string{},
	}
}

// SetMetrics attaches the argus_* emitter (from main, once telemetry is up).
func (g *Gateway) SetMetrics(m *metrics.Metrics) { g.metrics = m }

// Quarantine and Recover implement learn.Actuator. Both are idempotent.
func (g *Gateway) Quarantine(model, fallback string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.reroute[model] = fallback
}

func (g *Gateway) Recover(model string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	delete(g.reroute, model)
}

func (g *Gateway) fallbackFor(model string) (string, bool) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	fb, ok := g.reroute[model]
	return fb, ok
}

type chatRequest struct {
	Model    string              `json:"model"`
	Messages []grounding.Message `json:"messages"`
}

// ChatCompletions is the PREVENT path: proxy -> check -> (block -> re-ground).
// The CLIENT span's parent is the caller's traceparent, so agent + gateway +
// recovery render as ONE trace in SigNoz.
func (g *Gateway) ChatCompletions(w http.ResponseWriter, r *http.Request) {
	ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))
	body, _ := io.ReadAll(r.Body)

	var req chatRequest
	_ = json.Unmarshal(body, &req)
	model := req.Model
	if model == "" {
		model = "unknown"
	}

	ctx, span := g.tracer.Start(ctx, "chat "+model, trace.WithSpanKind(trace.SpanKindClient))
	defer span.End()

	// LEARN reroute: if this model is quarantined, send the call to its fallback.
	// This is how a WINDOWED decision (quarantine) changes LIVE behavior.
	if fb, ok := g.fallbackFor(model); ok {
		span.SetAttributes(attribute.String("argus.rerouted_from", model))
		body = rewriteModel(body, fb)
		model = fb
	}
	span.SetAttributes(
		attribute.String("gen_ai.operation.name", "chat"),
		attribute.String("gen_ai.provider.name", "openai"),
		attribute.String("gen_ai.request.model", model),
	)

	// Outcome captured for the argus_* metrics; recorded exactly once on return.
	var (
		decision        string
		statusClass     = "2xx"
		primaryGrounded bool
		behavioral      bool
		costUSD         float64
	)
	defer func() {
		g.metrics.Record(model, decision, statusClass, primaryGrounded, behavioral, costUSD)
	}()

	// The context we ground against travels IN the request. argusd never reads a
	// fixture file — that keeps the check honest for any caller, not just our agent.
	provided := grounding.ExtractContext(req.Messages)

	respBody, status, err := g.call(ctx, body)
	if err != nil {
		decision, statusClass = "transport_error", "5xx"
		g.fail(w, span, err)
		return
	}
	statusClass = classOf(status)
	behavioral = status < 400 // an upstream error is infra, not behavior (R2)

	// #25: an upstream non-2xx must NEVER read as a healthy, grounded pass. An error
	// body carries no entity claims, so the Grounding Check would happily call it
	// "grounded" and leave the span UNSET (= success). Mark it, skip the check, and
	// pass the status through unchanged. Telemetry correctness is product correctness.
	if status >= 400 {
		decision = "upstream_error"
		span.SetStatus(codes.Error, http.StatusText(status))
		span.SetAttributes(
			attribute.String("error.type", errorTypeFor(status)),
			attribute.Int("argus.upstream.status", status),
			attribute.String("argus.decision", decision),
		)
		writeJSON(w, status, respBody)
		return
	}

	recordUsage(span, respBody)
	costUSD = costOf(respBody)

	start := time.Now()
	res := grounding.Check(answerOf(respBody), provided)
	primaryGrounded = res.Grounded // Intelligence Health measures RAW model quality
	span.SetAttributes(
		attribute.Float64("argus.grounding.check_ms", msSince(start)),
		attribute.Bool("argus.grounding.skipped", res.Skipped),
		attribute.StringSlice("argus.grounding.claims", res.Claims),
	)

	if res.Grounded {
		decision = "pass"
		span.SetAttributes(attribute.String("argus.decision", decision))
		writeJSON(w, status, respBody)
		return
	}

	// ---- PREVENT: block the ungrounded answer, then recover -----------------
	span.AddEvent("argus.behavior.blocked", trace.WithAttributes(
		attribute.String("argus.signal", "grounding"),
		attribute.StringSlice("argus.grounding.unsupported", res.Unsupported),
	))

	rctx, rspan := g.tracer.Start(ctx, "argus.recovery.reground",
		trace.WithSpanKind(trace.SpanKindInternal))
	rspan.SetAttributes(attribute.StringSlice("argus.grounding.unsupported", res.Unsupported))

	rBody, rStatus, rErr := g.call(rctx, withReground(body))
	if rErr != nil {
		rspan.RecordError(rErr)
		rspan.SetStatus(codes.Error, rErr.Error())
		rspan.End()
		decision = "refused"
		span.SetAttributes(attribute.String("argus.decision", decision))
		writeJSON(w, http.StatusOK, refusal(model))
		return
	}

	// #25 (recovery leg): a non-2xx on the RE-GROUND retry must never read as a
	// healthy "recovered". Its error body has no entities, so the Grounding Check
	// would call it grounded. Mark it, and serve a safe refusal — never the raw
	// upstream error, never a green span.
	if rStatus >= 400 {
		rspan.SetStatus(codes.Error, http.StatusText(rStatus))
		rspan.SetAttributes(
			attribute.String("error.type", errorTypeFor(rStatus)),
			attribute.Int("argus.upstream.status", rStatus),
		)
		rspan.End()
		decision = "upstream_error"
		span.SetStatus(codes.Error, http.StatusText(rStatus))
		span.SetAttributes(attribute.String("argus.decision", decision))
		writeJSON(w, http.StatusOK, refusal(model))
		return
	}

	rRes := grounding.Check(answerOf(rBody), provided)
	rspan.SetAttributes(attribute.Bool("argus.recovery.grounded", rRes.Grounded))
	rspan.End()

	if rRes.Grounded {
		decision = "recovered"
		span.SetAttributes(attribute.String("argus.decision", decision))
		writeJSON(w, rStatus, rBody) // the caller only ever sees the corrected answer
		return
	}

	decision = "refused"
	span.SetAttributes(attribute.String("argus.decision", decision))
	writeJSON(w, http.StatusOK, refusal(model))
}

// rewriteModel replaces the "model" field, preserving all other request fields.
func rewriteModel(body []byte, model string) []byte {
	var m map[string]any
	if json.Unmarshal(body, &m) != nil {
		return body
	}
	m["model"] = model
	out, err := json.Marshal(m)
	if err != nil {
		return body
	}
	return out
}

func classOf(status int) string {
	switch {
	case status >= 500:
		return "5xx"
	case status >= 400:
		return "4xx"
	case status >= 200 && status < 300:
		return "2xx"
	default:
		return "other"
	}
}

// costOf estimates request cost from token usage — real enough for the cost penalty
// without a pricing table (grounding, not cost, is the primary health driver).
func costOf(respBody []byte) float64 {
	var r struct {
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
		} `json:"usage"`
	}
	if json.Unmarshal(respBody, &r) != nil {
		return 0
	}
	const perToken = 0.000002
	return float64(r.Usage.PromptTokens+r.Usage.CompletionTokens) * perToken
}

func (g *Gateway) call(ctx context.Context, body []byte) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		g.upstream+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(req.Header))

	resp, err := g.client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	return b, resp.StatusCode, err
}

func (g *Gateway) fail(w http.ResponseWriter, span trace.Span, err error) {
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
	http.Error(w, err.Error(), http.StatusBadGateway)
}

// errorTypeFor keeps error.type LOW-CARDINALITY: a class, never a message.
func errorTypeFor(status int) string {
	switch {
	case status >= 500:
		return "upstream_5xx"
	case status >= 400:
		return "upstream_4xx"
	default:
		return ""
	}
}

func msSince(t time.Time) float64 { return float64(time.Since(t).Microseconds()) / 1000.0 }

func answerOf(b []byte) string {
	var r struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if json.Unmarshal(b, &r) != nil || len(r.Choices) == 0 {
		return ""
	}
	return r.Choices[0].Message.Content
}

// withReground appends the re-ground instruction, preserving every other field of
// the original request (temperature, tools, ...) by round-tripping the raw JSON.
func withReground(body []byte) []byte {
	var m map[string]any
	if json.Unmarshal(body, &m) != nil {
		return body
	}
	msgs, ok := m["messages"].([]any)
	if !ok {
		return body
	}
	m["messages"] = append(msgs, map[string]any{"role": "system", "content": regroundInstruction})
	out, err := json.Marshal(m)
	if err != nil {
		return body
	}
	return out
}

func refusal(model string) []byte {
	b, _ := json.Marshal(map[string]any{
		"id":     "chatcmpl-argus-refusal",
		"object": "chat.completion",
		"model":  model,
		"choices": []any{map[string]any{
			"index":         0,
			"finish_reason": "stop",
			"message":       map[string]any{"role": "assistant", "content": refusalText},
		}},
	})
	return b
}

func recordUsage(span trace.Span, respBody []byte) {
	var r struct {
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
		} `json:"usage"`
	}
	if json.Unmarshal(respBody, &r) != nil {
		return
	}
	// No gen_ai.usage.total_tokens — it does not exist in the spec; derive at query time.
	span.SetAttributes(
		attribute.Int("gen_ai.usage.input_tokens", r.Usage.PromptTokens),
		attribute.Int("gen_ai.usage.output_tokens", r.Usage.CompletionTokens),
	)
}

func writeJSON(w http.ResponseWriter, status int, body []byte) {
	w.Header().Set("Content-Type", "application/json")
	if status == 0 {
		status = http.StatusOK
	}
	w.WriteHeader(status)
	_, _ = w.Write(body)
}
