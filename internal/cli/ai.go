package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/waaldev/hjkl/internal/ai"
	"github.com/waaldev/hjkl/internal/app"
	"github.com/waaldev/hjkl/internal/config"
	"github.com/waaldev/hjkl/internal/progress"
)

func newAICmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ai",
		Short: "Optional sensei - setup, test, and list providers",
	}
	cmd.AddCommand(newAISetupCmd(), newAITestCmd(), newAIListCmd())
	return cmd
}

func newAIListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "Show configured and available providers",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			fmt.Printf("resolved provider: %s\n", empty(cfg.ResolvedProvider(), "(none)"))
			fmt.Printf("  anthropic   env %s  set=%v\n", envName(cfg.AI.Anthropic.APIKeyEnv, "ANTHROPIC_API_KEY"), os.Getenv(envName(cfg.AI.Anthropic.APIKeyEnv, "ANTHROPIC_API_KEY")) != "")
			fmt.Printf("  openai      env %s  base %s\n", envName(cfg.AI.OpenAI.APIKeyEnv, "OPENAI_API_KEY"), empty(cfg.AI.OpenAI.BaseURL, "https://api.openai.com/v1"))
			fmt.Printf("  cli-agent   %s %s\n", empty(cfg.AI.CLIAgent.Command, "(unset)"), strings.Join(cfg.AI.CLIAgent.Args, " "))
			fmt.Println("\nAI is optional. The core game is offline. The sensei never types for you.")
			return nil
		},
	}
}

func newAITestCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "test",
		Short: "Send a ping through the configured provider",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			p, err := ai.FromConfig(cfg)
			if err != nil {
				return err
			}
			fmt.Printf("provider %s …\n", p.Name())
			if err := ai.Ping(context.Background(), p); err != nil {
				return err
			}
			fmt.Println("ok")
			return nil
		},
	}
}

func newAISetupCmd() *cobra.Command {
	var provider, model, base, command string
	cmd := &cobra.Command{
		Use:   "setup",
		Short: "Write provider settings to config.toml",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			if provider == "" {
				fmt.Print("provider [anthropic|openai|cli-agent]: ")
				provider = readLine()
			}
			cfg.AI.Provider = strings.TrimSpace(provider)
			if model != "" {
				cfg.AI.Model = model
			}
			if base != "" {
				cfg.AI.OpenAI.BaseURL = base
			}
			if command != "" {
				cfg.AI.CLIAgent.Command = command
			}
			if err := config.Save(cfg); err != nil {
				return err
			}
			p, _ := config.Path()
			fmt.Println("wrote", p)
			return nil
		},
	}
	cmd.Flags().StringVar(&provider, "provider", "", "anthropic, openai, or cli-agent")
	cmd.Flags().StringVar(&model, "model", "", "model id")
	cmd.Flags().StringVar(&base, "base-url", "", "OpenAI-compatible base URL (Ollama: http://localhost:11434/v1)")
	cmd.Flags().StringVar(&command, "command", "", "cli-agent executable (opencode, claude, …)")
	return cmd
}

func newAskCmd() *cobra.Command {
	var noDrill bool
	cmd := &cobra.Command{
		Use:   "ask [question]",
		Short: "Ask the sensei (optional AI), then practice the answer. Never types for you.",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return withApp(func(a *app.App) error {
				p, err := ai.FromConfig(a.Cfg)
				if err != nil {
					return err
				}
				question := strings.Join(args, " ")
				stars, _ := a.Store.Stars(context.Background())
				unlocked := progress.UnlockedSkills(a.Cat, stars)
				answer, err := ai.Ask(context.Background(), p, question, unlocked)
				if err != nil {
					return err
				}
				fmt.Println(answer)
				// An answer you read is forgotten; one you type sticks.
				if noDrill || !isTTY() {
					return nil
				}
				fmt.Print("\nPractice it now in a short drill? [Y/n] ")
				if r := strings.ToLower(strings.TrimSpace(readLine())); r != "" && r != "y" && r != "yes" {
					return nil
				}
				return generateAndPlay(a, p, ai.DrillRequest{Question: question, Answer: answer, Unlocked: unlocked})
			})
		},
	}
	cmd.Flags().BoolVar(&noDrill, "no-drill", false, "just answer, no practice drill")
	return cmd
}

func runAIDrill(a *app.App, skill string) error {
	p, err := ai.FromConfig(a.Cfg)
	if err != nil {
		return err
	}
	if skill == "" {
		skill = "operators"
	}
	stars, _ := a.Store.Stars(context.Background())
	return generateAndPlay(a, p, ai.DrillRequest{Skill: skill, Unlocked: progress.UnlockedSkills(a.Cat, stars)})
}

// generateAndPlay writes a drill with the AI, keeps it only if nvim
// confirms it, saves it to the personal drills (so reviews can reuse it,
// offline), and plays it.
func generateAndPlay(a *app.App, p ai.Provider, req ai.DrillRequest) error {
	if req.Language == "" {
		req.Language = "go"
	}
	fmt.Println("writing a drill and checking it in nvim…")
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	ch, err := ai.GenerateDrill(ctx, p, req, ai.NvimVerifier(a.Cfg.Nvim))
	if err != nil {
		return err
	}
	if _, err := savePersonal(ch); err != nil {
		fmt.Println("(could not save the drill for reviews:", err, ")")
	} else {
		a.Cat.AddPersonal(ch)
		fmt.Println("checked in nvim and saved - it will come back in your reviews")
	}
	return playOne(a, ch)
}

func envName(v, fallback string) string {
	if v == "" {
		return fallback
	}
	return v
}

func empty(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return v
}

func readLine() string {
	sc := bufio.NewScanner(os.Stdin)
	if sc.Scan() {
		return sc.Text()
	}
	return ""
}
