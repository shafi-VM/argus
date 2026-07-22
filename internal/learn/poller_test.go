package learn

import (
	"context"
	"testing"
	"time"

	"github.com/shafi-VM/argus/internal/health"
)

// fakeSignoz returns scripted (value, age) pairs, one per Tick.
type fakeSignoz struct {
	values []float64
	age    time.Duration
	i      int
}

func (f *fakeSignoz) LatestHealth(context.Context) (float64, time.Duration, error) {
	v := f.values[min(f.i, len(f.values)-1)]
	f.i++
	return v, f.age, nil
}

type fakeActuator struct{ quarantines, recovers int }

func (a *fakeActuator) Quarantine(_, _ string) { a.quarantines++ }
func (a *fakeActuator) Recover(_ string)       { a.recovers++ }

func drive(t *testing.T, sn *fakeSignoz) (*fakeActuator, []health.Action) {
	t.Helper()
	act := &fakeActuator{}
	p := New(Config{
		Client: sn, Actuator: act,
		Governor: health.NewGovernor(0.5, 0.7, 2),
		Model:    "gpt-4o", Fallback: "gpt-4o-mini",
		MaxAge: 15 * time.Second,
	})
	var actions []health.Action
	for range sn.values {
		actions = append(actions, p.Tick(context.Background()))
	}
	return act, actions
}

// The full LEARN arc: healthy -> drift -> quarantine -> recover, exactly once each.
func TestLearnArcQuarantineThenRecover(t *testing.T) {
	sn := &fakeSignoz{
		values: []float64{0.95, 0.40, 0.30, 0.30, 0.80, 0.85, 0.85},
		age:    2 * time.Second, // fresh
	}
	act, _ := drive(t, sn)
	if act.quarantines != 1 {
		t.Errorf("quarantines = %d, want exactly 1 (idempotent)", act.quarantines)
	}
	if act.recovers != 1 {
		t.Errorf("recovers = %d, want exactly 1", act.recovers)
	}
}

// A 5xx storm shows up as stale-or-absent behavioral data, not as drift. Here we
// prove the staleness guard: old data must NEVER trigger an action (R5).
func TestStaleDataNeverActs(t *testing.T) {
	sn := &fakeSignoz{
		values: []float64{0.10, 0.10, 0.10, 0.10}, // very unhealthy...
		age:    60 * time.Second,                  // ...but stale
	}
	act, actions := drive(t, sn)
	if act.quarantines != 0 {
		t.Errorf("acted on stale data: quarantines = %d, want 0", act.quarantines)
	}
	for _, a := range actions {
		if a != health.None {
			t.Errorf("stale tick produced action %v, want None", a)
		}
	}
}

// A brief dip between thresholds must not quarantine, and must not flap.
func TestBriefDipDoesNotQuarantine(t *testing.T) {
	sn := &fakeSignoz{
		values: []float64{0.9, 0.4, 0.9, 0.4, 0.9}, // single dips, never 2 in a row
		age:    1 * time.Second,
	}
	act, _ := drive(t, sn)
	if act.quarantines != 0 {
		t.Errorf("quarantined on transient dips: quarantines = %d, want 0", act.quarantines)
	}
}
