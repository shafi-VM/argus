package learn

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// REAL v5 scalar shape: rows are ARRAYS aligned to `columns`, not objects.
// pass=2, recovered=6, refused=1, upstream_error=5. There is also an empty-decision
// group (non-chat spans) that must be ignored. grounding rate = 2/(2+6+1)=0.222, and
// upstream_error must NOT dilute it (R2).
const traceGroups = `{"data":{"data":{"results":[{` +
	`"columns":[{"name":"argus.decision","columnType":"group"},{"name":"__result_0","columnType":"aggregation"}],` +
	`"data":[[null,99],["pass",2],["recovered",6],["refused",1],["upstream_error",5]]}]}}}`

func TestSigNozGroundingRateFromTraces(t *testing.T) {
	var gotKey, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotKey = r.Header.Get("SIGNOZ-API-KEY")
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		_, _ = w.Write([]byte(traceGroups))
	}))
	defer srv.Close()

	s := NewSigNoz(srv.URL, "secret-key")
	val, age, err := s.LatestHealth(context.Background())
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if val < 0.221 || val > 0.223 {
		t.Errorf("grounding rate = %v, want 0.222 (2/9, upstream_error excluded)", val)
	}
	if age != 0 {
		t.Errorf("age = %v, want 0 (traces fresh by window)", age)
	}
	if gotKey != "secret-key" {
		t.Errorf("SIGNOZ-API-KEY = %q", gotKey)
	}
	if !strings.Contains(gotBody, `"signal":"traces"`) || !strings.Contains(gotBody, "argus.decision") {
		t.Errorf("query not a trace group-by: %s", gotBody)
	}
}

func TestSigNozTooFewSamplesHolds(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"data":{"data":{"results":[{"data":[` +
			`{"argus.decision":"pass","__result_0":1}]}]}}}`)) // 1 < minSamples
	}))
	defer srv.Close()
	s := NewSigNoz(srv.URL, "")
	if _, _, err := s.LatestHealth(context.Background()); err == nil {
		t.Error("too few samples should error (warm-up hold), got nil")
	}
}
