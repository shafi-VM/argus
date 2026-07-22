package health

import (
	"math"
	"testing"
	"time"
)

type clock struct{ t time.Time }

func (c *clock) now() time.Time       { return c.t }
func (c *clock) tick(d time.Duration) { c.t = c.t.Add(d) }

func newWin(c *clock) *Window {
	return NewWindow(Config{BucketSecs: 3, NBuckets: 5, MinSamples: 3, BudgetUSD: 0.05, Now: c.now})
}

func near(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

func TestWarmUpHoldsColdValue(t *testing.T) {
	w := newWin(&clock{t: time.Unix(1000, 0)})
	w.Record(true, 0, 0)
	w.Record(true, 0, 0) // only 2 samples, < MinSamples=3
	if got, n := w.Score(1.0); !near(got, 1.0) || n != 2 {
		t.Fatalf("warm-up: got %v (n=%d), want cold 1.0", got, n)
	}
}

func TestGroundingRateDrivesScore(t *testing.T) {
	w := newWin(&clock{t: time.Unix(1000, 0)})
	for i := 0; i < 6; i++ {
		w.Record(true, 0, 0)
	}
	for i := 0; i < 4; i++ {
		w.Record(false, 0, 0) // 6/10 grounded
	}
	if got, n := w.Score(1.0); !near(got, 0.6) || n != 10 {
		t.Fatalf("score = %v (n=%d), want 0.60", got, n)
	}
}

func TestPenaltiesAreBounded(t *testing.T) {
	w := newWin(&clock{t: time.Unix(1000, 0)})
	// all grounded but heavy loops + cost -> penalties cap, score stays >= 0.5
	for i := 0; i < 5; i++ {
		w.Record(true, 100, 100.0)
	}
	got, _ := w.Score(1.0)
	// grounding 1.0 - loop cap 0.30 - cost cap 0.20 = 0.50 exactly
	if !near(got, 0.5) {
		t.Fatalf("bounded penalties: score = %v, want 0.50", got)
	}
}

func TestOldSamplesEvictedByTime(t *testing.T) {
	c := &clock{t: time.Unix(1000, 0)}
	w := newWin(c)
	for i := 0; i < 5; i++ {
		w.Record(false, 0, 0) // 5 ungrounded now
	}
	c.tick(30 * time.Second) // window is 15s -> all evicted
	for i := 0; i < 5; i++ {
		w.Record(true, 0, 0) // 5 grounded later
	}
	if got, n := w.Score(1.0); !near(got, 1.0) || n != 5 {
		t.Fatalf("after eviction score = %v (n=%d), want 1.0 over 5 fresh", got, n)
	}
}

// The governor must not double-fire and must not oscillate.
func TestGovernorNoDoubleFire(t *testing.T) {
	g := NewGovernor(0.5, 0.7, 2)
	seq := []float64{0.9, 0.4, 0.4, 0.3, 0.3, 0.2} // dips and stays low
	var acts []Action
	for _, h := range seq {
		acts = append(acts, g.Observe(h))
	}
	// exactly one Quarantine (on the 3rd reading = 2nd consecutive below), then None
	quarantines := 0
	for _, a := range acts {
		if a == Quarantine {
			quarantines++
		}
	}
	if quarantines != 1 {
		t.Fatalf("quarantines = %d, want exactly 1 (idempotent)", quarantines)
	}
	if g.Phase() != Quarantined {
		t.Fatalf("phase = %v, want quarantined", g.Phase())
	}
}

func TestGovernorHysteresisNoFlap(t *testing.T) {
	g := NewGovernor(0.5, 0.7, 2)
	g.Observe(0.4)
	g.Observe(0.4) // -> Quarantine
	// a single bounce to 0.6 (between down and up) must NOT recover
	if a := g.Observe(0.6); a != None {
		t.Fatalf("recovered at 0.6 (below up=0.7); hysteresis broken: %v", a)
	}
	if a := g.Observe(0.6); a != None {
		t.Fatalf("still recovered below up threshold: %v", a)
	}
	if g.Phase() != Quarantined {
		t.Fatalf("phase flapped out of quarantine, got %v", g.Phase())
	}
	// sustained above up recovers once
	g.Observe(0.8)
	if a := g.Observe(0.8); a != Recover {
		t.Fatalf("did not recover after 2 evals >= up: %v", a)
	}
	if a := g.Observe(0.9); a != None {
		t.Fatalf("re-fired Recover; not idempotent: %v", a)
	}
}
