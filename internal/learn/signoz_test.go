package learn

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestSigNozLatestHealthPicksNewestPoint(t *testing.T) {
	now := time.Unix(2_000_000, 0)
	tsNew := now.Add(-4 * time.Second).UnixMilli()
	tsOld := now.Add(-40 * time.Second).UnixMilli()

	var gotKey, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotKey = r.Header.Get("SIGNOZ-API-KEY")
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		// two points; newest (tsNew) has value 0.12, older has 0.90
		resp := fmt.Sprintf(`{"data":{"data":{"results":[{"aggregations":[{"series":[{"values":[`+
			`{"timestamp":%d,"value":0.90},{"timestamp":%d,"value":0.12}]}]}]}]}}}`, tsOld, tsNew)
		_, _ = w.Write([]byte(resp))
	}))
	defer srv.Close()

	s := NewSigNoz(srv.URL, "secret-key", "argus_intelligence_health_ratio")
	s.now = func() time.Time { return now }

	val, age, err := s.LatestHealth(context.Background())
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if val != 0.12 {
		t.Errorf("value = %v, want newest 0.12", val)
	}
	if age < 3*time.Second || age > 5*time.Second {
		t.Errorf("age = %v, want ~4s", age)
	}
	if gotKey != "secret-key" {
		t.Errorf("SIGNOZ-API-KEY = %q, want secret-key", gotKey)
	}
	if !strings.Contains(gotBody, "argus_intelligence_health_ratio") ||
		!strings.Contains(gotBody, `"signal":"metrics"`) {
		t.Errorf("query body missing metric/signal: %s", gotBody)
	}
}

func TestSigNozNoDataIsAnError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"data":{"data":{"results":[]}}}`))
	}))
	defer srv.Close()
	s := NewSigNoz(srv.URL, "", "argus_intelligence_health_ratio")
	if _, _, err := s.LatestHealth(context.Background()); err == nil {
		t.Error("empty results should error (no data to act on), got nil")
	}
}
