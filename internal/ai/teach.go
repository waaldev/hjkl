package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/waaldev/hjkl/internal/curriculum"
	"github.com/waaldev/hjkl/internal/runner"
)

const teachSystem = `You are the hjkl sensei. You teach Vim (Neovim) using Practical Vim ideas.
You NEVER type keys for the student. You explain, then they press the keys.
Stay inside skills they have unlocked, plus at most one "next step".
Be concise, concrete, and a little dry-witty. No markdown headings.`

func Debrief(ctx context.Context, p Provider, ch curriculum.Challenge, userKeys, parKeys string, unlocked []string) (string, error) {
	prompt := fmt.Sprintf(`Compare the student's keystrokes to the par solution.
Challenge: %s (%s)
Brief: %s
Principle: %s
Unlocked skills: %s
Student keys: %q
Par keys: %q

Explain the difference in terms of the principle. One next-step only.`,
		ch.Title, ch.ID, ch.Brief, ch.Principle, strings.Join(unlocked, ", "), userKeys, parKeys)
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

// VerifyGate runs the challenge solution headlessly. Rejects wrong or over-par solutions.
func VerifyGate(ctx context.Context, ch curriculum.Challenge, nvim string) error {
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
	return nil
}
