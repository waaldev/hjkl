package cli

import (
	"testing"
	"time"

	"github.com/waaldev/hjkl/internal/curriculum"
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

	if got := pickReview(chs, stars, last, base); got.ID != "b" || got.Start != "b" {
		t.Fatalf("want least recently played cleared drill b, got %s %q", got.ID, got.Start)
	}
	if got := pickReview(chs, stars, last, variant); got.Start != "b2" {
		t.Fatalf("want variant b2, got %q", got.Start)
	}
	// Never-cleared drills are new material, not review.
	if got := pickReview(chs, map[string]int{"a": 1}, last, base); got.ID != "a" {
		t.Fatalf("want a, got %s", got.ID)
	}
	if got := pickReview(chs[2:], nil, nil, base); got.ID != "new" {
		t.Fatalf("fallback to uncleared, got %s", got.ID)
	}
}
