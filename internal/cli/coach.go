package cli

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/waaldev/hjkl/embeds"
	"github.com/waaldev/hjkl/internal/ai"
	"github.com/waaldev/hjkl/internal/app"
	"github.com/waaldev/hjkl/internal/config"
	"github.com/waaldev/hjkl/internal/progress"
)

func newCoachCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "coach",
		Short: "Install the Neovim coach, read its report, or see today's quest",
	}
	cmd.AddCommand(newCoachInstallCmd(), newCoachReportCmd(), newCoachQuestCmd(), newCoachReviewCmd())
	return cmd
}

func newCoachInstallCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "install",
		Short: "Copy the coach plugin and print a lazy.nvim snippet",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, data, err := config.Dirs()
			if err != nil {
				return err
			}
			dest := filepath.Join(data, "nvim")
			if err := os.MkdirAll(dest, 0o755); err != nil {
				return err
			}
			err = fs.WalkDir(embeds.FS, "nvim", func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}
				rel, _ := filepath.Rel("nvim", path)
				target := filepath.Join(dest, rel)
				if d.IsDir() {
					return os.MkdirAll(target, 0o755)
				}
				b, err := embeds.FS.ReadFile(path)
				if err != nil {
					return err
				}
				if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
					return err
				}
				return os.WriteFile(target, b, 0o644)
			})
			if err != nil {
				return err
			}
			events, _ := config.CoachEventsPath()
			fmt.Printf("coach plugin copied to %s\n\n", dest)
			fmt.Println("lazy.nvim snippet:")
			fmt.Printf(`
{
  dir = %q,
  name = "hjkl",
  lazy = false,
  opts = {
    events_path = %q,
    quiet = false,
    hint_throttle_ms = 8000,
    hint_limit = 5,
  },
  config = function(_, opts)
    require("hjkl.coach").setup(opts)
  end,
},
`, dest, events)
			fmt.Println("\nThen restart nvim. mash jjjjjjj in a real file - a hint should appear.")
			fmt.Println(":HjklDrill practices the last flagged habit; :HjklSnooze mutes it for a week.")
			return nil
		},
	}
}

func newCoachReportCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "report",
		Short: "Show aggregated anti-patterns (no file contents)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withApp(func(a *app.App) error {
				ctx := context.Background()
				sum, err := a.Store.CoachSummary(ctx, time.Now().Add(-30*24*time.Hour))
				if err != nil {
					return err
				}
				if used, _ := a.Store.CoachUsage(ctx, time.Now().Add(-30*24*time.Hour)); len(sum) == 0 && len(used) == 0 {
					fmt.Println("No coach events yet. hjkl coach install, then edit as usual.")
					return nil
				}
				fmt.Println("last 30 days (counts, never file contents)")
				for pat, n := range sum {
					fmt.Printf("  %-20s  %d\n", pat, n)
				}
				if used, _ := a.Store.CoachUsage(ctx, time.Now().Add(-30*24*time.Hour)); len(used) > 0 {
					fmt.Println("\nused in real work")
					for k, n := range used {
						fmt.Printf("  %-20s  %d\n", k, n)
					}
				}
				return nil
			})
		},
	}
}

func newCoachQuestCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "quest",
		Short: "Today's quest from the coach and SRS",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withApp(func(a *app.App) error {
				ctx := context.Background()
				sum, _ := a.Store.CoachSummary(ctx, time.Now().Add(-24*time.Hour))
				due, _ := a.Store.DueSkills(ctx, time.Now(), 3)
				fmt.Println("today's quest")
				if n := sum["repeat-j"]; n >= 5 {
					fmt.Printf("  • the coach saw %d j-streaks yesterday - use 5j or /search today\n", n)
				}
				if n := sum["visual-iw-delete"]; n > 0 {
					fmt.Println("  • swap one viwd for diw so . can replay it")
				}
				if n := sum["repeat-x"]; n >= 3 {
					fmt.Println("  • try dw / diw instead of xxxx")
				}
				if len(due) > 0 {
					var skills []string
					for _, d := range due {
						skills = append(skills, d.Skill)
					}
					fmt.Printf("  • review due skills: %s  (hjkl daily)\n", strings.Join(skills, ", "))
				}
				stars, _ := a.Store.Stars(ctx)
				if ch, ok := progress.NextChallenge(a.Cat, stars); ok {
					fmt.Printf("  • next dojo drill: %s (%s)\n", ch.Title, ch.Belt)
				}
				if sum["repeat-j"] == 0 && sum["visual-iw-delete"] == 0 && sum["repeat-x"] == 0 && len(due) == 0 {
					fmt.Println("  • light up two new Grammar Grid cells (hjkl stats)")
				}
				return nil
			})
		},
	}
}

func newCoachReviewCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "review",
		Short: "Weekly coach review (AI if configured; otherwise the numbers)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withApp(func(a *app.App) error {
				ctx := context.Background()
				sum, err := a.Store.CoachSummary(ctx, time.Now().Add(-7*24*time.Hour))
				if err != nil {
					return err
				}
				if len(sum) == 0 {
					fmt.Println("No coach events this week.")
					return nil
				}
				var b strings.Builder
				b.WriteString("counts per anti-pattern this week (no file contents):\n")
				for pat, n := range sum {
					fmt.Fprintf(&b, "- %s: %d\n", pat, n)
				}
				p, err := ai.FromConfig(a.Cfg)
				if err != nil {
					fmt.Print(b.String())
					fmt.Println("(configure an AI provider for a written review)")
					return nil
				}
				stars, _ := a.Store.Stars(ctx)
				text, err := ai.Ask(ctx, p, "Write a short weekly Vim coaching review from these aggregated stats. No new keys beyond unlocked skills. Stats:\n"+b.String(), progress.UnlockedSkills(a.Cat, stars))
				if err != nil {
					return err
				}
				fmt.Println(text)
				return nil
			})
		},
	}
}
