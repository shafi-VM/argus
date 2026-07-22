package health

// Governor is the LEARN state machine. It turns a stream of health scores into
// discrete, idempotent actions with hysteresis, so LEARN can never double-fire
// (R3) or oscillate (R4).
//
// Transitions (confirm=2, down=0.5, up=0.7):
//
//	Healthy --health<down for 2 evals--> Quarantined   (emit Quarantine)
//	Quarantined --health>=up for 2 evals--> Healthy     (emit Recover)
//
// While Quarantined, further low scores return None — the action already happened.
type Governor struct {
	phase       Phase
	down, up    float64
	confirm     int
	belowStreak int
	aboveStreak int
}

type Phase int

const (
	Healthy Phase = iota
	Quarantined
)

func (p Phase) String() string {
	if p == Quarantined {
		return "quarantined"
	}
	return "healthy"
}

// Action a single Observe may emit.
type Action int

const (
	None Action = iota
	Quarantine
	Recover
)

func (a Action) String() string {
	switch a {
	case Quarantine:
		return "quarantine"
	case Recover:
		return "recover"
	default:
		return "none"
	}
}

// NewGovernor uses demo-tuned hysteresis. down<up is required for stability.
func NewGovernor(down, up float64, confirm int) *Governor {
	if down == 0 && up == 0 {
		down, up = 0.5, 0.7
	}
	if confirm <= 0 {
		confirm = 2
	}
	return &Governor{phase: Healthy, down: down, up: up, confirm: confirm}
}

// Observe feeds one health score and returns the action to take (usually None).
func (g *Governor) Observe(health float64) Action {
	switch g.phase {
	case Healthy:
		if health < g.down {
			g.belowStreak++
			if g.belowStreak >= g.confirm {
				g.phase = Quarantined
				g.belowStreak, g.aboveStreak = 0, 0
				return Quarantine
			}
		} else {
			g.belowStreak = 0
		}
	case Quarantined:
		if health >= g.up {
			g.aboveStreak++
			if g.aboveStreak >= g.confirm {
				g.phase = Healthy
				g.belowStreak, g.aboveStreak = 0, 0
				return Recover
			}
		} else {
			g.aboveStreak = 0
		}
	}
	return None
}

// Phase reports the current governor phase (for span attributes / Mission Control).
func (g *Governor) Phase() Phase { return g.phase }
