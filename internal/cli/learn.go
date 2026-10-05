package cli

import (
	"bufio"
	"context"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/waaldev/hjkl/internal/ai"
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
				var ok, review bool
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
					ch, review, ok = progress.Next(context.Background(), a.Store, a.Cat, time.Now(), rand.Intn)
				}
				if !ok {
					fmt.Println("Nothing left to learn. Try hjkl daily.")
					return nil
				}
				if review {
					fmt.Println("Interleaved review: an older skill between new ones.")
				}
				return play(a, ch, review)
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
	return play(a, ch, false)
}

// play runs a drill, shows the results card, and offers "show me" and
// "retry" while the run is short of three stars. Watching the par solution
// and then doing it yourself straight away is the most valuable repetition.
func play(a *app.App, ch curriculum.Challenge, review bool) error {
	if review {
		ch = ch.ForReview()
	}
	in := bufio.NewReader(os.Stdin)
	for {
		if review {
			fmt.Printf("\nreview · %s · %s\npar %d\n\n", strings.Join(ch.Skills, ", "), ch.Title, ch.Par)
		} else {
			fmt.Printf("\n%s\n%s\npar %d\n\n", ch.Title, ch.Brief, ch.Par)
			if ch.Teach != "" {
				fmt.Println(ch.Teach)
				fmt.Println()
			}
		}
		res, err := runner.Play(context.Background(), ch, a.Cfg.Nvim)
		if err != nil && res.Keys == "" && res.Buffer == "" {
			return err
		}
		out, aerr := progress.Apply(context.Background(), a.Store, a.Cat, ch, res)
		if aerr != nil {
			return aerr
		}
		printResult(ch, res, out)
		if out.Stars == 3 || !isTTY() {
			return nil
		}
		sensei, aiErr := ai.FromConfig(a.Cfg)
		for {
			prompt := "\n[s] show me   [r] retry   [enter] continue  "
			if aiErr == nil {
				prompt = "\n[s] show me   [r] retry   [?] debrief   [enter] continue  "
			}
			fmt.Print(prompt)
			line, _ := in.ReadString('\n')
			switch strings.TrimSpace(line) {
			case "?":
				if aiErr != nil {
					fmt.Println("configure an AI provider for debriefs: hjkl ai setup")
					continue
				}
				stars, _ := a.Store.Stars(context.Background())
				text, err := ai.Debrief(context.Background(), sensei, ch, res, progress.UnlockedSkills(a.Cat, stars))
				if err != nil {
					fmt.Println("debrief failed:", err)
					continue
				}
				fmt.Println("\n" + text)
				continue
			case "s":
				if err := runner.Demo(context.Background(), ch, a.Cfg.Nvim); err != nil {
					return err
				}
				continue
			case "r":
			default:
				return nil
			}
			break
		}
	}
}

func printResult(ch curriculum.Challenge, res runner.Result, out progress.Outcome) {
	status := "MISS"
	if res.OK {
		status = "CLEAR"
	}
	fmt.Printf("\n%s  %s  keys %d / par %d  %.1fs  xp +%d\n", status, starsText(out.Stars), res.KeyCount, ch.Par, float64(res.DurationMS)/1000, out.XP)
	if out.Fluent {
		fmt.Println("fluent - that one is in your fingers")
	} else if res.OK && out.Stars == 3 {
		fmt.Println("right keys - next time, faster: speed is how muscle memory shows")
	}
	if ch.Type == curriculum.TypeSpeedrun {
		fmt.Printf("clock %.1fs / target %.1fs (from your first key)\n", float64(res.DurationMS)/1000, float64(ch.TimeTargetMS)/1000)
	}
	fmt.Printf("you  %s\n", res.Keys)
	// On a miss the answer stays hidden: try again, or ask to be shown.
	if res.OK {
		fmt.Printf("par  %s\n", ch.Solution)
	}
	if out.HintsUsed > 0 {
		fmt.Printf("hints used: %d (-%d star)\n", out.HintsUsed, out.HintsUsed)
	}
	if out.Technique != "" {
		fmt.Println(out.Technique)
	}
	if out.DotScore != "" {
		fmt.Println(out.DotScore)
	}
	if p := curriculum.PrincipleText(ch.Principle); p != "" {
		fmt.Println(p)
	}
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
