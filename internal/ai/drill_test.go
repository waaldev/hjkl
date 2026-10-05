package ai

import (
	"context"
	"strings"
	"testing"

	"github.com/waaldev/hjkl/internal/curriculum"
)

const goodDrill = `{"title":"Delete inner word","type":"transform","skills":["text-objects"],"brief":"Remove hello.",
"start":"say hello now","target":"say  now","start_cursor":[1,6],"par":3,"solution":"diw"}`

func TestGenerateDrillForSkill(t *testing.T) {
	p := &scripted{replies: []string{goodDrill}}
	ch, err := GenerateDrill(context.Background(), p, DrillRequest{Skill: "text-objects", Language: "go"}, nvimVerify(t))
	if err != nil {
		t.Fatal(err)
	}
	if ch.Belt != curriculum.PersonalBelt || !strings.HasPrefix(ch.ID, "ai-text-objects-") || ch.Skills[0] != "text-objects" {
		t.Fatalf("drill=%+v", ch)
	}
	// Same drill, same id: saving it twice does not duplicate it.
	again, _ := GenerateDrill(context.Background(), &scripted{replies: []string{goodDrill}}, DrillRequest{Skill: "text-objects"}, nvimVerify(t))
	if again.ID != ch.ID {
		t.Fatalf("ids differ: %s %s", ch.ID, again.ID)
	}
}

func TestGenerateDrillRetriesRejectedDrills(t *testing.T) {
	p := &scripted{replies: []string{
		// Passes nvim but does not use the skill: rejected before nvim runs.
		`{"type":"transform","skills":["text-objects"],"start":"say hello now","target":"say  now","start_cursor":[1,5],"par":6,"solution":"lxxxxx"}`,
		goodDrill,
	}}
	ch, err := GenerateDrill(context.Background(), p, DrillRequest{Skill: "text-objects"}, nvimVerify(t))
	if err != nil {
		t.Fatal(err)
	}
	if ch.Solution != "diw" || len(p.prompts) != 2 || !strings.Contains(p.prompts[1], "was rejected") {
		t.Fatalf("want a retry with feedback; prompts=%d solution=%q", len(p.prompts), ch.Solution)
	}
}

func TestGenerateDrillGivesUp(t *testing.T) {
	broken := `{"type":"transform","skills":["text-objects"],"start":"say hello now","target":"say  now","start_cursor":[1,6],"par":3,"solution":"daw"}`
	p := &scripted{replies: []string{broken}}
	if _, err := GenerateDrill(context.Background(), p, DrillRequest{Skill: "text-objects"}, nvimVerify(t)); err == nil {
		t.Fatal("a drill nvim never confirms must not be returned")
	}
	if len(p.prompts) != drillAttempts {
		t.Fatalf("attempts=%d", len(p.prompts))
	}
}

func TestGenerateDrillFromQuestion(t *testing.T) {
	p := &scripted{replies: []string{goodDrill}}
	req := DrillRequest{Question: "how do I delete a word from the middle?", Answer: "Use diw."}
	ch, err := GenerateDrill(context.Background(), p, req, nvimVerify(t))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(p.prompts[0], req.Question) || !strings.Contains(p.prompts[0], req.Answer) {
		t.Fatalf("prompt should carry the question and answer:\n%s", p.prompts[0])
	}
	if ch.Skills[0] != "text-objects" {
		t.Fatalf("skills come from the drill: %v", ch.Skills)
	}
}
