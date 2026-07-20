// Package gateway is argusd's request path: a drop-in OpenAI-compatible proxy
// that wraps every LLM call in a gen_ai CLIENT span. PREVENT logic lands here later.
package gateway

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
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

// ChatCompletions proxies POST /v1/chat/completions. The CLIENT span's parent is
// the caller's incoming traceparent, so agent+gateway render as one trace in SigNoz.
func (g *Gateway) ChatCompletions(w http.ResponseWriter, r *http.Request) {
	ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))
	body, _ := io.ReadAll(r.Body)

	var req map[string]any
	_ = json.Unmarshal(body, &req)
	model, _ := req["model"].(string)
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

	upReq, err := http.NewRequestWithContext(ctx, http.MethodPost, g.upstream+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		g.fail(w, span, err)
		return
	}
	upReq.Header.Set("Content-Type", "application/json")
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(upReq.Header))

	resp, err := g.client.Do(upReq)
	if err != nil {
		g.fail(w, span, err)
		return
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)

	var respJSON map[string]any
	if json.Unmarshal(respBody, &respJSON) == nil {
		if usage, ok := respJSON["usage"].(map[string]any); ok {
			if v, ok := usage["prompt_tokens"].(float64); ok {
				span.SetAttributes(attribute.Int("gen_ai.usage.input_tokens", int(v)))
			}
			if v, ok := usage["completion_tokens"].(float64); ok {
				span.SetAttributes(attribute.Int("gen_ai.usage.output_tokens", int(v)))
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	_, _ = w.Write(respBody)
}

func (g *Gateway) fail(w http.ResponseWriter, span trace.Span, err error) {
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
	http.Error(w, err.Error(), http.StatusBadGateway)
}
