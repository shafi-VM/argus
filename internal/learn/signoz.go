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

// LatestHealth returns the windowed grounding rate. Traces are fresh by construction
// (the query window is the last `lookback`), so age is ~0 when data exists; with too
// few samples it returns an error so the poller holds (warm-up).
func (s *SigNoz) LatestHealth(ctx context.Context) (float64, time.Duration, error) {
	now := s.now()
	q := fmt.Sprintf(`{"schemaVersion":"v1","start":%d,"end":%d,"requestType":"scalar",`+
		`"compositeQuery":{"queries":[{"type":"builder_query","spec":{"name":"A","signal":"traces",`+
		`"aggregations":[{"expression":"count()"}],`+
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
	return groundingRate(raw, s.minSamples)
}

// groundingRate parses the grouped counts and returns pass/(pass+recovered+refused).
// upstream_error / transport_error are EXCLUDED — infra failures are not behavioral
// drift (red-team R2), so they must not move the health signal.
func groundingRate(raw []byte, minSamples int) (float64, time.Duration, error) {
	// v5 scalar rows are ARRAYS aligned to `columns` ([group, aggregation]), not
	// objects. Read the column layout, then index each row positionally.
	var out struct {
		Data struct {
			Data struct {
				Results []struct {
					Columns []struct {
						Name       string `json:"name"`
						ColumnType string `json:"columnType"`
					} `json:"columns"`
					Data [][]json.RawMessage `json:"data"`
				} `json:"results"`
			} `json:"data"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return 0, 0, err
	}

	counts := map[string]float64{}
	for _, r := range out.Data.Data.Results {
		gi, ai := -1, -1
		for i, c := range r.Columns {
			switch c.ColumnType {
			case "group":
				gi = i
			case "aggregation":
				ai = i
			}
		}
		if gi < 0 || ai < 0 {
			continue
		}
		for _, row := range r.Data {
			if gi >= len(row) || ai >= len(row) {
				continue
			}
			var decision string
			_ = json.Unmarshal(row[gi], &decision)
			var n float64
			_ = json.Unmarshal(row[ai], &n)
			counts[decision] += n
		}
	}

	behavioral := counts["pass"] + counts["recovered"] + counts["refused"]
	if behavioral < float64(minSamples) {
		return 0, 0, fmt.Errorf("signoz: only %.0f behavioral spans in window (< %d)", behavioral, minSamples)
	}
	return counts["pass"] / behavioral, 0, nil
}
