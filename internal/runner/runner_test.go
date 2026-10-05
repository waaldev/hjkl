package runner

import (
	"context"
	"os/exec"
	"testing"
	"time"

	"github.com/waaldev/hjkl/internal/curriculum"
)

func TestVerifyInsertChallenge(t *testing.T) {
	if _, err := exec.LookPath("nvim"); err != nil {
		t.Skip("nvim not installed")
	}
	ch := curriculum.Challenge{
		ID:          "spike-insert",
		Belt:        "white",
		Title:       "Insert a letter",
		Type:        curriculum.TypeTransform,
		Start:       "ht",
		Target:      "hat",
		StartCursor: []int{1, 2},
		Par:         3,
		Solution:    "ia<Esc>",
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	res, err := Verify(ctx, ch, "nvim")
	if err != nil {
		t.Fatal(err)
	}
	if !res.OK {
		t.Fatalf("not ok: %+v", res)
	}
	if res.Buffer != "hat" {
		t.Fatalf("buffer=%q", res.Buffer)
	}
}

func TestVerifyRejectsWrongSolution(t *testing.T) {
	if _, err := exec.LookPath("nvim"); err != nil {
		t.Skip("nvim not installed")
	}
	ch := curriculum.Challenge{
		ID:          "spike-bad",
		Belt:        "white",
		Title:       "bad",
		Type:        curriculum.TypeTransform,
		Start:       "ht",
		Target:      "hat",
		StartCursor: []int{1, 2},
		Par:         1,
		Solution:    "l",
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	_, err := Verify(ctx, ch, "nvim")
	if err == nil {
		t.Fatal("expected verify to reject a wrong solution")
	}
}

func TestVerifyRecordsCommandKeysOnly(t *testing.T) {
	if _, err := exec.LookPath("nvim"); err != nil {
		t.Skip("nvim not installed")
	}
	ch := curriculum.Challenge{
		ID: "cmd-keys", Belt: "orange", Title: "Change a word", Type: curriculum.TypeTransform,
		Start: "foo bar", Target: "yes bar", StartCursor: []int{1, 1}, Par: 6, Solution: "cwyes<Esc>",
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	res, err := Verify(ctx, ch, "nvim")
	if err != nil {
		t.Fatal(err)
	}
	// "yes" was typed in Insert mode, so it must not look like y+e.
	if res.CmdKeys != "cw" {
		t.Fatalf("cmd_keys=%q, want %q (keys=%q)", res.CmdKeys, "cw", res.Keys)
	}
}

func TestVerifyCountsHints(t *testing.T) {
	if _, err := exec.LookPath("nvim"); err != nil {
		t.Skip("nvim not installed")
	}
	ch := curriculum.Challenge{
		ID: "hints", Belt: "orange", Title: "Operator + word", Type: curriculum.TypeTransform,
		Skills: []string{"operators"}, Hint: "d waits for a motion",
		Start: "a b", Target: "b", StartCursor: []int{1, 1}, Par: 2, Solution: "<F1><F1>dw",
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	res, err := Verify(ctx, ch, "nvim")
	if err != nil {
		t.Fatal(err)
	}
	if res.HintsUsed != 2 || res.KeyCount != 2 {
		t.Fatalf("hints_used=%d key_count=%d, want 2 and 2 (F1 is not a keystroke)", res.HintsUsed, res.KeyCount)
	}
}

func TestTargetRegister(t *testing.T) {
	if _, err := exec.LookPath("nvim"); err != nil {
		t.Skip("nvim not installed")
	}
	ch := curriculum.Challenge{
		ID: "yank", Belt: "orange", Title: "Yank", Type: curriculum.TypeTransform,
		Start: "token rest", Target: "token rest", StartCursor: []int{1, 1}, Par: 2,
		TargetRegister: &curriculum.Register{Name: `"`, Value: "token"},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	ch.Solution = "ye"
	if _, err := Verify(ctx, ch, "nvim"); err != nil {
		t.Fatalf("ye should win: %v", err)
	}
	ch.Solution = "yw" // yanks "token " - the trailing space is the lesson
	if _, err := Verify(ctx, ch, "nvim"); err == nil {
		t.Fatal("yw must not win a drill that wants exactly the word")
	}
}
