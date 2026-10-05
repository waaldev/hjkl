package ai

import (
	"context"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/waaldev/hjkl/internal/curriculum"
	"github.com/waaldev/hjkl/internal/game"
	"github.com/waaldev/hjkl/internal/keys"
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

// DrillRequest asks for one drill: either for a skill, or for the
// technique that answers a student's question (Question and Answer set).
type DrillRequest struct {
	Skill    string
	Question string
	Answer   string
	Language string
	Unlocked []string
}

const drillAttempts = 3

// GenerateDrill asks the provider for a drill and returns it only once it
// passes the gate: it must teach something (CheckGenerated) and verify must
// accept it (in production: replayed in nvim, within par, following its own
// rules; see NvimVerifier). Failures are fed back to the model.
func GenerateDrill(ctx context.Context, p Provider, req DrillRequest, verify Verifier) (curriculum.Challenge, error) {
	return gated(ctx, p, drillPrompt(req), func(raw json.RawMessage) (curriculum.Challenge, error) {
		var ch curriculum.Challenge
		if err := json.Unmarshal(raw, &ch); err != nil {
			return ch, fmt.Errorf("reply did not match the schema")
		}
		ch = normalizeDrill(ch, req.Skill)
		return ch, CheckGenerated(ch, ch.Skills[0])
	}, verify)
}

// gated asks for a drill until one passes: build parses and checks the
// reply, verify replays it. Every rejection is fed back to the model, up
// to drillAttempts times.
func gated(ctx context.Context, p Provider, prompt string, build func(json.RawMessage) (curriculum.Challenge, error), verify Verifier) (curriculum.Challenge, error) {
	feedback := ""
	var lastErr error
	for attempt := 0; attempt < drillAttempts; attempt++ {
		raw, err := CompleteJSON(ctx, p, Request{System: teachSystem, Prompt: prompt + feedback})
		if err != nil {
			return curriculum.Challenge{}, err
		}
		ch, err := build(raw)
		if err == nil {
			err = verify(ctx, ch)
		}
		if err != nil {
			lastErr = err
			feedback = fmt.Sprintf("\n\nYour previous drill was rejected: %v. Its solution was %q. Fix it.", err, ch.Solution)
			continue
		}
		return ch, nil
	}
	return curriculum.Challenge{}, fmt.Errorf("no drill passed the check: %v", lastErr)
}

func drillPrompt(req DrillRequest) string {
	focus := "Focus skill: " + req.Skill
	if req.Question != "" {
		focus = fmt.Sprintf("Focus: the technique that answers this question.\nQuestion: %s\nThe sensei's answer: %s\nPut the skill ids it uses in skills, most important first.", req.Question, req.Answer)
	}
	return fmt.Sprintf(`Generate ONE hjkl drill as a JSON object matching this schema:
%s

%s
Language flavor (comments/identifiers only; keep the buffer tiny): %s
Unlocked skills (do not require others, except at most one stretch): %s
Known skill ids: %s

Rules:
- ASCII only.
- par MUST equal the keystroke count of solution (<Esc> counts as 1).
- solution must actually transform start into target (or land on target_cursor for navigate), ending in Normal mode.
- start and target must differ for transform/dot/golf/boss.
- Keep start under 8 lines.`, challengeSchema, focus, req.Language, strings.Join(req.Unlocked, ", "), strings.Join(skillIDs(), ", "))
}

// normalizeDrill gives a generated drill a stable personal id and known
// skills (the requested skill first), so it can be saved and reviewed.
func normalizeDrill(ch curriculum.Challenge, focus string) curriculum.Challenge {
	skills := knownSkills(ch.Skills)
	if focus != "" && !contains(skills, focus) {
		skills = append([]string{focus}, skills...)
	}
	if len(skills) > 1 && skills[len(skills)-1] == "golf" && !contains(ch.Skills, "golf") {
		skills = skills[:len(skills)-1] // knownSkills' fallback, not needed
	}
	ch.Skills = skills
	sum := sha1.Sum([]byte(ch.Start + "\x00" + ch.Target + "\x00" + ch.Solution))
	ch.ID = fmt.Sprintf("ai-%s-%x", skills[0], sum[:4])
	ch.Belt = curriculum.PersonalBelt
	if ch.Type == "" {
		ch.Type = curriculum.TypeTransform
	}
	if len(ch.StartCursor) != 2 {
		ch.StartCursor = []int{1, 1}
	}
	if ch.Par <= 0 {
		ch.Par = keys.Count(ch.Solution)
	}
	return ch
}

func skillIDs() []string {
	out := make([]string, 0, len(curriculum.SkillDocs))
	for _, d := range curriculum.SkillDocs {
		out = append(out, d.ID)
	}
	return out
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
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

// NvimVerifier replays a drill's solution in a headless nvim and checks it
// reaches the target within par and follows the drill's own rules.
func NvimVerifier(nvim string) Verifier {
	return func(ctx context.Context, ch curriculum.Challenge) error {
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
}

// VerifyGate checks a generated drill: it must teach the focus skill
// (CheckGenerated) and pass NvimVerifier.
func VerifyGate(ctx context.Context, ch curriculum.Challenge, focus, nvim string) error {
	if err := CheckGenerated(ch, focus); err != nil {
		return err
	}
	return NvimVerifier(nvim)(ctx, ch)
}
