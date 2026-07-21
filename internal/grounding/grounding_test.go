package grounding

import "testing"

const ctx = `{"tool":"flight_search","results":[{"flight":"AA42","depart":"SFO 09:15"}]}`

func TestCheck(t *testing.T) {
	cases := []struct {
		name        string
		answer      string
		ctx         string
		grounded    bool
		skipped     bool
		unsupported int
	}{
		{"grounded cites AA42", "Flight AA42 departs SFO 09:15, nonstop.", ctx, true, false, 0},
		{"hallucinated cites UA99", "Flight UA99 departs SFO 06:00.", ctx, false, false, 1},
		{"no entity claimed", "I couldn't find a matching flight.", ctx, true, false, 0},
		{"no context -> fail OPEN", "Flight UA99 departs SFO 06:00.", "", true, true, 0},
		{"one supported one not", "Try AA42, or UA99 as backup.", ctx, false, false, 1},
		{"repeat claim counted once", "UA99 then UA99 again.", ctx, false, false, 1},
		// regression #25: substring matching wrongly accepted a prefix.
		{"prefix must NOT satisfy a claim (#25)", "Flight AA42 departs.",
			`{"results":[{"flight":"AA420"}]}`, false, false, 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := Check(c.answer, c.ctx)
			if got.Grounded != c.grounded {
				t.Errorf("Grounded = %v, want %v", got.Grounded, c.grounded)
			}
			if got.Skipped != c.skipped {
				t.Errorf("Skipped = %v, want %v", got.Skipped, c.skipped)
			}
			if len(got.Unsupported) != c.unsupported {
				t.Errorf("Unsupported = %v, want %d entries", got.Unsupported, c.unsupported)
			}
		})
	}
}

func TestExtractContext(t *testing.T) {
	msgs := []Message{
		{Role: "system", Content: "Answer only using this.\nRETRIEVED_CONTEXT: {\"a\":1}"},
		{Role: "user", Content: "book a flight"},
	}
	if got := ExtractContext(msgs); got != `{"a":1}` {
		t.Errorf("ExtractContext = %q, want %q", got, `{"a":1}`)
	}
	if got := ExtractContext([]Message{{Role: "user", Content: "hi"}}); got != "" {
		t.Errorf("expected empty context, got %q", got)
	}
}
