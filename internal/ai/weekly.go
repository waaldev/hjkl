package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// Quest is a measurable goal for the coming week, checked against coach
// events: "use" counts real uses of Keys, "avoid" keeps an anti-pattern
// under Target.
type Quest struct {
	Kind    string `json:"kind"`              // use | avoid
	Keys    string `json:"keys,omitempty"`    // for use: a command, e.g. ciw
	Pattern string `json:"pattern,omitempty"` // for avoid: e.g. repeat-x
	Target  int    `json:"target"`
	Text    string `json:"text"`
}

// WeeklyReview is the AI's short review plus next week's quests.
type WeeklyReview struct {
	Review string  `json:"review"`
	Quests []Quest `json:"quests"`
}

// WeeklyStats is what the review sees: aggregated counts only.
type WeeklyStats struct {
	AntiPatterns map[string]int // pattern → count
	Used         map[string]int // command → count
	Patterns     []string       // anti-patterns the coach can count
	Unlocked     []string
}

// Weekly asks for a review and quests from a week of coach counts. The
// quests are not checked here; the caller keeps only ones the coach can
// measure.
func Weekly(ctx context.Context, p Provider, st WeeklyStats) (WeeklyReview, error) {
	raw, err := CompleteJSON(ctx, p, Request{System: teachSystem, Prompt: weeklyPrompt(st)})
	if err != nil {
		return WeeklyReview{}, err
	}
	var w WeeklyReview
	if err := json.Unmarshal(raw, &w); err != nil {
		return WeeklyReview{}, fmt.Errorf("weekly review: %w", err)
	}
	w.Review = strings.TrimSpace(w.Review)
	return w, nil
}

func weeklyPrompt(st WeeklyStats) string {
	return fmt.Sprintf(`Write a short weekly Vim coaching review from these aggregated counts
(no file contents exist here), then set up to three quests for next week.

Slow habits this week (pattern: count):
%s
Commands used well in real work (command: count):
%s
Unlocked skills: %s

Quests must be measurable by the coach:
- {"kind":"use","keys":"<a command from the used list's style, e.g. ciw, dap, ., cgn, ;>","target":N,"text":"..."}
- {"kind":"avoid","pattern":"<one of: %s>","target":N,"text":"..."} (stay at or under N)
Pick quests that replace this week's worst habits with unlocked skills.
Use realistic targets. Reply with one JSON object:
{"review": "3-5 sentences, plain text", "quests": [...]}`,
		counts(st.AntiPatterns), counts(st.Used), strings.Join(st.Unlocked, ", "), strings.Join(st.Patterns, ", "))
}

func counts(m map[string]int) string {
	if len(m) == 0 {
		return "(none)"
	}
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	sort.Slice(ks, func(i, j int) bool { return m[ks[i]] > m[ks[j]] })
	var b strings.Builder
	for _, k := range ks {
		fmt.Fprintf(&b, "- %s: %d\n", k, m[k])
	}
	return b.String()
}
