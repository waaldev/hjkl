package curriculum

import (
	"regexp"
	"strings"
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

func TestTechniqueRulesCompile(t *testing.T) {
	cat, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	for _, ch := range cat.Challenges {
		for _, pat := range append(append([]string{}, ch.Require...), ch.Forbid...) {
			if _, err := regexp.Compile(pat); err != nil {
				t.Fatalf("%s: %v", ch.ID, err)
			}
		}
	}
}

func TestHintLadder(t *testing.T) {
	ch := Challenge{Skills: []string{"text-objects"}, Hint: "Stand in hello and diw.", Solution: "diw"}
	l := ch.HintLadder()
	if len(l) != 3 || !strings.HasPrefix(l[0], "think: text objects") || l[1] != ch.Hint || l[2] != "answer: diw" {
		t.Fatalf("ladder=%q", l)
	}
	// A hint that is just the solution collapses into the answer rung.
	ch.Hint = "diw"
	if l := ch.HintLadder(); len(l) != 2 {
		t.Fatalf("ladder=%q", l)
	}
	r := ch.ForReview()
	if !r.Review || r.Brief != "" || r.Teach != "" || len(r.HintLadder()) != 1 || strings.Contains(r.HintLadder()[0], "diw") {
		t.Fatalf("review=%+v", r)
	}
}

func TestWithVariant(t *testing.T) {
	ch := Challenge{Start: "a b", Target: "b", Solution: "dw", Par: 2,
		Variants: []Variant{{Start: "x y z\n", Target: "z", Solution: "d2w"}}}
	v := ch.WithVariant(0)
	if v.Start != "x y z" || v.Target != "z" || v.Solution != "d2w" || v.Par != 3 {
		t.Fatalf("variant=%+v", v)
	}
	if b := ch.WithVariant(5); b.Start != "a b" {
		t.Fatal("out of range should return base")
	}
}

// Every skill needs a cheat-sheet entry: hjkl cheat and the first rung of
// the hint ladder are built from it.
func TestSkillsHaveDocs(t *testing.T) {
	cat, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	docs := map[string]bool{}
	for _, d := range SkillDocs {
		docs[d.ID] = true
	}
	for _, s := range cat.Skills() {
		if !docs[s] {
			t.Errorf("skill %q has no SkillDocs entry", s)
		}
	}
}
