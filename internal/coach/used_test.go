package coach

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/waaldev/hjkl/internal/srs"
	"github.com/waaldev/hjkl/internal/store"
)

func TestUsedSkills(t *testing.T) {
	cases := map[string][]string{
		"ciw": {"operators", "text-objects"},
		"d2w": {"operators", "words", "counts"},
		".":   {"dot"},
		"dd":  {"dd-yy-p"},
		"d<":  {},
	}
	for keys, want := range cases {
		got := UsedSkills(keys)
		if len(got) != len(want) {
			t.Fatalf("%s: %v want %v", keys, got, want)
		}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("%s: %v want %v", keys, got, want)
			}
		}
	}
}

func TestIngestUsedLightsGridAndReviewsKnownSkills(t *testing.T) {
	dir := t.TempDir()
	st, err := store.Open(filepath.Join(dir, "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	ctx := context.Background()

	// text-objects was taught (has a card, due now); operators never was.
	past := time.Now().Add(-48 * time.Hour)
	c := srs.NewCard("text-objects", past)
	if err := st.UpsertSRS(ctx, store.SRSCard{Skill: c.Skill, Easiness: c.Easiness, DueAt: c.DueAt}); err != nil {
		t.Fatal(err)
	}

	events := filepath.Join(dir, "ev.jsonl")
	line := `{"pattern":"used","keys":"ciw","count":3,"ts":"` + time.Now().UTC().Format(time.RFC3339) + `"}` + "\n" +
		`{"pattern":"repeat-j","keys":"jjjjj","count":5,"ts":"` + time.Now().UTC().Format(time.RFC3339) + `"}` + "\n"
	if err := os.WriteFile(events, []byte(line), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := IngestJSONL(ctx, st, events); err != nil {
		t.Fatal(err)
	}

	grid, _ := st.GrammarGrid(ctx)
	if grid["c\tiw"] == 0 {
		t.Fatalf("ciw in real work should light the grid: %v", grid)
	}
	card, ok, _ := st.SRS(ctx, "text-objects")
	if !ok || card.Repetitions != 1 || card.LastReviewAt == nil {
		t.Fatalf("known skill should get a review: %+v", card)
	}
	if _, ok, _ := st.SRS(ctx, "operators"); ok {
		t.Fatal("real-world use must not schedule a skill the dojo hasn't taught")
	}
	sum, _ := st.CoachSummary(ctx, time.Now().Add(-time.Hour))
	if _, ok := sum["used"]; ok || sum["repeat-j"] != 5 {
		t.Fatalf("anti-pattern summary should exclude used events: %v", sum)
	}
	used, _ := st.CoachUsage(ctx, time.Now().Add(-time.Hour))
	if used["ciw"] != 3 {
		t.Fatalf("usage=%v", used)
	}
}
