// Command argusd is the Argus gateway — a drop-in OpenAI-compatible proxy that
// observes (and, later, PREVENTs on) AI agent behavior. Day 1: the spine.
package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/shafi-VM/argus/internal/gateway"
	"github.com/shafi-VM/argus/internal/telemetry"
)

func main() {
	ctx := context.Background()

	shutdown, err := telemetry.Init(ctx, "argusd")
	if err != nil {
		log.Fatalf("telemetry init: %v", err)
	}
	defer func() { _ = shutdown(ctx) }()

	upstream := getenv("ARGUS_UPSTREAM", "http://127.0.0.1:9099")
	gw := gateway.New(upstream)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/chat/completions", gw.ChatCompletions)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("ok")) })

	addr := getenv("ARGUS_ADDR", ":8088")
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
