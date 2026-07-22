package gateway

import (
	"testing"
	"time"
)

// Mission Control reads live state from the gateway in-process. These lock down that
// read-model: quarantines and the last transition/decision are reflected accurately,
// and a repeated (idempotent) quarantine does NOT reset the "last action" timestamp.

func TestMissionStateReflectsQuarantineAndRecover(t *testing.T) {
	g := New("http://upstream")

	if s := g.MissionState(); len(s.Quarantined) != 0 || s.LastAction != "" {
		t.Fatalf("fresh gateway should be clean, got %+v", s)
	}

	g.Quarantine("gpt-4o", "gpt-4o-mini")
	s := g.MissionState()
	if s.Quarantined["gpt-4o"] != "gpt-4o-mini" {
		t.Errorf("quarantine not reflected: %+v", s.Quarantined)
	}
	if s.LastAction != "Quarantined gpt-4o → gpt-4o-mini" {
		t.Errorf("last action = %q", s.LastAction)
	}
	if s.LastActionAt.IsZero() {
		t.Error("last action timestamp should be set")
	}

	g.Recover("gpt-4o")
	s = g.MissionState()
	if len(s.Quarantined) != 0 {
		t.Errorf("recover should clear reroute, got %+v", s.Quarantined)
	}
	if s.LastAction != "Recovered gpt-4o" {
		t.Errorf("last action = %q", s.LastAction)
	}
}

func TestQuarantineIdempotentDoesNotResetTimestamp(t *testing.T) {
	g := New("http://upstream")
	g.Quarantine("gpt-4o", "gpt-4o-mini")
	first := g.MissionState().LastActionAt

	time.Sleep(3 * time.Millisecond)
	g.Quarantine("gpt-4o", "gpt-4o-mini") // repeat: no real transition

	if got := g.MissionState().LastActionAt; !got.Equal(first) {
		t.Errorf("idempotent re-quarantine reset the action timestamp: %v -> %v", first, got)
	}
}

func TestNoteDecisionSnapshot(t *testing.T) {
	g := New("http://upstream")
	g.noteDecision("gpt-4o", "recovered")
	s := g.MissionState()
	if s.LastDecision != "recovered" || s.LastDecisionModel != "gpt-4o" {
		t.Errorf("last decision = %q/%q", s.LastDecision, s.LastDecisionModel)
	}
	g.noteDecision("gpt-4o", "") // empty decision must not overwrite
	if g.MissionState().LastDecision != "recovered" {
		t.Error("empty decision overwrote the last real one")
	}
}
