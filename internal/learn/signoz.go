package learn

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// SigNoz implements SigNozClient by reading the windowed grounding rate back through
// SigNoz's v5 query API — the ADR-0003-correct source for LEARN's windowed decision.
//
// It queries TRACES, not the intelligence_health metric gauge: measured on 2026-07-22,
// SigNoz metric query-lag is ~60s while trace ingestion is ~5s (P1). A control loop
// that must fire within a demo beat has to read the fast signal. The health value is
// the windowed grounding rate = pass / (pass+recovered+refused), which is exactly the
// primary driver of the Intelligence Health score, computed from argus.decision spans.
type SigNoz struct {
	url        string
	apiKey     string
	lookback   time.Duration
	minSamples int
	client     *http.Client
	now        func() time.Time
}

func NewSigNoz(url, apiKey string) *SigNoz {
	return &SigNoz{
		url:        strings.TrimRight(url, "/"),
		apiKey:     apiKey,
		lookback:   30 * time.Second, // > query_range freshness lag (~13s, measured) + samples
		minSamples: 3,
		client:     &http.Client{Timeout: 5 * time.Second},
		now:        time.Now,
	}
}

// LatestHealth returns the windowed grounding rate and the REAL age of the freshest
// span backing it (now - max(span timestamp)). Age is not ~0: trace ingestion lags
// ~13s, so a truthful age lets the poller refuse to act when SigNoz falls further
// behind (R5), and makes argus.learn.age_s an honest observed-lag signal rather than
// a hardcoded zero. Too few samples returns an error so the poller holds (warm-up).
func (s *SigNoz) LatestHealth(ctx context.Context) (float64, time.Duration, error) {
	now := s.now()
	// count() drives the grounding rate; max(timestamp) drives the freshness age.
	q := fmt.Sprintf(`{"schemaVersion":"v1","start":%d,"end":%d,"requestType":"scalar",`+
		`"compositeQuery":{"queries":[{"type":"builder_query","spec":{"name":"A","signal":"traces",`+
		`"aggregations":[{"expression":"count()"},{"expression":"max(timestamp)"}],`+
		`"groupBy":[{"name":"argus.decision","fieldContext":"span"}]}}]}}`,
		now.Add(-s.lookback).UnixMilli(), now.UnixMilli())

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.url+"/api/v5/query_range",
		bytes.NewReader([]byte(q)))
	if err != nil {
		return 0, 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	if s.apiKey != "" {
		req.Header.Set("SIGNOZ-API-KEY", s.apiKey)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return 0, 0, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return 0, 0, fmt.Errorf("signoz query_range: http %d", resp.StatusCode)
	}
	rate, newest, err := groundingRate(raw, s.minSamples)
	if err != nil {
		return 0, 0, err
	}
	age := now.Sub(newest)
	if age < 0 { // clock skew / span stamped in the future — treat as fresh, never negative
		age = 0
	}
	return rate, age, nil
}

// groundingRate parses the grouped counts and returns pass/(pass+recovered+refused)
// plus the newest span timestamp in the window (for the freshness age).
//
// NOTE (review #2): this is NOT the same computation as the hero dashboard's
// Intelligence Health gauge. That gauge is health.Window.Score =
// grounding_rate − loop_penalty − cost_penalty, from argusd's LOCAL in-process
// buckets. This is the raw grounding rate from TRACE spans. They agree only while
// the loop/cost penalties are zero; the moment either fires, the number the dashboard
// shows and the number LEARN acts on diverge. Kept intentionally simple here because
// grounding is the dominant driver and it's the signal SigNoz can serve back.
//
// upstream_error / transport_error are EXCLUDED — infra failures are not behavioral
// drift (red-team R2), so they must not move the health signal.
func groundingRate(raw []byte, minSamples int) (float64, time.Time, error) {
	// v5 scalar rows are ARRAYS aligned to `columns` ([group, agg0, agg1]), not
	// objects. Columns carry aggregationIndex: 0 = count(), 1 = max(timestamp).
	var out struct {
		Data struct {
			Data struct {
				Results []struct {
					Columns []struct {
						Name       string `json:"name"`
						ColumnType string `json:"columnType"`
						AggIndex   int    `json:"aggregationIndex"`
					} `json:"columns"`
					Data [][]json.RawMessage `json:"data"`
				} `json:"results"`
			} `json:"data"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return 0, time.Time{}, err
	}

	counts := map[string]float64{}
	var newest time.Time
	for _, r := range out.Data.Data.Results {
		gi, ci, ti := -1, -1, -1
		for i, c := range r.Columns {
			switch {
			case c.ColumnType == "group":
				gi = i
			case c.ColumnType == "aggregation" && c.AggIndex == 0:
				ci = i // count()
			case c.ColumnType == "aggregation" && c.AggIndex == 1:
				ti = i // max(timestamp)
			}
		}
		if gi < 0 || ci < 0 {
			continue
		}
		for _, row := range r.Data {
			if gi >= len(row) || ci >= len(row) {
				continue
			}
			var decision string
			_ = json.Unmarshal(row[gi], &decision)
			var n float64
			_ = json.Unmarshal(row[ci], &n)
			counts[decision] += n
			// Freshness = age of the newest BEHAVIORAL span (pass/recovered/refused).
			// Not the empty-decision group (agent/LEARN spans keep flowing every ~2s
			// even if chat decisions stopped — that would defeat the staleness guard),
			// and not infra errors. The guard must track the signal it acts on.
			if isBehavioral(decision) && ti >= 0 && ti < len(row) {
				if t, ok := parseSpanTime(row[ti]); ok && t.After(newest) {
					newest = t
				}
			}
		}
	}

	behavioral := counts["pass"] + counts["recovered"] + counts["refused"]
	if behavioral < float64(minSamples) {
		return 0, time.Time{}, fmt.Errorf("signoz: only %.0f behavioral spans in window (< %d)", behavioral, minSamples)
	}
	if newest.IsZero() {
		// We have counts but no parseable timestamp — we cannot vouch for freshness,
		// so fail toward holding rather than acting on data of unknown age.
		return 0, time.Time{}, fmt.Errorf("signoz: could not determine data freshness")
	}
	return counts["pass"] / behavioral, newest, nil
}

// parseSpanTime reads SigNoz's max(timestamp) aggregation value. The v5 API returns
// it as a NUMERIC epoch (seconds, with a fractional part) — e.g. 1784888514.913 —
// NOT an RFC3339 string. (The earlier RFC3339-only parse silently failed on every
// real response, so `newest` stayed zero and the freshness guard held forever — the
// LEARN loop never fired. This is the live-verified shape, 2026-07-24.) We parse the
// number first and keep an RFC3339 fallback for robustness across versions.
func parseSpanTime(raw json.RawMessage) (time.Time, bool) {
	var sec float64
	if json.Unmarshal(raw, &sec) == nil && sec > 0 {
		// Tolerate seconds / millis / nanos by magnitude (epoch-2001..2286 in secs).
		switch {
		case sec > 1e17: // nanoseconds
			return time.Unix(0, int64(sec)), true
		case sec > 1e14: // microseconds
			return time.Unix(0, int64(sec)*1e3), true
		case sec > 1e11: // milliseconds
			return time.Unix(0, int64(sec)*1e6), true
		default: // seconds (with fractional part)
			whole := int64(sec)
			return time.Unix(whole, int64((sec-float64(whole))*1e9)), true
		}
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

// isBehavioral reports whether a decision contributes to the grounding rate (and thus
// to the freshness age). Infra errors (upstream_error/transport_error) and non-chat
// spans (empty group) are excluded — R2.
func isBehavioral(decision string) bool {
	return decision == "pass" || decision == "recovered" || decision == "refused"
}
