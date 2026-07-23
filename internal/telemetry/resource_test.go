package telemetry

import (
	"context"
	"testing"
)

// Guards the P8 "complete resource" claim: a SigNoz engineer looks for a per-instance
// id and a low-cardinality environment, not just name/version. Drop either and this
// goes red. Also pins that traces and metrics share ONE instance id (correlation).
func TestResourceIsComplete(t *testing.T) {
	r, err := newResource(context.Background(), "argusd")
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, kv := range r.Attributes() {
		got[string(kv.Key)] = kv.Value.Emit()
	}
	for _, k := range []string{
		"service.name", "service.version",
		"service.instance.id", "deployment.environment.name",
	} {
		if got[k] == "" {
			t.Errorf("resource missing %q", k)
		}
	}
	if got["service.name"] != "argusd" {
		t.Errorf("service.name = %q, want argusd (never unknown_service)", got["service.name"])
	}

	// trace + metric providers must resolve the SAME instance id.
	r2, _ := newResource(context.Background(), "argusd")
	var id2 string
	for _, kv := range r2.Attributes() {
		if string(kv.Key) == "service.instance.id" {
			id2 = kv.Value.Emit()
		}
	}
	if got["service.instance.id"] != id2 {
		t.Errorf("instance id differs across resources (%q vs %q) — breaks signal correlation",
			got["service.instance.id"], id2)
	}
}
