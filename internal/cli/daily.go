package cli

import (
	"context"
	"fmt"
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
				fmt.Printf("%d skill(s) due\n", len(due))
				for _, card := range due {
					chs := a.Cat.ChallengesForSkill(card.Skill)
					if len(chs) == 0 {
						continue
					}
					ch := pickReview(chs, stars)
					fmt.Printf("\n- review %s -\n", card.Skill)
					if err := playOne(a, ch); err != nil {
						return err
					}
				}
				return nil
			})
		},
	}
}

func pickReview(chs []curriculum.Challenge, stars map[string]int) curriculum.Challenge {
	best := chs[0]
	bestS := 99
	for _, ch := range chs {
		s := stars[ch.ID]
		if s < bestS {
			bestS = s
			best = ch
		}
	}
	return best
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
