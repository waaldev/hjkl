package curriculum

import "strings"

// Variant overrides a challenge's buffers and solution. Empty fields keep
// the base challenge's value.
type Variant struct {
	Start        string `yaml:"start"`
	Target       string `yaml:"target"`
	StartCursor  []int  `yaml:"start_cursor"`
	TargetCursor []int  `yaml:"target_cursor,omitempty"`
	Par          int    `yaml:"par"`
	Solution     string `yaml:"solution"`
}

// HintLadder returns the hints shown one at a time on F1, gentlest first:
// the idea (skill family), then the author's hint, then the answer. Each
// step costs a star, so the player decides how much help to take.
func (ch Challenge) HintLadder() []string {
	if len(ch.Hints) > 0 {
		return ch.Hints
	}
	var out []string
	add := func(h string) {
		h = strings.TrimSpace(h)
		if h == "" {
			return
		}
		for _, prev := range out {
			if prev == h {
				return
			}
		}
		out = append(out, h)
	}
	add(skillNudge(ch.Skills))
	if ch.Hint != ch.Solution {
		add(ch.Hint)
	}
	add("answer: " + ch.Solution)
	return out
}

// skillNudge names the tools without saying which one: "think: text
// objects (iw aw i\" a( ip it)".
func skillNudge(skills []string) string {
	var parts []string
	for _, id := range skills {
		for _, d := range SkillDocs {
			if d.ID == id {
				parts = append(parts, strings.ReplaceAll(id, "-", " ")+" ("+d.Keys+")")
			}
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return "think: " + strings.Join(parts, ", ")
}

// ForReview returns the challenge as a retrieval test: no brief or lesson,
// and a single hint that only names the skill family.
func (ch Challenge) ForReview() Challenge {
	ch.Review = true
	ch.Brief = ""
	ch.Teach = ""
	ch.Hints = nil
	if n := skillNudge(ch.Skills); n != "" {
		ch.Hints = []string{n}
	} else {
		ch.Hints = []string{"no hints in review - you know this one"}
	}
	return ch
}

// WithVariant returns the challenge with variant i applied (i < 0 or out of
// range returns the base challenge).
func (ch Challenge) WithVariant(i int) Challenge {
	if i < 0 || i >= len(ch.Variants) {
		return ch
	}
	v := ch.Variants[i]
	if v.Start != "" {
		ch.Start = normalizeText(v.Start)
		ch.Target = ch.Start
	}
	if v.Target != "" {
		ch.Target = normalizeText(v.Target)
	}
	if len(v.StartCursor) == 2 {
		ch.StartCursor = v.StartCursor
	}
	if len(v.TargetCursor) == 2 {
		ch.TargetCursor = v.TargetCursor
	}
	if v.Solution != "" {
		ch.Solution = v.Solution
		ch.Par = v.Par
		if ch.Par <= 0 {
			ch.Par = keysCount(v.Solution)
		}
	}
	return ch
}
