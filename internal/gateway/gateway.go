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
	"time"

	"github.com/shafi-VM/argus/internal/grounding"
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

// Gateway proxies OpenAI-compatible chat completions to an upstream.
type Gateway struct {
	upstream string
	client   *http.Client
	tracer   trace.Tracer
}

// New returns a Gateway forwarding to upstream (e.g. http://127.0.0.1:9099).
func New(upstream string) *Gateway {
	return &Gateway{
		upstream: upstream,
		client:   &http.Client{Timeout: 30 * time.Second},
		tracer:   otel.Tracer("argusd/gateway"),
	}
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
	span.SetAttributes(
		attribute.String("gen_ai.operation.name", "chat"),
		attribute.String("gen_ai.provider.name", "openai"),
		attribute.String("gen_ai.request.model", model),
	)

	// The context we ground against travels IN the request. argusd never reads a
	// fixture file — that keeps the check honest for any caller, not just our agent.
	provided := grounding.ExtractContext(req.Messages)

	respBody, status, err := g.call(ctx, body)
	if err != nil {
		g.fail(w, span, err)
		return
	}

	// #25: an upstream non-2xx must NEVER read as a healthy, grounded pass. An error
	// body carries no entity claims, so the Grounding Check would happily call it
	// "grounded" and leave the span UNSET (= success). Mark it, skip the check, and
	// pass the status through unchanged. Telemetry correctness is product correctness.
	if status >= 400 {
		span.SetStatus(codes.Error, http.StatusText(status))
		span.SetAttributes(
			attribute.String("error.type", errorTypeFor(status)),
			attribute.Int("argus.upstream.status", status),
			attribute.String("argus.decision", "upstream_error"),
		)
		writeJSON(w, status, respBody)
		return
	}

	recordUsage(span, respBody)

	start := time.Now()
	res := grounding.Check(answerOf(respBody), provided)
	span.SetAttributes(
		attribute.Float64("argus.grounding.check_ms", msSince(start)),
		attribute.Bool("argus.grounding.skipped", res.Skipped),
		attribute.StringSlice("argus.grounding.claims", res.Claims),
	)

	if res.Grounded {
		span.SetAttributes(attribute.String("argus.decision", "pass"))
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
		span.SetAttributes(attribute.String("argus.decision", "refused"))
		writeJSON(w, http.StatusOK, refusal(model))
		return
	}

	rRes := grounding.Check(answerOf(rBody), provided)
	rspan.SetAttributes(attribute.Bool("argus.recovery.grounded", rRes.Grounded))
	rspan.End()

	if rRes.Grounded {
		span.SetAttributes(attribute.String("argus.decision", "recovered"))
		writeJSON(w, rStatus, rBody) // the caller only ever sees the corrected answer
		return
	}

	span.SetAttributes(attribute.String("argus.decision", "refused"))
	writeJSON(w, http.StatusOK, refusal(model))
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
