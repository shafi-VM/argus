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

// REAL v5 scalar shape (live-verified 2026-07-24): rows are ARRAYS aligned to
// `columns`, two aggregation columns — aggregationIndex 0 = count(), 1 = max(timestamp).
// CRITICAL: max(timestamp) is a NUMERIC epoch (seconds, fractional), e.g. 1784888012.913
// — NOT an RFC3339 string. An earlier version of this fixture used strings, which hid a
// real bug: the parser RFC3339-only, so `newest` stayed zero on every live response and
// the freshness guard held forever → LEARN never fired. This fixture is now the real shape.
// pass=2, recovered=6, refused=1, upstream_error=5, plus an empty-decision group (ignored).
// grounding rate = 2/(2+6+1)=0.222 (upstream_error excluded, R2). Newest BEHAVIORAL span is
// recovered @ 1784888012; the empty group's newer 1784888020 must NOT be treated as fresh.
const traceGroups = `{"data":{"data":{"results":[{` +
	`"columns":[` +
	`{"name":"argus.decision","columnType":"group","aggregationIndex":0},` +
	`{"name":"__result_0","columnType":"aggregation","aggregationIndex":0},` +
	`{"name":"__result_1","columnType":"aggregation","aggregationIndex":1}],` +
	`"data":[` +
	`[null,99,1784888020.4],` +
	`["pass",2,1784888009.1],` +
	`["recovered",6,1784888012.9],` +
	`["refused",1,1784888005.0],` +
	`["upstream_error",5,1784888018.7]]}]}}}`

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
	// Freeze the clock ~13s after the newest behavioral span (recovered @ 1784888012.9)
	// so the computed age is deterministic and real (not the old hardcoded 0, and proving
	// the NUMERIC epoch is parsed — a string parse would leave age at the IsZero error).
	s.now = func() time.Time { return time.Unix(1784888026, 0).UTC() } // 1784888026 − 1784888012.9 ≈ 13.1s

	val, age, err := s.LatestHealth(context.Background())
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if val < 0.221 || val > 0.223 {
		t.Errorf("grounding rate = %v, want 0.222 (2/9, upstream_error excluded)", val)
	}
	// now(1784888026) − recovered(1784888012.9) ≈ 13.1s, computed from the NUMERIC epoch.
	// Crucially NOT ~5.6s (the empty group's 1784888020.4, which R2 must exclude), and NOT
	// the old IsZero-error path a string parse would take. Allow a small float tolerance.
	if age < 13*time.Second || age > 132*time.Second/10 {
		t.Errorf("age = %v, want ~13.1s (now − newest behavioral span, numeric epoch)", age)
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
		`"data":[["pass",1,1784888009.1]]}]}}}` // numeric epoch, real shape
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()
	s := NewSigNoz(srv.URL, "")
	if _, _, err := s.LatestHealth(context.Background()); err == nil {
		t.Error("too few samples should error (warm-up hold), got nil")
	}
}
