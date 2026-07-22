// Package mission is Argus Mission Control — the demo god-mode surface.
//
// It is deliberately minimal (KILL_LIST: "one chaos button only"): it shows System /
// PREVENT / LEARN state + the last Argus action, and offers ONE control — inject or
// stop behavioral drift. All state is read from the gateway IN-PROCESS (0ms); nothing
// here reads SigNoz (ADR-0003). The chaos button is a demo control that toggles the
// replay engine's persona; it targets the configured upstream's /admin/chaos.
package mission

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/shafi-VM/argus/internal/gateway"
)

// Provider is the read side Mission Control needs — satisfied by *gateway.Gateway.
type Provider interface {
	MissionState() gateway.Snapshot
}

// Handler serves the control surface. chaosURL is the demo fault target (the replay
// engine's /admin/chaos); empty disables the chaos control.
type Handler struct {
	gw       Provider
	chaosURL string
	client   *http.Client
}

func New(gw Provider, chaosURL string) *Handler {
	return &Handler{
		gw:       gw,
		chaosURL: strings.TrimRight(chaosURL, "/"),
		client:   &http.Client{Timeout: 2 * time.Second},
	}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /mission", h.page)
	mux.HandleFunc("GET /mission/state", h.state)
	mux.HandleFunc("POST /mission/chaos", h.chaos)
}

func (h *Handler) page(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = io.WriteString(w, pageHTML)
}

// state is the poll target: live System / PREVENT / LEARN / last-action + chaos mode.
func (h *Handler) state(w http.ResponseWriter, r *http.Request) {
	s := h.gw.MissionState()
	now := time.Now()
	ago := func(t time.Time) any {
		if t.IsZero() {
			return nil
		}
		return now.Sub(t).Milliseconds()
	}
	mode, chaosOK := h.currentChaos(r.Context())

	writeJSON(w, http.StatusOK, map[string]any{
		"system": map[string]any{"ok": true, "detail": "argusd operational"},
		"prevent": map[string]any{
			"active":       true,
			"lastDecision": s.LastDecision,
			"lastModel":    s.LastDecisionModel,
			"agoMs":        ago(s.LastDecisionAt),
		},
		"learn": map[string]any{
			"quarantined": s.Quarantined,
			"healthy":     len(s.Quarantined) == 0,
		},
		"lastAction": map[string]any{"text": s.LastAction, "agoMs": ago(s.LastActionAt)},
		"chaos":      map[string]any{"drift": mode == "hallucinated", "mode": mode, "available": chaosOK},
	})
}

// chaos is the ONE control. Body {"drift": bool} maps to the replay engine's persona
// (hallucinated = drift on, grounded = off) and forwards to <upstream>/admin/chaos.
func (h *Handler) chaos(w http.ResponseWriter, r *http.Request) {
	if h.chaosURL == "" {
		writeJSON(w, http.StatusNotImplemented, map[string]any{"error": "chaos target not configured"})
		return
	}
	var in struct {
		Drift bool `json:"drift"`
	}
	body, _ := io.ReadAll(r.Body)
	_ = json.Unmarshal(body, &in)

	mode := "grounded"
	if in.Drift {
		mode = "hallucinated"
	}
	payload, _ := json.Marshal(map[string]string{"mode": mode})
	req, err := http.NewRequestWithContext(r.Context(), http.MethodPost,
		h.chaosURL+"/admin/chaos", bytes.NewReader(payload))
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": err.Error()})
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := h.client.Do(req)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": err.Error()})
		return
	}
	defer resp.Body.Close()
	out, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": "chaos target rejected", "status": resp.StatusCode})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"drift": in.Drift, "mode": mode, "upstream": json.RawMessage(out)})
}

// currentChaos best-effort reads the replay engine's persona so the button shows the
// right label. A failure just means "chaos unavailable" — it never blocks the page.
func (h *Handler) currentChaos(ctx context.Context) (string, bool) {
	if h.chaosURL == "" {
		return "", false
	}
	cctx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
	defer cancel()
	req, err := http.NewRequestWithContext(cctx, http.MethodGet, h.chaosURL+"/admin/chaos", nil)
	if err != nil {
		return "", false
	}
	resp, err := h.client.Do(req)
	if err != nil {
		return "", false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", false
	}
	var m struct {
		Mode string `json:"mode"`
	}
	if json.NewDecoder(resp.Body).Decode(&m) != nil {
		return "", false
	}
	return m.Mode, true
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
