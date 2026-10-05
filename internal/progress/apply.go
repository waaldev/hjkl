package progress

import (
	"context"
	"time"

	"github.com/waaldev/hjkl/internal/curriculum"
	"github.com/waaldev/hjkl/internal/game"
	"github.com/waaldev/hjkl/internal/runner"
	"github.com/waaldev/hjkl/internal/srs"
	"github.com/waaldev/hjkl/internal/store"
)

type Outcome struct {
	Stars      int
	XP         int
	FirstClear bool
	Quality    int
	DotScore   string
}

func Apply(ctx context.Context, st *store.Store, cat *curriculum.Catalog, ch curriculum.Challenge, res runner.Result) (Outcome, error) {
	beltRank := 0
	if b, ok := cat.Belt(ch.Belt); ok {
		beltRank = b.Rank
	}
	stars := game.Stars(res.OK, res.KeyCount, ch.Par)
	quality := game.Quality(res.OK, stars, res.KeyCount, ch.Par)

	prev, _ := st.Progress(ctx)
	first := res.OK && (prev[ch.ID].FirstClearAt == nil)
	xp := game.XP(beltRank, stars, first)
	if !res.OK {
		xp = 0
		first = false
	}

	_, err := st.RecordAttempt(ctx, store.Attempt{
		ChallengeID: ch.ID,
		OK:          res.OK,
		Keys:        res.Keys,
		KeyCount:    res.KeyCount,
		Par:         ch.Par,
		Stars:       stars,
		XP:          xp,
		DurationMS:  res.DurationMS,
		At:          time.Now(),
	})
	if err != nil {
		return Outcome{}, err
	}

	now := time.Now()
	for _, skill := range ch.Skills {
		card, ok, err := st.SRS(ctx, skill)
		if err != nil {
			return Outcome{}, err
		}
		var sc srs.Card
		if ok {
			sc = srs.Card{
				Skill: card.Skill, Easiness: card.Easiness, IntervalDays: card.IntervalDays,
				Repetitions: card.Repetitions, DueAt: card.DueAt, LastQuality: card.LastQuality, LastReviewAt: card.LastReviewAt,
			}
		} else {
			sc = srs.NewCard(skill, now)
		}
		sc = srs.Review(sc, quality, now)
		if err := st.UpsertSRS(ctx, store.SRSCard{
			Skill: sc.Skill, Easiness: sc.Easiness, IntervalDays: sc.IntervalDays,
			Repetitions: sc.Repetitions, DueAt: sc.DueAt, LastQuality: sc.LastQuality, LastReviewAt: sc.LastReviewAt,
		}); err != nil {
			return Outcome{}, err
		}
	}

	seq := res.Keys
	if seq == "" {
		seq = ch.Solution
	}
	for _, pair := range game.LightFromKeys(seq) {
		_ = st.LightGrammar(ctx, pair[0], pair[1])
	}
	if res.OK {
		for _, pair := range game.LightFromKeys(ch.Solution) {
			_ = st.LightGrammar(ctx, pair[0], pair[1])
		}
	}

	return Outcome{
		Stars:      stars,
		XP:         xp,
		FirstClear: first,
		Quality:    quality,
		DotScore:   game.DotScore(res.Keys, ch.Principle),
	}, nil
}

func UnlockedSkills(cat *curriculum.Catalog, stars map[string]int) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, ch := range cat.Challenges {
		if stars[ch.ID] <= 0 {
			continue
		}
		for _, s := range ch.Skills {
			if _, ok := seen[s]; ok {
				continue
			}
			seen[s] = struct{}{}
			out = append(out, s)
		}
	}
	// White-belt basics are always visible on the cheatsheet.
	for _, s := range []string{"modes", "insert", "hjkl"} {
		if _, ok := seen[s]; !ok {
			out = append([]string{s}, out...)
			seen[s] = struct{}{}
		}
	}
	return out
}

func NextChallenge(cat *curriculum.Catalog, stars map[string]int) (curriculum.Challenge, bool) {
	unlocked := game.Unlocked(cat.BeltIDs(), cat.ChallengeBelts(), stars)
	for _, ch := range cat.Challenges {
		if !unlocked[ch.Belt] {
			continue
		}
		if stars[ch.ID] == 0 {
			return ch, true
		}
	}
	return curriculum.Challenge{}, false
}

func NextInBelt(cat *curriculum.Catalog, belt string, stars map[string]int) (curriculum.Challenge, bool) {
	for _, ch := range cat.ChallengesForBelt(belt) {
		if stars[ch.ID] == 0 {
			return ch, true
		}
	}
	return curriculum.Challenge{}, false
}
