package progress

import (
	"context"
	"strconv"
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
	Technique  string // set when a require/forbid rule capped the stars
	HintsUsed  int
	Fluent     bool // at par and within FluentMS
}

func Apply(ctx context.Context, st *store.Store, cat *curriculum.Catalog, ch curriculum.Challenge, res runner.Result) (Outcome, error) {
	beltRank := 0
	if b, ok := cat.Belt(ch.Belt); ok {
		beltRank = b.Rank
	}
	stars := game.Stars(res.OK, res.KeyCount, ch.Par)
	technique := ""
	if res.OK {
		ok, msg, err := game.CheckTechnique(res.CmdKeys, ch.Require, ch.Forbid)
		if err != nil {
			return Outcome{}, err
		}
		if !ok {
			if stars > 1 {
				stars = 1
			}
			if ch.Technique != "" {
				msg = "solved, but " + ch.Technique
			}
		}
		technique = msg
	}
	stars = game.HintCap(stars, res.HintsUsed)
	sawAnswer := !ch.Review && res.HintsUsed >= len(ch.HintLadder())
	quality := game.HintQuality(game.Quality(res.OK, stars, res.DurationMS, ch.Par), res.HintsUsed, sawAnswer)

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

	// Track new clears since the last review, for interleaving (Next).
	switch {
	case ch.Review && res.OK:
		_ = st.SetMeta(ctx, metaNewSinceReview, "0")
	case first:
		v, _ := st.Meta(ctx, metaNewSinceReview)
		_ = st.SetMeta(ctx, metaNewSinceReview, strconv.Itoa(atoi(v)+1))
	}

	now := time.Now()
	for _, skill := range ch.Skills {
		if err := ReviewSkill(ctx, st, skill, quality, now, true); err != nil {
			return Outcome{}, err
		}
	}

	// The grid shows what you actually typed: your own command keys, on
	// success only. Never the canonical solution, never Insert-mode text.
	if res.OK {
		for _, pair := range game.LightFromKeys(res.CmdKeys) {
			_ = st.LightGrammar(ctx, pair[0], pair[1])
		}
	}

	return Outcome{
		Stars:      stars,
		XP:         xp,
		FirstClear: first,
		Quality:    quality,
		DotScore:   game.DotScore(res.CmdKeys, ch.Principle),
		Technique:  technique,
		HintsUsed:  res.HintsUsed,
		Fluent:     res.OK && stars == 3 && res.DurationMS <= game.FluentMS(ch.Par),
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

// ReviewSkill schedules one SRS review for a skill, at most once per day (see
// srs.Counts). With create=false a skill that has no card yet is left alone,
// so real-world use never schedules a skill the dojo hasn't taught.
func ReviewSkill(ctx context.Context, st *store.Store, skill string, quality int, now time.Time, create bool) error {
	card, ok, err := st.SRS(ctx, skill)
	if err != nil {
		return err
	}
	var sc srs.Card
	switch {
	case ok:
		sc = srs.Card{
			Skill: card.Skill, Easiness: card.Easiness, IntervalDays: card.IntervalDays,
			Repetitions: card.Repetitions, DueAt: card.DueAt, LastQuality: card.LastQuality, LastReviewAt: card.LastReviewAt,
		}
	case create:
		sc = srs.NewCard(skill, now)
	default:
		return nil
	}
	if !srs.Counts(sc, quality, now) {
		return nil
	}
	sc = srs.Review(sc, quality, now)
	return st.UpsertSRS(ctx, store.SRSCard{
		Skill: sc.Skill, Easiness: sc.Easiness, IntervalDays: sc.IntervalDays,
		Repetitions: sc.Repetitions, DueAt: sc.DueAt, LastQuality: sc.LastQuality, LastReviewAt: sc.LastReviewAt,
	})
}

// PickReview rotates through the drills that teach a skill, least recently
// played first, and picks one of its variants at random, so a review tests
// the skill rather than the memory of one puzzle. Drills never cleared are
// skipped when a cleared one exists: a review should be recall, not new
// material.
func PickReview(chs []curriculum.Challenge, stars map[string]int, last map[string]time.Time, intn func(int) int) curriculum.Challenge {
	pool := chs[:0:0]
	for _, ch := range chs {
		if stars[ch.ID] > 0 {
			pool = append(pool, ch)
		}
	}
	if len(pool) == 0 {
		pool = chs
	}
	best := pool[0]
	for _, ch := range pool[1:] {
		if last[ch.ID].Before(last[best.ID]) {
			best = ch
		}
	}
	// -1 is the base drill; 0..n-1 are its variants.
	return best.WithVariant(intn(len(best.Variants)+1) - 1)
}

// InterleaveEvery is how many new drills are cleared before "next" serves
// a due review. Mixing old skills into new practice beats learning in
// blocks for long-term retention.
const InterleaveEvery = 3

const metaNewSinceReview = "new_since_review"

// Next returns the next drill to play: usually the next new drill, but
// after InterleaveEvery new clears, a review of a due skill (review=true;
// the challenge is already in review form).
func Next(ctx context.Context, st *store.Store, cat *curriculum.Catalog, now time.Time, intn func(int) int) (ch curriculum.Challenge, review bool, ok bool) {
	stars, _ := st.Stars(ctx)
	if v, _ := st.Meta(ctx, metaNewSinceReview); atoi(v) >= InterleaveEvery {
		if due, _ := st.DueSkills(ctx, now, 1); len(due) > 0 {
			if chs := cat.ChallengesForSkill(due[0].Skill); len(chs) > 0 {
				last, _ := st.LastAttempts(ctx)
				return PickReview(chs, stars, last, intn).ForReview(), true, true
			}
		}
	}
	ch, ok = NextChallenge(cat, stars)
	return ch, false, ok
}

func atoi(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}
