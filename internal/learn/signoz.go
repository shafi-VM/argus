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

// SigNoz implements SigNozClient by reading the windowed Intelligence Health gauge
// back through SigNoz's v5 query API — the ADR-0003-correct source for LEARN's
// windowed decision (not ClickHouse-direct, not local state).
type SigNoz struct {
	url    string
	apiKey string
	metric string
	lookba time.Duration
	client *http.Client
	now    func() time.Time
}

// NewSigNoz targets metric (e.g. "argus_intelligence_health_ratio") at a SigNoz base
// URL (e.g. http://localhost:8081) with a service-account API key.
func NewSigNoz(url, apiKey, metric string) *SigNoz {
	return &SigNoz{
		url:    strings.TrimRight(url, "/"),
		apiKey: apiKey,
		metric: metric,
		lookba: 60 * time.Second,
		client: &http.Client{Timeout: 5 * time.Second},
		now:    time.Now,
	}
}

// LatestHealth returns the newest data point for the metric and how old it is.
func (s *SigNoz) LatestHealth(ctx context.Context) (float64, time.Duration, error) {
	now := s.now()
	q := fmt.Sprintf(`{"schemaVersion":"v1","start":%d,"end":%d,"requestType":"time_series",`+
		`"compositeQuery":{"queries":[{"type":"builder_query","spec":{"name":"A","signal":"metrics",`+
		`"aggregations":[{"metricName":%q,"temporality":"unspecified","timeAggregation":"avg","spaceAggregation":"avg"}],`+
		`"stepInterval":5}}]}}`,
		now.Add(-s.lookba).UnixMilli(), now.UnixMilli(), s.metric)

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
	return parseLatest(raw, now)
}

func parseLatest(raw []byte, now time.Time) (float64, time.Duration, error) {
	var out struct {
		Data struct {
			Data struct {
				Results []struct {
					Aggregations []struct {
						Series []struct {
							Values []struct {
								Timestamp int64   `json:"timestamp"`
								Value     float64 `json:"value"`
							} `json:"values"`
						} `json:"series"`
					} `json:"aggregations"`
				} `json:"results"`
			} `json:"data"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return 0, 0, err
	}

	var newestTs int64
	var newestVal float64
	found := false
	for _, r := range out.Data.Data.Results {
		for _, a := range r.Aggregations {
			for _, ser := range a.Series {
				for _, v := range ser.Values {
					if !found || v.Timestamp > newestTs {
						newestTs, newestVal, found = v.Timestamp, v.Value, true
					}
				}
			}
		}
	}
	if !found {
		return 0, 0, fmt.Errorf("signoz query_range: no data points for metric")
	}
	age := now.Sub(time.UnixMilli(newestTs))
	if age < 0 {
		age = 0
	}
	return newestVal, age, nil
}
