package ai

import (
	"context"
	"os/exec"
	"strings"
	"testing"

	"github.com/waaldev/hjkl/internal/curriculum"
	"github.com/waaldev/hjkl/internal/runner"
)

// scripted replies with one canned answer per call and records prompts.
type scripted struct {
	replies []string
	prompts []string
}

func (s *scripted) Name() string { return "scripted" }

func (s *scripted) Complete(_ context.Context, req Request) (Response, error) {
	s.prompts = append(s.prompts, req.Prompt)
	r := s.replies[0]
	if len(s.replies) > 1 {
		s.replies = s.replies[1:]
	}
	return Response{Text: r}, nil
}

func nvimVerify(t *testing.T) Verifier {
	t.Helper()
	if _, err := exec.LookPath("nvim"); err != nil {
		t.Skip("nvim not installed")
	}
	return func(ctx context.Context, ch curriculum.Challenge) error {
		_, err := runner.Verify(ctx, ch, "nvim")
		return err
	}
}

// The user renamed three foo's by retyping each one.
var renameEdit = EditInput{
	Before:   "foo := 1\nfoo := 2\nfoo := 3",
	After:    "bar := 1\nbar := 2\nbar := 3",
	Cursor:   []int{1, 1},
	Keys:     "xxxibar<Esc>jhhxxxibar<Esc>jhhxxxibar<Esc>",
	Filetype: "go",
}

func TestSuggestEditVerified(t *testing.T) {
	p := &scripted{replies: []string{`{"keys":"ciwbar<Esc>j.j.","explanation":"ciw changes the whole word; . repeats it.","principle":"dot","skills":["text-objects","dot"]}`}}
	s, ch, err := SuggestEdit(context.Background(), p, renameEdit, []string{"operators"}, nvimVerify(t))
	if err != nil {
		t.Fatal(err)
	}
	if s.Keys != "ciwbar<Esc>j.j." || ch.Belt != "personal" || ch.Par != 11 || ch.Start != renameEdit.Before {
		t.Fatalf("suggestion=%+v challenge=%+v", s, ch)
	}
	if !strings.Contains(p.prompts[0], "BEFORE:") || !strings.Contains(p.prompts[0], "  1| foo := 1") {
		t.Fatalf("prompt should carry the numbered snippet:\n%s", p.prompts[0])
	}
}

func TestSuggestEditRetriesWrongAnswers(t *testing.T) {
	p := &scripted{replies: []string{
		`{"keys":"cwbar<Esc>j.j."}`, // cw from mid-word: wrong result in nvim
		`{"keys":"ciwbar<Esc>j.j.","explanation":"x","skills":["dot"]}`,
	}}
	s, _, err := SuggestEdit(context.Background(), p, renameEdit, nil, nvimVerify(t))
	if err != nil {
		t.Fatal(err)
	}
	if s.Keys != "ciwbar<Esc>j.j." || len(p.prompts) != 2 || !strings.Contains(p.prompts[1], "did not produce the AFTER text") {
		t.Fatalf("want a retry with feedback; prompts=%d keys=%q", len(p.prompts), s.Keys)
	}
}

func TestSuggestEditRejectsLongerOrUnverified(t *testing.T) {
	never := func(context.Context, curriculum.Challenge) error { return nil }
	long := &scripted{replies: []string{`{"keys":"` + renameEdit.Keys + `x"}`}}
	if _, _, err := SuggestEdit(context.Background(), long, renameEdit, nil, never); err == nil {
		t.Fatal("a suggestion that is not shorter must be rejected")
	}
	bad := &scripted{replies: []string{`{"keys":"dd"}`}}
	if _, _, err := SuggestEdit(context.Background(), bad, renameEdit, nil, nvimVerify(t)); err == nil {
		t.Fatal("a suggestion nvim does not confirm must be rejected")
	}
	if len(bad.prompts) != suggestAttempts {
		t.Fatalf("attempts=%d", len(bad.prompts))
	}
}

func TestSuggestEditInputLimits(t *testing.T) {
	never := func(context.Context, curriculum.Challenge) error { return nil }
	p := &scripted{replies: []string{`{"keys":"x"}`}}
	big := renameEdit
	big.Before = strings.Repeat("x\n", MaxSuggestLines+1)
	for name, in := range map[string]EditInput{
		"no change": {Before: "a", After: "a", Cursor: []int{1, 1}, Keys: "ab"},
		"too big":   big,
	} {
		if _, _, err := SuggestEdit(context.Background(), p, in, nil, never); err == nil {
			t.Errorf("%s: expected rejection", name)
		}
	}
	if len(p.prompts) != 0 {
		t.Fatal("rejected input must never reach the provider")
	}
}
