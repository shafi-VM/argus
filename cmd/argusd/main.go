// Command argusd is the Argus gateway — a drop-in OpenAI-compatible proxy that
// PREVENTs behavioral failure inline and drives the LEARN loop through SigNoz.
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/shafi-VM/argus/internal/gateway"
	"github.com/shafi-VM/argus/internal/health"
	"github.com/shafi-VM/argus/internal/learn"
	"github.com/shafi-VM/argus/internal/metrics"
	"github.com/shafi-VM/argus/internal/mission"
	"github.com/shafi-VM/argus/internal/telemetry"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	shutTraces, err := telemetry.Init(ctx, "argusd")
	if err != nil {
		log.Fatalf("telemetry (traces): %v", err)
	}
	defer func() { _ = shutTraces(ctx) }()

	shutMetrics, err := telemetry.InitMetrics(ctx, "argusd")
	if err != nil {
		log.Fatalf("telemetry (metrics): %v", err)
	}
	defer func() { _ = shutMetrics(ctx) }()

	// Intelligence Health window (owned by the metrics emitter, exported as a gauge).
	win := health.NewWindow(health.Config{})
	m, err := metrics.New(win)
	if err != nil {
		log.Fatalf("metrics: %v", err)
	}

	upstream := getenv("ARGUS_UPSTREAM", "http://127.0.0.1:9099")
	gw := gateway.New(upstream)
	gw.SetMetrics(m)

	// LEARN loop: only if SigNoz creds are present. It reads the windowed health
	// BACK from SigNoz and quarantines/reroutes via the gateway (the Actuator).
	learnOn := false
	if url, key := os.Getenv("SIGNOZ_URL"), os.Getenv("SIGNOZ_API_KEY"); url != "" && key != "" {
		poller := learn.New(learn.Config{
			Client:   learn.NewSigNoz(url, key),
			Actuator: gw,
			Model:    getenv("ARGUS_PRIMARY_MODEL", "gpt-4o"),
			Fallback: getenv("ARGUS_FALLBACK_MODEL", "gpt-4o-mini"),
			Interval: 2 * time.Second,
		})
		go poller.Run(ctx)
		learnOn = true
		log.Printf("LEARN poller on: reading windowed decision from SigNoz query_range at %s", url)
	} else {
		log.Printf("LEARN poller off (set SIGNOZ_URL + SIGNOZ_API_KEY to enable)")
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/chat/completions", gw.ChatCompletions)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("ok")) })

	// Mission Control: the demo god-mode surface. Reads live state from the gateway
	// (in-process, ADR-0003) and drives ONE control — inject/stop drift. The chaos
	// target defaults to the upstream, which IS the replay engine in the demo.
	mission.New(gw, getenv("ARGUS_CHAOS_URL", upstream)).SetLearnActive(learnOn).Register(mux)

	addr := getenv("ARGUS_ADDR", ":8088")
	log.Printf("Mission Control at http://localhost%s/mission", addr)
	log.Printf("argusd listening on %s -> upstream %s", addr, upstream)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}

func getenv(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
