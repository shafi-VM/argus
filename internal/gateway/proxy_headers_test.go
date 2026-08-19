package gateway

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// argusd is a drop-in proxy in front of a REAL LLM, so the caller's Authorization
// (API key) and provider-specific headers MUST reach the upstream — otherwise a real
// provider (OpenAI/Groq/…) returns 401 and the whole thing only ever works with the
// demo mock. This guards that passthrough.
func TestForwardsAuthAndProviderHeaders(t *testing.T) {
	var got http.Header
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Clone()
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"Flight AA42 departs."}}]}`))
	}))
	defer up.Close()

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(reqBody))
	req.Header.Set("Authorization", "Bearer sk-test-123")
	req.Header.Set("OpenAI-Organization", "org-abc")
	New(up.URL).ChatCompletions(httptest.NewRecorder(), req)

	if got.Get("Authorization") != "Bearer sk-test-123" {
		t.Errorf("Authorization not forwarded upstream: %q — a real provider would 401", got.Get("Authorization"))
	}
	if got.Get("OpenAI-Organization") != "org-abc" {
		t.Errorf("OpenAI-Organization not forwarded: %q", got.Get("OpenAI-Organization"))
	}
	// argusd must NOT leak headers it never received.
	if got.Get("Anthropic-Version") != "" {
		t.Errorf("forwarded a header the caller never sent: %q", got.Get("Anthropic-Version"))
	}
}
