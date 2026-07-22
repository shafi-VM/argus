// Package health computes the windowed Intelligence Health score and runs the LEARN
// governor that decides quarantine/recover. Pure logic, no I/O — this is where
// LEARN's correctness lives (window math, no double-fire, no oscillation), so it is
// unit-tested in isolation before it is wired to argusd or SigNoz.
package health

import (
	"sync"
	"time"
)

// Window is a rolling, bucketed view of request outcomes. Only 2xx responses are
// recorded: an upstream error is an INFRASTRUCTURE failure, not behavioral drift
// (issue #25 / red-team R2), so it must never pull the behavioral score down.
type Window struct {
	mu         sync.Mutex
	buckets    []bucket
	nBuckets   int64
	bucketSecs int64
	minSamples int
	budgetUSD  float64
	now        func() time.Time
}

type bucket struct {
	epoch    int64 // unixSec/bucketSecs at last write; detects slot reuse across cycles
	total    int
	grounded int
	loops    int
	cost     float64
}

// Config for a Window; zero values fall back to demo-tuned defaults.
type Config struct {
	BucketSecs int64   // width of each bucket in seconds
	NBuckets   int64   // buckets in the window (window length = BucketSecs*NBuckets)
	MinSamples int     // below this, Score returns the cold value (warm-up guard)
	BudgetUSD  float64 // per-request cost budget — a CONFIGURED CONSTANT, not a metric
	Now        func() time.Time
}

// NewWindow defaults to a 15s window (5×3s) — deliberately short so the demo score
// visibly moves on demo cadence. Tune after measuring (red-team R1).
func NewWindow(c Config) *Window {
	if c.BucketSecs == 0 {
		c.BucketSecs = 3
	}
	if c.NBuckets == 0 {
		c.NBuckets = 5
	}
	if c.MinSamples == 0 {
		c.MinSamples = 3
	}
	if c.BudgetUSD == 0 {
		c.BudgetUSD = 0.05
	}
	if c.Now == nil {
		c.Now = time.Now
	}
	return &Window{
		buckets:    make([]bucket, c.NBuckets),
		nBuckets:   c.NBuckets,
		bucketSecs: c.BucketSecs,
		minSamples: c.MinSamples,
		budgetUSD:  c.BudgetUSD,
		now:        c.Now,
	}
}

func (w *Window) epoch() int64 { return w.now().Unix() / w.bucketSecs }

// Record adds one 2xx response outcome to the current bucket.
func (w *Window) Record(grounded bool, loops int, costUSD float64) {
	w.mu.Lock()
	defer w.mu.Unlock()
	e := w.epoch()
	b := &w.buckets[e%w.nBuckets]
	if b.epoch != e { // slot belongs to an older cycle -> reset it for this epoch
		*b = bucket{epoch: e}
	}
	b.total++
	if grounded {
		b.grounded++
	}
	b.loops += loops
	b.cost += costUSD
}

// Score returns Intelligence Health in [0,1] and the sample count in the window.
// Below MinSamples it returns coldValue (warm-up guard: one request can't slam the
// gauge to 0 or 1).
//
//	health = clamp(grounding_rate - loop_penalty - cost_penalty, 0, 1)
func (w *Window) Score(coldValue float64) (float64, int) {
	w.mu.Lock()
	defer w.mu.Unlock()
	e := w.epoch()
	var total, grounded, loops int
	var cost float64
	for _, b := range w.buckets {
		if d := e - b.epoch; d >= 0 && d < w.nBuckets { // bucket falls inside the window
			total += b.total
			grounded += b.grounded
			loops += b.loops
			cost += b.cost
		}
	}
	if total < w.minSamples {
		return coldValue, total
	}
	n := float64(total)
	groundingRate := float64(grounded) / n
	loopPenalty := min(0.30, 0.10*float64(loops)/n)
	costPenalty := min(0.20, 0.20*max(0, (cost/n)/w.budgetUSD-1))
	return clamp(groundingRate-loopPenalty-costPenalty, 0, 1), total
}

func clamp(v, lo, hi float64) float64 { return min(hi, max(lo, v)) }
