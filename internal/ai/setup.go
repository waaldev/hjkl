package ai

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/waaldev/hjkl/internal/config"
)

func FromConfig(cfg config.Config) (Provider, error) {
	name := cfg.ResolvedProvider()
	if name == "" {
		return nil, fmt.Errorf("no AI provider configured (set ANTHROPIC_API_KEY or run hjkl ai setup)")
	}
	switch name {
	case "anthropic":
		env := cfg.AI.Anthropic.APIKeyEnv
		if env == "" {
			env = "ANTHROPIC_API_KEY"
		}
		return Anthropic{APIKey: os.Getenv(env), Model: first(cfg.AI.Model, DefaultAnthropicModel), Effort: cfg.AI.Anthropic.Effort}, nil
	case "openai":
		env := cfg.AI.OpenAI.APIKeyEnv
		if env == "" {
			env = "OPENAI_API_KEY"
		}
		model := cfg.AI.OpenAI.Model
		if model == "" {
			model = first(cfg.AI.Model, "gpt-4o")
		}
		return OpenAI{APIKey: os.Getenv(env), Model: model, BaseURL: cfg.AI.OpenAI.BaseURL}, nil
	case "cli-agent":
		return CLIAgent{Command: cfg.AI.CLIAgent.Command, Args: cfg.AI.CLIAgent.Args}, nil
	default:
		return nil, fmt.Errorf("unknown provider %q", name)
	}
}

func first(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return v
}

func Ping(ctx context.Context, p Provider) error {
	_, err := p.Complete(ctx, Request{Prompt: "Reply with the single word pong."})
	return err
}
