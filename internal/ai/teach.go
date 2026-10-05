package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/waaldev/hjkl/internal/curriculum"
	"github.com/waaldev/hjkl/internal/game"
	"github.com/waaldev/hjkl/internal/runner"
)

const teachSystem = `You are the hjkl sensei. You teach Vim (Neovim) using Practical Vim ideas.
You NEVER type keys for the student. You explain, then they press the keys.
Stay inside skills they have unlocked, plus at most one "next step".
Be concise, concrete, and a little dry-witty. No markdown headings.`

// Debrief compares the student's run with the par solution. It gets what
// the drill taught, the keys typed as commands (not Insert-mode text), the
// hints taken and the time, so it can explain the difference instead of
// guessing.
func Debrief(ctx context.Context, p Provider, ch curriculum.Challenge, res runner.Result, unlocked []string) (string, error) {
	prompt := fmt.Sprintf(`Compare the student's keystrokes to the par solution.
Challenge: %s (%s)
Brief: %s
Lesson: %s
Principle: %s
Unlocked skills: %s
Solved: %v
Student keys (all): %q
Student command keys (Insert-mode text removed): %q
Hints taken: %d
Time: %.1fs (fluent is about %.1fs)
Par keys: %q

Explain the difference in terms of the principle. If they solved it at par
and fluently, say so in one line. One next step only.`,
		ch.Title, ch.ID, ch.Brief, strings.TrimSpace(ch.Teach), ch.Principle, strings.Join(unlocked, ", "),
		res.OK, res.Keys, res.CmdKeys, res.HintsUsed,
		float64(res.DurationMS)/1000, float64(game.FluentMS(ch.Par))/1000, ch.Solution)
	resp, err := p.Complete(ctx, Request{System: teachSystem, Prompt: prompt})
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(resp.Text), nil
}

func Ask(ctx context.Context, p Provider, question string, unlocked []string) (string, error) {
	prompt := fmt.Sprintf("Unlocked skills: %s\n\nStudent question: %s", strings.Join(unlocked, ", "), question)
	resp, err := p.Complete(ctx, Request{System: teachSystem, Prompt: prompt})
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(resp.Text), nil
}

const challengeSchema = `{
  "id": "string",
  "belt": "white|yellow|orange|green|blue|purple|brown|black",
  "title": "string",
  "type": "transform|navigate|dot|golf|boss",
  "skills": ["string"],
  "principle": "string",
  "brief": "string",
  "teach": "string",
  "hint": "string",
  "start": "string",
  "target": "string",
  "start_cursor": [1, 1],
  "target_cursor": [1, 1],
  "par": 1,
  "solution": "Vim key notation using <Esc> <CR> <C-d>",
  "language": "optional"
}`

func GenerateChallenge(ctx context.Context, p Provider, skill, language string, unlocked []string) (curriculum.Challenge, error) {
	prompt := fmt.Sprintf(`Generate ONE hjkl drill as a JSON object matching this schema:
%s

Focus skill: %s
Language flavor (comments/identifiers only; keep the buffer tiny): %s
Unlocked skills (do not require others, except at most one stretch): %s

Rules:
- ASCII only.
- par MUST equal the keystroke count of solution (<Esc> counts as 1).
- solution must actually transform start into target (or land on target_cursor for navigate).
- start and target should differ for transform/dot/golf/boss.
- Keep start under 8 lines.`, challengeSchema, skill, language, strings.Join(unlocked, ", "))
	raw, err := CompleteJSON(ctx, p, Request{System: teachSystem, Prompt: prompt})
	if err != nil {
		return curriculum.Challenge{}, err
	}
	var ch curriculum.Challenge
	if err := json.Unmarshal(raw, &ch); err != nil {
		return curriculum.Challenge{}, err
	}
	if ch.Type == "" {
		ch.Type = curriculum.TypeTransform
	}
	if len(ch.StartCursor) == 0 {
		ch.StartCursor = []int{1, 1}
	}
	return ch, nil
}

// skillSignatures are keys a solution must contain to exercise a skill. A
// "text-objects" drill solved with xxxx passes the nvim check but teaches
// nothing.
var skillSignatures = map[string]*regexp.Regexp{
	"operators":    regexp.MustCompile(`[dcy]`),
	"words":        regexp.MustCompile(`[wbeWBE]`),
	"line-ends":    regexp.MustCompile(`[0^$]`),
	"counts":       regexp.MustCompile(`[1-9]`),
	"dot":          regexp.MustCompile(`\.`),
	"find-char":    regexp.MustCompile(`[fFtT].|[;,]`),
	"text-objects": regexp.MustCompile(`[ia][wWsp"'()\[\]{}<>bBt]`),
	"percent":      regexp.MustCompile(`%`),
	"search":       regexp.MustCompile(`[/?*#nN]`),
	"gn":           regexp.MustCompile(`gn`),
	"visual":       regexp.MustCompile(`[vV]|<C-v>`),
	"registers":    regexp.MustCompile(`"[a-zA-Z0-9_]|<C-r>`),
	"macros":       regexp.MustCompile(`q[a-z].*q|@`),
	"substitute":   regexp.MustCompile(`:[%0-9.,$']*s/|&`),
	"global":       regexp.MustCompile(`:[%0-9.,$']*(g|v|norm)`),
	"dd-yy-p":      regexp.MustCompile(`dd|yy|[pP]`),
}

// CheckGenerated rejects drills that would pass the nvim check without
// teaching anything: nothing to change, nowhere to go, or a solution that
// does not use the skill being practiced.
func CheckGenerated(ch curriculum.Challenge, focus string) error {
	switch {
	case ch.Type == curriculum.TypeNavigate:
		if len(ch.TargetCursor) == 2 && len(ch.StartCursor) == 2 &&
			ch.TargetCursor[0] == ch.StartCursor[0] && ch.TargetCursor[1] == ch.StartCursor[1] {
			return fmt.Errorf("verify gate: navigate drill already starts on its target")
		}
	case ch.TargetRegister == nil && strings.TrimSpace(ch.Start) == strings.TrimSpace(ch.Target):
		return fmt.Errorf("verify gate: start and target are the same")
	}
	if re, ok := skillSignatures[focus]; ok && !re.MatchString(ch.Solution) {
		return fmt.Errorf("verify gate: solution %q does not use %s", ch.Solution, focus)
	}
	return nil
}

// VerifyGate checks a generated drill: it must teach the focus skill
// (CheckGenerated), and its solution must reach the target in nvim, within
// par, following its own technique rules.
func VerifyGate(ctx context.Context, ch curriculum.Challenge, focus, nvim string) error {
	if err := CheckGenerated(ch, focus); err != nil {
		return err
	}
	res, err := runner.Verify(ctx, ch, nvim)
	if err != nil {
		return fmt.Errorf("verify gate: %w", err)
	}
	if !res.OK {
		return fmt.Errorf("verify gate: solution did not reach the target")
	}
	if ch.Par > 0 && res.KeyCount > ch.Par {
		return fmt.Errorf("verify gate: solution took %d keys, par is %d", res.KeyCount, ch.Par)
	}
	if ok, msg, err := game.CheckTechnique(res.CmdKeys, ch.Require, ch.Forbid); err != nil || !ok {
		return fmt.Errorf("verify gate: technique rules: %v%s", err, msg)
	}
	return nil
}
