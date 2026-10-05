package ai

import (
	"context"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/waaldev/hjkl/internal/curriculum"
	"github.com/waaldev/hjkl/internal/keys"
)

// EditInput is one edit from real work: the changed lines before and
// after (with a little context), where the cursor started, and the keys
// the user pressed. Only these lines are ever sent to the provider.
type EditInput struct {
	Before   string `json:"before"`
	After    string `json:"after"`
	Cursor   []int  `json:"cursor"` // [line, col], 1-based, within Before
	Keys     string `json:"keys"`   // Vim notation, Insert-mode text included
	Filetype string `json:"filetype"`
}

// Suggestion is a shorter way to make the same edit.
type Suggestion struct {
	Keys        string   `json:"keys"`
	Explanation string   `json:"explanation"`
	Principle   string   `json:"principle"`
	Skills      []string `json:"skills"`
}

// MaxSuggestLines caps how much of a file one request may contain.
const MaxSuggestLines = 40

const suggestAttempts = 3

// Verifier replays a challenge's solution and reports whether it reaches
// the target (runner.Verify in production).
type Verifier func(ctx context.Context, ch curriculum.Challenge) error

// SuggestEdit asks the provider for a shorter key sequence for a real
// edit. A suggestion is returned only if replaying it in nvim turns Before
// into exactly After, in fewer keys than the user pressed. Failed attempts
// are fed back to the model, up to suggestAttempts times.
func SuggestEdit(ctx context.Context, p Provider, in EditInput, unlocked []string, verify Verifier) (Suggestion, curriculum.Challenge, error) {
	if err := checkEditInput(in); err != nil {
		return Suggestion{}, curriculum.Challenge{}, err
	}
	userKeys := keys.Count(in.Keys)
	feedback := ""
	var lastErr error
	for attempt := 0; attempt < suggestAttempts; attempt++ {
		raw, err := CompleteJSON(ctx, p, Request{System: suggestSystem, Prompt: suggestPrompt(in, unlocked) + feedback})
		if err != nil {
			return Suggestion{}, curriculum.Challenge{}, err
		}
		var s Suggestion
		if err := json.Unmarshal(raw, &s); err != nil || strings.TrimSpace(s.Keys) == "" {
			lastErr = fmt.Errorf("unusable reply")
			feedback = "\n\nYour previous reply had no usable keys field. Reply with the JSON object only."
			continue
		}
		ch := editChallenge(in, s)
		if n := keys.Count(s.Keys); n >= userKeys {
			lastErr = fmt.Errorf("suggestion %q is not shorter (%d keys vs %d)", s.Keys, n, userKeys)
			feedback = fmt.Sprintf("\n\nYour previous answer %q took %d keys; the user took %d. Find a shorter sequence, or reply with the shortest you can verify.", s.Keys, n, userKeys)
			continue
		}
		if err := verify(ctx, ch); err != nil {
			lastErr = err
			feedback = fmt.Sprintf("\n\nYour previous answer %q was replayed in Neovim and did not produce the AFTER text (%v). Check it key by key.", s.Keys, err)
			continue
		}
		return s, ch, nil
	}
	return Suggestion{}, curriculum.Challenge{}, fmt.Errorf("no verified shorter way found: %v", lastErr)
}

func checkEditInput(in EditInput) error {
	switch {
	case in.Before == in.After:
		return fmt.Errorf("no change to improve")
	case strings.Count(in.Before, "\n") >= MaxSuggestLines || strings.Count(in.After, "\n") >= MaxSuggestLines:
		return fmt.Errorf("edit spans more than %d lines", MaxSuggestLines)
	case keys.Count(in.Keys) < 2:
		return fmt.Errorf("edit is already one key")
	case len(in.Cursor) != 2:
		return fmt.Errorf("missing start cursor")
	}
	return nil
}

const suggestSystem = `You are the hjkl sensei, a Neovim efficiency coach. A student made an edit in
their own project. Find a shorter Normal-mode key sequence that makes exactly
the same edit, starting from the same cursor position. Prefer sequences that
teach a reusable habit (operator + text object, the dot formula, search then
cgn) over clever one-offs. Reply with one JSON object only.`

func suggestPrompt(in EditInput, unlocked []string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Filetype: %s\n", in.Filetype)
	fmt.Fprintf(&b, "Cursor starts at line %d, column %d of BEFORE (1-based), in Normal mode.\n\n", in.Cursor[0], in.Cursor[1])
	b.WriteString("BEFORE:\n" + numbered(in.Before) + "\n\nAFTER:\n" + numbered(in.After) + "\n\n")
	fmt.Fprintf(&b, "The student pressed %d keys: %s\n", keys.Count(in.Keys), in.Keys)
	fmt.Fprintf(&b, "Skills they have learned: %s (you may use others if clearly better).\n\n", strings.Join(unlocked, ", "))
	b.WriteString(`Reply with:
{"keys": "Vim key notation, e.g. ciwnew<Esc>j. (<Esc>, <CR>, <C-v> count as one key)",
 "explanation": "two short sentences: what the keys do and why they are shorter",
 "principle": "dot | grammar | text-objects | gn | ex-ranges | macros | registers",
 "skills": ["skill ids used, e.g. text-objects, dot, find-char, search"]}
The keys must end in Normal mode and turn BEFORE into exactly AFTER.`)
	return b.String()
}

func numbered(s string) string {
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		lines[i] = fmt.Sprintf("%3d| %s", i+1, l)
	}
	return strings.Join(lines, "\n")
}

// editChallenge turns a real edit into a drill so the shorter way can be
// practiced, and saved for spaced repetition.
func editChallenge(in EditInput, s Suggestion) curriculum.Challenge {
	sum := sha1.Sum([]byte(in.Before + "\x00" + in.After))
	title := strings.TrimSpace(firstChangedLine(in.Before, in.After))
	if len(title) > 32 {
		title = title[:32] + "…"
	}
	skills := knownSkills(s.Skills)
	return curriculum.Challenge{
		ID:          fmt.Sprintf("why-%x", sum[:5]),
		Belt:        "personal",
		Title:       "Your edit: " + title,
		Type:        curriculum.TypeTransform,
		Skills:      skills,
		Principle:   s.Principle,
		Brief:       "Make the edit you made in your project - the short way.",
		Teach:       s.Explanation,
		Start:       in.Before,
		Target:      in.After,
		StartCursor: in.Cursor,
		Par:         keys.Count(s.Keys),
		Solution:    s.Keys,
		Language:    in.Filetype,
	}
}

func firstChangedLine(before, after string) string {
	b, a := strings.Split(before, "\n"), strings.Split(after, "\n")
	for i := range b {
		if i >= len(a) || a[i] != b[i] {
			return b[i]
		}
	}
	return b[len(b)-1]
}

func knownSkills(in []string) []string {
	var out []string
	for _, s := range in {
		for _, d := range curriculum.SkillDocs {
			if d.ID == s {
				out = append(out, s)
				break
			}
		}
	}
	if len(out) == 0 {
		out = []string{"golf"}
	}
	return out
}
