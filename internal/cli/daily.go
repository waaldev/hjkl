package cli

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/waaldev/hjkl/internal/app"
	"github.com/waaldev/hjkl/internal/curriculum"
	"github.com/waaldev/hjkl/internal/progress"
)

func newDailyCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "daily",
		Short: "Spaced-repetition reviews for due skills",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withApp(func(a *app.App) error {
				ctx := context.Background()
				due, err := a.Store.DueSkills(ctx, time.Now(), 5)
				if err != nil {
					return err
				}
				stars, _ := a.Store.Stars(ctx)
				if len(due) == 0 {
					if ch, ok := progress.NextChallenge(a.Cat, stars); ok {
						fmt.Println("Nothing due. Opening the next unsolved drill.")
						return playOne(a, ch)
					}
					fmt.Println("Nothing due and the dojo is clear. Take a walk.")
					return nil
				}
				fmt.Printf("%d skill(s) due - reviews hide the lesson, so recall it\n", len(due))
				last, _ := a.Store.LastAttempts(ctx)
				for _, card := range due {
					chs := a.Cat.ChallengesForSkill(card.Skill)
					if len(chs) == 0 {
						continue
					}
					ch := pickReview(chs, stars, last, rand.Intn)
					if err := play(a, ch, true); err != nil {
						return err
					}
				}
				return nil
			})
		},
	}
}

// pickReview rotates through the drills that teach a skill, least recently
// played first, and picks one of its variants at random, so a review tests
// the skill rather than the memory of one puzzle. Drills never cleared are
// skipped when a cleared one exists: a review should be recall, not new
// material.
func pickReview(chs []curriculum.Challenge, stars map[string]int, last map[string]time.Time, intn func(int) int) curriculum.Challenge {
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

func newCheatCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "cheat [query]",
		Short: "Cheatsheet of unlocked skills",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return withApp(func(a *app.App) error {
				stars, _ := a.Store.Stars(context.Background())
				unlocked := map[string]struct{}{}
				for _, s := range progress.UnlockedSkills(a.Cat, stars) {
					unlocked[s] = struct{}{}
				}
				q := ""
				if len(args) == 1 {
					q = strings.ToLower(args[0])
				}
				n := 0
				for _, d := range curriculum.SkillDocs {
					if _, ok := unlocked[d.ID]; !ok && d.Belt != "white" {
						continue
					}
					blob := strings.ToLower(d.ID + " " + d.Keys + " " + d.Summary)
					if q != "" && !strings.Contains(blob, q) {
						continue
					}
					fmt.Printf("%-22s  %s\n", d.Keys, d.Summary)
					n++
				}
				if n == 0 {
					return failf("no unlocked skills match %q", q)
				}
				return nil
			})
		},
	}
}
