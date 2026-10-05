package coach

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"time"

	"github.com/waaldev/hjkl/internal/game"
	"github.com/waaldev/hjkl/internal/progress"
	"github.com/waaldev/hjkl/internal/store"
)

type fileEvent struct {
	TS       string `json:"ts"`
	Pattern  string `json:"pattern"`
	Keys     string `json:"keys"`
	Skill    string `json:"skill"`
	Filetype string `json:"filetype"`
	Count    int    `json:"count"`
}

// IngestJSONL reads coach events from the plugin's JSONL file into SQLite
// and truncates the file so events are not double-counted.
func IngestJSONL(ctx context.Context, st *store.Store, path string) (int, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	sc := bufio.NewScanner(f)
	n := 0
	for sc.Scan() {
		line := sc.Bytes()
		if len(line) == 0 {
			continue
		}
		var ev fileEvent
		if err := json.Unmarshal(line, &ev); err != nil {
			continue
		}
		ts := time.Now()
		if ev.TS != "" {
			if parsed, err := time.Parse(time.RFC3339Nano, ev.TS); err == nil {
				ts = parsed
			} else if parsed, err := time.Parse(time.RFC3339, ev.TS); err == nil {
				ts = parsed
			}
		}
		if ev.Count == 0 {
			ev.Count = 1
		}
		if ev.Pattern == "used" {
			if err := applyUsed(ctx, st, ev, ts); err != nil {
				_ = f.Close()
				return n, err
			}
		}
		if err := st.AddCoachEvent(ctx, store.CoachEvent{
			TS: ts, Pattern: ev.Pattern, Keys: ev.Keys, Skill: ev.Skill, Filetype: ev.Filetype, Count: ev.Count,
		}); err != nil {
			_ = f.Close()
			return n, err
		}
		n++
	}
	cerr := f.Close()
	if err := sc.Err(); err != nil {
		return n, err
	}
	if cerr != nil {
		return n, cerr
	}
	if n > 0 {
		if err := os.WriteFile(path, nil, 0o600); err != nil {
			return n, err
		}
	}
	return n, nil
}

// usedQuality is the SRS quality for using a skill unprompted in real work:
// a solid recall, short of a perfect drill.
const usedQuality = 4

// applyUsed turns real-world use into progress: it lights the Grammar Grid
// and counts as an SRS review for skills the dojo has already taught.
func applyUsed(ctx context.Context, st *store.Store, ev fileEvent, ts time.Time) error {
	for _, pair := range game.LightFromKeys(ev.Keys) {
		if err := st.LightGrammar(ctx, pair[0], pair[1]); err != nil {
			return err
		}
	}
	for _, skill := range UsedSkills(ev.Keys) {
		if err := progress.ReviewSkill(ctx, st, skill, usedQuality, ts, false); err != nil {
			return err
		}
	}
	return nil
}
