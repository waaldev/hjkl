package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/waaldev/hjkl/internal/app"
	"github.com/waaldev/hjkl/internal/curriculum"
	"github.com/waaldev/hjkl/internal/game"
	"github.com/waaldev/hjkl/internal/progress"
	"github.com/waaldev/hjkl/internal/runner"
)

func newLearnCmd() *cobra.Command {
	var noTUI bool
	cmd := &cobra.Command{
		Use:   "learn [belt]",
		Short: "Open the dojo, or play the next drill in a belt",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !noTUI && isTTY() {
				return runTUI()
			}
			return withApp(func(a *app.App) error {
				stars, err := a.Store.Stars(context.Background())
				if err != nil {
					return err
				}
				var ch curriculum.Challenge
				var ok bool
				if len(args) == 1 {
					if _, found := a.Cat.Belt(args[0]); !found {
						return failf("unknown belt %q", args[0])
					}
					unlocked := game.Unlocked(a.Cat.BeltIDs(), a.Cat.ChallengeBelts(), stars)
					if !unlocked[args[0]] {
						return failf("belt %s is locked - clear 80%% of the previous belt", args[0])
					}
					ch, ok = progress.NextInBelt(a.Cat, args[0], stars)
				} else {
					ch, ok = progress.NextChallenge(a.Cat, stars)
				}
				if !ok {
					fmt.Println("Nothing left to learn. Try hjkl daily.")
					return nil
				}
				return playOne(a, ch)
			})
		},
	}
	cmd.Flags().BoolVar(&noTUI, "no-tui", false, "run the next drill without the full TUI")
	return cmd
}

func newDrillCmd() *cobra.Command {
	var skill, id string
	var useAI bool
	cmd := &cobra.Command{
		Use:   "drill [skill]",
		Short: "Practice a skill, a challenge id, or an AI-generated drill",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 1 && skill == "" {
				skill = args[0]
			}
			return withApp(func(a *app.App) error {
				if useAI {
					return runAIDrill(a, skill)
				}
				if id != "" {
					ch, ok := a.Cat.Challenge(id)
					if !ok {
						return failf("unknown challenge %q", id)
					}
					return playOne(a, ch)
				}
				if skill == "" {
					stars, _ := a.Store.Stars(context.Background())
					ch, ok := progress.NextChallenge(a.Cat, stars)
					if !ok {
						return failf("no drills left; pass a skill or --id")
					}
					return playOne(a, ch)
				}
				chs := a.Cat.ChallengesForSkill(skill)
				if len(chs) == 0 {
					return failf("no drills for skill %q", skill)
				}
				stars, _ := a.Store.Stars(context.Background())
				for _, ch := range chs {
					if stars[ch.ID] == 0 {
						return playOne(a, ch)
					}
				}
				return playOne(a, chs[0])
			})
		},
	}
	cmd.Flags().StringVar(&skill, "skill", "", "skill id to practice")
	cmd.Flags().StringVar(&id, "id", "", "challenge id")
	cmd.Flags().BoolVar(&useAI, "ai", false, "generate a fresh drill (requires AI provider)")
	return cmd
}

func playOne(a *app.App, ch curriculum.Challenge) error {
	fmt.Printf("\n%s\n%s\npar %d\n\n", ch.Title, ch.Brief, ch.Par)
	if ch.Teach != "" {
		fmt.Println(ch.Teach)
		fmt.Println()
	}
	res, err := runner.Play(context.Background(), ch, a.Cfg.Nvim)
	if err != nil && res.Keys == "" && res.Buffer == "" {
		return err
	}
	out, aerr := progress.Apply(context.Background(), a.Store, a.Cat, ch, res)
	if aerr != nil {
		return aerr
	}
	status := "MISS"
	if res.OK {
		status = "CLEAR"
	}
	fmt.Printf("\n%s  %s  keys %d / par %d  xp +%d\n", status, starsText(out.Stars), res.KeyCount, ch.Par, out.XP)
	fmt.Printf("you  %s\npar  %s\n", res.Keys, ch.Solution)
	if out.DotScore != "" {
		fmt.Println(out.DotScore)
	}
	if p := curriculum.PrincipleText(ch.Principle); p != "" {
		fmt.Println(p)
	}
	return nil
}

func starsText(n int) string {
	s := ""
	for i := 1; i <= 3; i++ {
		if i <= n {
			s += "★"
		} else {
			s += "☆"
		}
	}
	return s
}

func isTTY() bool {
	st, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return st.Mode()&os.ModeCharDevice != 0
}

func newStatsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stats",
		Short: "XP, belts, streak, and the Grammar Grid",
		RunE: func(cmd *cobra.Command, args []string) error {
			if isTTY() {
				return runTUI()
			}
			return withApp(func(a *app.App) error {
				ctx := context.Background()
				xp, _ := a.Store.TotalXP(ctx)
				st, _ := a.Store.Streak(ctx)
				stars, _ := a.Store.Stars(ctx)
				fmt.Printf("xp %d    streak %d (best %d)\n\n", xp, st.Current, st.Best)
				for _, b := range a.Cat.Belts {
					frac := game.BeltComplete(b.ID, a.Cat.ChallengeBelts(), stars)
					fmt.Printf("%s %-8s  %3.0f%%\n", b.Glyph, b.Name, frac*100)
				}
				fmt.Println()
				return nil
			})
		},
	}
}
