package coach

import (
	"context"
	"encoding/json"
	"time"

	"github.com/waaldev/hjkl/internal/ai"
	"github.com/waaldev/hjkl/internal/store"
)

// Patterns are the anti-patterns the coach plugin records.
var Patterns = []string{
	"repeat-j", "repeat-k", "repeat-h", "repeat-l", "repeat-x", "repeat-w",
	"arrow-keys", "insert-arrows", "visual-iw-delete", "visual-iw-yank",
	"visual-line-walk", "costly-edit",
}

const maxQuests = 3

// ValidQuests keeps quests the coach can actually measure, within the
// player's unlocked skills, with sane targets.
func ValidQuests(qs []ai.Quest, unlocked []string) []ai.Quest {
	known := map[string]bool{}
	for _, p := range Patterns {
		known[p] = true
	}
	have := map[string]bool{}
	for _, s := range unlocked {
		have[s] = true
	}
	var out []ai.Quest
	for _, q := range qs {
		if q.Target < 0 || q.Target > 100 || q.Text == "" {
			continue
		}
		switch q.Kind {
		case "use":
			skills := UsedSkills(q.Keys)
			if q.Target < 1 || len(skills) == 0 {
				continue // the coach would never count it
			}
			ok := true
			for _, s := range skills {
				ok = ok && have[s]
			}
			if !ok {
				continue
			}
		case "avoid":
			if !known[q.Pattern] {
				continue
			}
		default:
			continue
		}
		out = append(out, q)
		if len(out) == maxQuests {
			break
		}
	}
	return out
}

// QuestSet is the stored quests and when they started counting.
type QuestSet struct {
	Start  time.Time  `json:"start"`
	Quests []ai.Quest `json:"quests"`
}

const metaQuests = "quests"

// QuestWeek is how long a quest set runs.
const QuestWeek = 7 * 24 * time.Hour

func SaveQuests(ctx context.Context, st *store.Store, qs QuestSet) error {
	raw, err := json.Marshal(qs)
	if err != nil {
		return err
	}
	return st.SetMeta(ctx, metaQuests, string(raw))
}

// ActiveQuests returns this week's quests, if any are set and still running.
func ActiveQuests(ctx context.Context, st *store.Store, now time.Time) (QuestSet, bool) {
	raw, err := st.Meta(ctx, metaQuests)
	if err != nil || raw == "" {
		return QuestSet{}, false
	}
	var qs QuestSet
	if json.Unmarshal([]byte(raw), &qs) != nil || len(qs.Quests) == 0 || now.Sub(qs.Start) > QuestWeek {
		return QuestSet{}, false
	}
	return qs, true
}

// QuestProgress is how far a quest has come since the set started.
type QuestProgress struct {
	Quest ai.Quest
	Count int
	Done  bool // use: reached the target; avoid: still at or under it
}

func Progress(ctx context.Context, st *store.Store, qs QuestSet) []QuestProgress {
	used, _ := st.CoachUsage(ctx, qs.Start)
	bad, _ := st.CoachSummary(ctx, qs.Start)
	out := make([]QuestProgress, 0, len(qs.Quests))
	for _, q := range qs.Quests {
		p := QuestProgress{Quest: q}
		switch q.Kind {
		case "use":
			p.Count = used[q.Keys]
			p.Done = p.Count >= q.Target
		case "avoid":
			p.Count = bad[q.Pattern]
			p.Done = p.Count <= q.Target
		}
		out = append(out, p)
	}
	return out
}
