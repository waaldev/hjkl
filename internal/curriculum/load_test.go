package curriculum

import (
	"os"
	"path/filepath"
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

func TestSpeedrunNeedsTimeTarget(t *testing.T) {
	ch := Challenge{ID: "s", Type: TypeSpeedrun, Solution: "dd"}
	if err := normalizeChallenge(&ch, Belt{ID: "white"}); err == nil {
		t.Fatal("a speedrun without time_target_ms should not load")
	}
	ch.TimeTargetMS = 3000
	if err := normalizeChallenge(&ch, Belt{ID: "white"}); err != nil {
		t.Fatal(err)
	}
}

func TestLoadPersonal(t *testing.T) {
	dir := t.TempDir()
	good := `{"id":"why-1","title":"Your edit","type":"transform","skills":["dot"],"start":"a","target":"b","start_cursor":[1,1],"par":3,"solution":"rb"}`
	if err := os.WriteFile(filepath.Join(dir, "why-1.json"), []byte(good), 0o600); err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(filepath.Join(dir, "broken.json"), []byte("{"), 0o600)
	chs, err := LoadPersonal(dir)
	if err != nil || len(chs) != 1 {
		t.Fatalf("chs=%v err=%v", chs, err)
	}
	cat, _ := Load()
	n := len(cat.ChallengesForSkill("dot"))
	cat.AddPersonal(chs...)
	if len(cat.ChallengesForSkill("dot")) != n+1 {
		t.Fatal("personal drill should join reviews for its skills")
	}
	if ch, _ := cat.Challenge("why-1"); ch.Belt != PersonalBelt {
		t.Fatalf("belt=%q", ch.Belt)
	}
	if missing, err := LoadPersonal(filepath.Join(dir, "nope")); err != nil || missing != nil {
		t.Fatal("a missing drills dir is fine")
	}
}
