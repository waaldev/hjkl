package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

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
	return &cobra.Command{
		Use:   "ask [question]",
		Short: "Ask the sensei (optional AI). Never types for you.",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return withApp(func(a *app.App) error {
				p, err := ai.FromConfig(a.Cfg)
				if err != nil {
					return err
				}
				stars, _ := a.Store.Stars(context.Background())
				text, err := ai.Ask(context.Background(), p, strings.Join(args, " "), progress.UnlockedSkills(a.Cat, stars))
				if err != nil {
					return err
				}
				fmt.Println(text)
				fmt.Println("\n(You press the keys. Try hjkl drill --ai after this.)")
				return nil
			})
		},
	}
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
	ch, err := ai.GenerateChallenge(context.Background(), p, skill, "go", progress.UnlockedSkills(a.Cat, stars))
	if err != nil {
		return err
	}
	if err := ai.VerifyGate(context.Background(), ch, a.Cfg.Nvim); err != nil {
		return err
	}
	fmt.Println("generated drill passed the verify gate")
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
