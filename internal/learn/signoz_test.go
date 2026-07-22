package learn

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// REAL v5 scalar shape: rows are ARRAYS aligned to `columns`, and there are TWO
// aggregation columns — aggregationIndex 0 = count(), 1 = max(timestamp). pass=2,
// recovered=6, refused=1, upstream_error=5, plus an empty-decision group (non-chat
// spans) that must be ignored. grounding rate = 2/(2+6+1)=0.222, upstream_error does
// not dilute it (R2). The newest BEHAVIORAL span is recovered @ 10:12:12Z; the empty
// group's newer timestamp (10:12:20Z) must NOT be treated as fresh (guard integrity).
const traceGroups = `{"data":{"data":{"results":[{` +
	`"columns":[` +
	`{"name":"argus.decision","columnType":"group","aggregationIndex":0},` +
	`{"name":"__result_0","columnType":"aggregation","aggregationIndex":0},` +
	`{"name":"__result_1","columnType":"aggregation","aggregationIndex":1}],` +
	`"data":[` +
	`[null,99,"2026-07-22T10:12:20Z"],` +
	`["pass",2,"2026-07-22T10:12:09Z"],` +
	`["recovered",6,"2026-07-22T10:12:12Z"],` +
	`["refused",1,"2026-07-22T10:12:05Z"],` +
	`["upstream_error",5,"2026-07-22T10:12:18Z"]]}]}}}`

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
	// Freeze the clock 13s after the newest behavioral span (recovered @ 10:12:12Z)
	// so the computed age is deterministic and real (not the old hardcoded 0).
	s.now = func() time.Time { return time.Date(2026, 7, 22, 10, 12, 25, 0, time.UTC) }

	val, age, err := s.LatestHealth(context.Background())
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if val < 0.221 || val > 0.223 {
		t.Errorf("grounding rate = %v, want 0.222 (2/9, upstream_error excluded)", val)
	}
	// 10:12:25 − 10:12:12 (recovered, the newest BEHAVIORAL span) = 13s. Crucially NOT
	// 5s (which the empty group's 10:12:20 timestamp would give if freshness ignored R2).
	if age != 13*time.Second {
		t.Errorf("age = %v, want 13s (now − newest behavioral span)", age)
	}
	if gotKey != "secret-key" {
		t.Errorf("SIGNOZ-API-KEY = %q", gotKey)
	}
	if !strings.Contains(gotBody, `"signal":"traces"`) ||
		!strings.Contains(gotBody, "argus.decision") ||
		!strings.Contains(gotBody, "max(timestamp)") {
		t.Errorf("query not the expected trace group-by with freshness agg: %s", gotBody)
	}
}

func TestSigNozTooFewSamplesHolds(t *testing.T) {
	// real array shape, 1 behavioral span < minSamples(3) -> warm-up hold (error).
	body := `{"data":{"data":{"results":[{` +
		`"columns":[` +
		`{"name":"argus.decision","columnType":"group","aggregationIndex":0},` +
		`{"name":"__result_0","columnType":"aggregation","aggregationIndex":0},` +
		`{"name":"__result_1","columnType":"aggregation","aggregationIndex":1}],` +
		`"data":[["pass",1,"2026-07-22T10:12:09Z"]]}]}}}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()
	s := NewSigNoz(srv.URL, "")
	if _, _, err := s.LatestHealth(context.Background()); err == nil {
		t.Error("too few samples should error (warm-up hold), got nil")
	}
}
