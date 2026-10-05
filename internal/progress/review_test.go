package progress

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/waaldev/hjkl/internal/curriculum"
	"github.com/waaldev/hjkl/internal/runner"
	"github.com/waaldev/hjkl/internal/store"
)

func TestPickReviewRotatesClearedDrills(t *testing.T) {
	now := time.Now()
	chs := []curriculum.Challenge{
		{ID: "a", Start: "a"},
		{ID: "b", Start: "b", Variants: []curriculum.Variant{{Start: "b2"}}},
		{ID: "new", Start: "n"},
	}
	stars := map[string]int{"a": 3, "b": 2}
	last := map[string]time.Time{"a": now, "b": now.Add(-48 * time.Hour)}
	base := func(int) int { return 0 }    // -1 → base drill
	variant := func(int) int { return 1 } // 0 → first variant

	if got := PickReview(chs, stars, last, base); got.ID != "b" || got.Start != "b" {
		t.Fatalf("want least recently played cleared drill b, got %s %q", got.ID, got.Start)
	}
	if got := PickReview(chs, stars, last, variant); got.Start != "b2" {
		t.Fatalf("want variant b2, got %q", got.Start)
	}
	// Never-cleared drills are new material, not review.
	if got := PickReview(chs, map[string]int{"a": 1}, last, base); got.ID != "a" {
		t.Fatalf("want a, got %s", got.ID)
	}
	if got := PickReview(chs[2:], nil, nil, base); got.ID != "new" {
		t.Fatalf("fallback to uncleared, got %s", got.ID)
	}
}

func TestNextInterleavesReviews(t *testing.T) {
	ctx := context.Background()
	st, err := store.Open(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	cat, err := curriculum.Load()
	if err != nil {
		t.Fatal(err)
	}
	base := func(int) int { return 0 }
	now := time.Now()

	// Clear three new drills; their skills become cards due tomorrow.
	for i := 0; i < InterleaveEvery; i++ {
		ch, review, ok := Next(ctx, st, cat, now, base)
		if !ok || review {
			t.Fatalf("drill %d: want a new drill, got review=%v ok=%v", i, review, ok)
		}
		if _, err := Apply(ctx, st, cat, ch, runner.Result{OK: true, KeyCount: ch.Par, DurationMS: 1000}); err != nil {
			t.Fatal(err)
		}
	}
	// Nothing due yet: keep going with new material.
	if _, review, _ := Next(ctx, st, cat, now, base); review {
		t.Fatal("no review before anything is due")
	}
	// Two days later the skills are due: the next drill is a review.
	later := now.Add(48 * time.Hour)
	ch, review, ok := Next(ctx, st, cat, later, base)
	if !ok || !review || !ch.Review || ch.Teach != "" {
		t.Fatalf("want an interleaved review, got review=%v %+v", review, ch)
	}
	// Clearing it resets the counter.
	if _, err := Apply(ctx, st, cat, ch, runner.Result{OK: true, KeyCount: ch.Par}); err != nil {
		t.Fatal(err)
	}
	if _, review, _ := Next(ctx, st, cat, later, base); review {
		t.Fatal("counter should reset after a review")
	}
}
