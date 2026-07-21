// Package grounding implements the deterministic PREVENT detector (ADR-0002).
//
// It compares entity claims in a model's answer against the context the model was
// handed IN the request. No LLM, no model, no network — pure string work, so it
// runs inline in the request path in well under the 50ms budget.
//
// Scope (stated honestly, per ADR-0002's non-goal): this is ENTITY-PRESENCE
// grounding. It catches "the answer cites something that is not in the provided
// context." It is deliberately NOT general hallucination detection — it will not
// catch a correct entity paired with wrong details.
package grounding

import (
	"regexp"
	"strings"
)

// contextMarker is the in-band convention a caller uses to pass retrieved context.
// argusd reads this from the REQUEST — never from a fixture file on disk — so the
// check stays valid for any caller, not just our demo agent.
const contextMarker = "RETRIEVED_CONTEXT:"

// entityRe matches booking-style entity ids (AA42, UA99, DL1234).
var entityRe = regexp.MustCompile(`\b[A-Z]{2}\d{2,4}\b`)

// Message is the minimal shape we need from an OpenAI-style chat request.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Result of a Grounding Check.
type Result struct {
	Grounded    bool
	Skipped     bool // no context supplied -> we cannot judge, so we fail OPEN
	Claims      []string
	Unsupported []string
}

// ExtractContext pulls the in-band retrieved context out of the request messages.
// Returns "" when the caller supplied none.
func ExtractContext(messages []Message) string {
	for _, m := range messages {
		if i := strings.Index(m.Content, contextMarker); i >= 0 {
			return strings.TrimSpace(m.Content[i+len(contextMarker):])
		}
	}
	return ""
}

// Check reports whether every entity claimed in answer appears in ctx.
//
// Fail-open is deliberate: with no context we cannot verify anything, and blocking
// what we cannot verify would produce false positives — the one failure mode that
// would make PREVENT worse than the disease.
func Check(answer, ctx string) Result {
	if strings.TrimSpace(ctx) == "" {
		return Result{Grounded: true, Skipped: true}
	}

	res := Result{Grounded: true}
	seen := make(map[string]bool)
	for _, claim := range entityRe.FindAllString(answer, -1) {
		if seen[claim] {
			continue
		}
		seen[claim] = true
		res.Claims = append(res.Claims, claim)
		if !strings.Contains(ctx, claim) {
			res.Unsupported = append(res.Unsupported, claim)
			res.Grounded = false
		}
	}
	return res
}
