package coach

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/waaldev/hjkl/internal/ai"
	"github.com/waaldev/hjkl/internal/store"
)

func TestValidQuests(t *testing.T) {
	unlocked := []string{"operators", "text-objects", "dot", "words"}
	in := []ai.Quest{
		{Kind: "use", Keys: "ciw", Target: 5, Text: "Use ciw 5 times"},            // ok
		{Kind: "use", Keys: "gUiw", Target: 5, Text: "uppercase"},                 // the coach cannot count it
		{Kind: "use", Keys: "cgn", Target: 3, Text: "cgn, but gn is locked"},      // skill locked
		{Kind: "use", Keys: ";", Target: 5, Text: "find-char not unlocked"},       // skill locked
		{Kind: "avoid", Pattern: "repeat-x", Target: 10, Text: "Fewer xxxx"},      // ok
		{Kind: "avoid", Pattern: "made-up", Target: 1, Text: "unknown pattern"},   // unknown
		{Kind: "use", Keys: "dw", Target: 0, Text: "zero target"},                 // nothing to do
		{Kind: "use", Keys: ".", Target: 20, Text: "Use . 20 times"},              // ok
		{Kind: "use", Keys: "d2w", Target: 3, Text: "fourth valid, over the cap"}, // capped at 3
	}
	got := ValidQuests(in, append(unlocked, "counts"))
	if len(got) != 3 || got[0].Keys != "ciw" || got[1].Pattern != "repeat-x" || got[2].Keys != "." {
		t.Fatalf("got %+v", got)
	}
}

func TestQuestProgress(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	st, err := store.Open(filepath.Join(dir, "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	start := time.Now().Add(-time.Hour)
	qs := QuestSet{Start: start, Quests: []ai.Quest{
		{Kind: "use", Keys: "ciw", Target: 3, Text: "ciw x3"},
		{Kind: "avoid", Pattern: "repeat-x", Target: 2, Text: "xxxx at most twice"},
	}}
	if err := SaveQuests(ctx, st, qs); err != nil {
		t.Fatal(err)
	}
	// Events from before the quests started do not count.
	old := start.Add(-time.Hour).UTC().Format(time.RFC3339)
	now := time.Now().UTC().Format(time.RFC3339)
	ev := `{"pattern":"used","keys":"ciw","count":9,"ts":"` + old + `"}
{"pattern":"used","keys":"ciw","count":3,"ts":"` + now + `"}
{"pattern":"repeat-x","keys":"xxxx","count":5,"ts":"` + now + `"}
`
	path := filepath.Join(dir, "ev.jsonl")
	if err := os.WriteFile(path, []byte(ev), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := IngestJSONL(ctx, st, path); err != nil {
		t.Fatal(err)
	}

	active, ok := ActiveQuests(ctx, st, time.Now())
	if !ok {
		t.Fatal("quests should be active")
	}
	p := Progress(ctx, st, active)
	if p[0].Count != 3 || !p[0].Done {
		t.Fatalf("use quest: %+v", p[0])
	}
	if p[1].Count != 5 || p[1].Done {
		t.Fatalf("avoid quest over its limit: %+v", p[1])
	}
	if _, ok := ActiveQuests(ctx, st, time.Now().Add(QuestWeek+time.Hour)); ok {
		t.Fatal("quests expire after a week")
	}
}
