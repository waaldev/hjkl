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

// BossRequest asks for a boss fight on a block of the player's own code.
type BossRequest struct {
	Code     string
	Language string
	Unlocked []string
}

// MaxBossLines caps how much of a file one boss request may contain.
const MaxBossLines = 30

// minBossKeys keeps a boss multi-step: a single cw is not a boss.
const minBossKeys = 8

// GenerateBoss asks the provider to invent a realistic multi-step refactor
// of the given code, using at least two skills. It is returned only after
// it passes the gate and verify (replayed in nvim in production).
func GenerateBoss(ctx context.Context, p Provider, req BossRequest, verify Verifier) (curriculum.Challenge, error) {
	if n := strings.Count(req.Code, "\n") + 1; n > MaxBossLines {
		return curriculum.Challenge{}, fmt.Errorf("block is %d lines; at most %d", n, MaxBossLines)
	}
	if strings.TrimSpace(req.Code) == "" {
		return curriculum.Challenge{}, fmt.Errorf("block is empty")
	}
	return gated(ctx, p, bossPrompt(req), func(raw json.RawMessage) (curriculum.Challenge, error) {
		var ch curriculum.Challenge
		if err := json.Unmarshal(raw, &ch); err != nil {
			return ch, fmt.Errorf("reply did not match the schema")
		}
		ch = normalizeBoss(ch, req)
		real := 0
		for _, s := range ch.Skills {
			if s != "golf" {
				real++
			}
		}
		switch {
		case real < 2:
			return ch, fmt.Errorf("a boss needs at least two skills, got %v", ch.Skills)
		case keys.Count(ch.Solution) < minBossKeys:
			return ch, fmt.Errorf("a boss needs several steps; the solution is only %d keys", keys.Count(ch.Solution))
		}
		return ch, CheckGenerated(ch, ch.Skills[0])
	}, verify)
}

func bossPrompt(req BossRequest) string {
	return fmt.Sprintf(`Here is a block of the student's own %s code. Invent a realistic, small
refactor of it that takes several steps and at least two different Vim skills
(for example: rename with cgn and ., change inside quotes, delete lines with :g,
append with A and .). The edit must make sense for this code.

CODE (the drill starts with exactly this text, cursor at line 1, column 1):
%s

Unlocked skills (prefer these; at most one stretch): %s
Known skill ids: %s

Reply with one JSON object:
{"title": "short name of the refactor",
 "brief": "what to change, in plain words, without naming keys",
 "teach": "two sentences on the skills it combines",
 "target": "the full text after the refactor",
 "solution": "Vim key notation from line 1 col 1, ending in Normal mode",
 "skills": ["skill ids used, most important first"],
 "principle": "dot | grammar | text-objects | gn | ex-ranges | macros | registers",
 "hints": ["a nudge", "a stronger nudge"]}
The solution must turn CODE into exactly target.`, req.Language, numbered(req.Code), strings.Join(req.Unlocked, ", "), strings.Join(skillIDs(), ", "))
}

func normalizeBoss(ch curriculum.Challenge, req BossRequest) curriculum.Challenge {
	ch.Start = req.Code
	ch.Target = strings.TrimSuffix(strings.ReplaceAll(ch.Target, "\r\n", "\n"), "\n")
	ch.StartCursor = []int{1, 1}
	ch.TargetCursor = nil
	ch.Type = curriculum.TypeBoss
	ch.Belt = curriculum.PersonalBelt
	ch.Language = req.Language
	ch.Skills = knownSkills(ch.Skills)
	ch.Par = keys.Count(ch.Solution)
	if ch.Title == "" {
		ch.Title = "Refactor"
	}
	ch.Title = "Boss: " + ch.Title
	sum := sha1.Sum([]byte(ch.Start + "\x00" + ch.Target + "\x00" + ch.Solution))
	ch.ID = fmt.Sprintf("boss-%x", sum[:5])
	if ch.Hints != nil {
		ch.Hints = append(ch.Hints, "answer: "+ch.Solution)
	}
	return ch
}
