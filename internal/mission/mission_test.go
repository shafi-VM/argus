package mission

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/shafi-VM/argus/internal/gateway"
)

type fakeProvider struct{ s gateway.Snapshot }

func (f fakeProvider) MissionState() gateway.Snapshot { return f.s }

// a stand-in replay engine: GET returns current mode, POST records what was set.
func chaosServer(mode *string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/admin/chaos" {
			w.WriteHeader(404)
			return
		}
		if r.Method == http.MethodPost {
			var b struct {
				Mode string `json:"mode"`
			}
			body, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(body, &b)
			*mode = b.Mode
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"mode": *mode})
	}))
}

func TestStateReflectsQuarantineAndChaosMode(t *testing.T) {
	mode := "hallucinated"
	chaos := chaosServer(&mode)
	defer chaos.Close()

	h := New(fakeProvider{gateway.Snapshot{
		Quarantined:  map[string]string{"gpt-4o": "gpt-4o-mini"},
		LastAction:   "Quarantined gpt-4o → gpt-4o-mini",
		LastDecision: "recovered", LastDecisionModel: "gpt-4o",
	}}, chaos.URL)

	rr := httptest.NewRecorder()
	h.state(rr, httptest.NewRequest("GET", "/mission/state", nil))
	if rr.Code != 200 {
		t.Fatalf("state code = %d", rr.Code)
	}
	var st struct {
		Learn struct {
			Quarantined map[string]string `json:"quarantined"`
			Healthy     bool              `json:"healthy"`
		} `json:"learn"`
		Chaos struct {
			Drift     bool `json:"drift"`
			Available bool `json:"available"`
		} `json:"chaos"`
		Prevent struct {
			LastDecision string `json:"lastDecision"`
		} `json:"prevent"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &st); err != nil {
		t.Fatalf("bad json: %v", err)
	}
	if st.Learn.Healthy {
		t.Error("learn should be unhealthy while a model is quarantined")
	}
	if st.Learn.Quarantined["gpt-4o"] != "gpt-4o-mini" {
		t.Errorf("quarantine not surfaced: %+v", st.Learn.Quarantined)
	}
	if !st.Chaos.Drift || !st.Chaos.Available {
		t.Errorf("chaos should read drift=true available=true, got %+v", st.Chaos)
	}
	if st.Prevent.LastDecision != "recovered" {
		t.Errorf("prevent.lastDecision = %q", st.Prevent.LastDecision)
	}
}

func TestChaosButtonForwardsMode(t *testing.T) {
	mode := "grounded"
	chaos := chaosServer(&mode)
	defer chaos.Close()
	h := New(fakeProvider{}, chaos.URL)

	// drift:true must set the replay engine to "hallucinated"
	rr := httptest.NewRecorder()
	h.chaos(rr, httptest.NewRequest("POST", "/mission/chaos", strings.NewReader(`{"drift":true}`)))
	if rr.Code != 200 {
		t.Fatalf("chaos code = %d body=%s", rr.Code, rr.Body.String())
	}
	if mode != "hallucinated" {
		t.Errorf("drift:true should set hallucinated, replay engine got %q", mode)
	}

	// drift:false must set it back to "grounded"
	rr = httptest.NewRecorder()
	h.chaos(rr, httptest.NewRequest("POST", "/mission/chaos", strings.NewReader(`{"drift":false}`)))
	if mode != "grounded" {
		t.Errorf("drift:false should set grounded, replay engine got %q", mode)
	}
}

func TestChaosDisabledWhenUnconfigured(t *testing.T) {
	h := New(fakeProvider{}, "") // no chaos target
	rr := httptest.NewRecorder()
	h.chaos(rr, httptest.NewRequest("POST", "/mission/chaos", strings.NewReader(`{"drift":true}`)))
	if rr.Code != http.StatusNotImplemented {
		t.Errorf("unconfigured chaos should 501, got %d", rr.Code)
	}
	// and state must report chaos unavailable, not crash
	rr = httptest.NewRecorder()
	h.state(rr, httptest.NewRequest("GET", "/mission/state", nil))
	if !strings.Contains(rr.Body.String(), `"available":false`) {
		t.Errorf("state should report chaos unavailable: %s", rr.Body.String())
	}
}

func TestPageServesHTML(t *testing.T) {
	h := New(fakeProvider{}, "")
	rr := httptest.NewRecorder()
	h.page(rr, httptest.NewRequest("GET", "/mission", nil))
	if rr.Code != 200 || !strings.Contains(rr.Body.String(), "Mission Control") {
		t.Errorf("page not served: code=%d", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Errorf("content-type = %q", ct)
	}
}
