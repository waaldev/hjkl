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
