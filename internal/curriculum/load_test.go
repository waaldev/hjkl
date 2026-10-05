package curriculum

import (
	"testing"

	"github.com/waaldev/hjkl/internal/keys"
)

func TestLoad(t *testing.T) {
	cat, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(cat.Belts) != 8 {
		t.Fatalf("belts=%d want 8", len(cat.Belts))
	}
	ids := map[string]struct{}{}
	var wg int
	for _, ch := range cat.Challenges {
		if _, ok := ids[ch.ID]; ok {
			t.Fatalf("duplicate id %s", ch.ID)
		}
		ids[ch.ID] = struct{}{}
		if ch.Solution == "" {
			t.Fatalf("%s missing solution", ch.ID)
		}
		if ch.Par <= 0 {
			t.Fatalf("%s par=%d", ch.ID, ch.Par)
		}
		if keys.Count(ch.Solution) == 0 {
			t.Fatalf("%s empty solution parse", ch.ID)
		}
		switch ch.Belt {
		case "white", "yellow", "orange", "green":
			wg++
		}
	}
	if wg < 50 {
		t.Fatalf("white-green challenges=%d want >= 50", wg)
	}
	if _, ok := cat.Challenge("white-01-insert"); !ok {
		t.Fatal("missing white-01-insert")
	}
}
