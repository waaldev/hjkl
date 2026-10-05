package coach

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"time"

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
