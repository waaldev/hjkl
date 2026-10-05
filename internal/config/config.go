package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

type Config struct {
	Nvim  string      `toml:"nvim"`
	AI    AIConfig    `toml:"ai"`
	Coach CoachConfig `toml:"coach"`
}

type AIConfig struct {
	Provider  string         `toml:"provider"`
	Model     string         `toml:"model"`
	Anthropic ProviderKeys   `toml:"anthropic"`
	OpenAI    OpenAIConfig   `toml:"openai"`
	CLIAgent  CLIAgentConfig `toml:"cli_agent"`
}

type ProviderKeys struct {
	APIKeyEnv string `toml:"api_key_env"`
	// Effort is sent as output_config.effort (low|medium|high|xhigh|max).
	// Empty uses "low" for the default model and nothing for others.
	Effort string `toml:"effort"`
}

type OpenAIConfig struct {
	BaseURL   string `toml:"base_url"`
	APIKeyEnv string `toml:"api_key_env"`
	Model     string `toml:"model"`
}

type CLIAgentConfig struct {
	Command string   `toml:"command"`
	Args    []string `toml:"args"`
}

type CoachConfig struct {
	Quiet          bool `toml:"quiet"`
	HintThrottleMS int  `toml:"hint_throttle_ms"`
}

func Defaults() Config {
	return Config{
		Nvim: "nvim",
		AI: AIConfig{
			Anthropic: ProviderKeys{APIKeyEnv: "ANTHROPIC_API_KEY"},
			OpenAI:    OpenAIConfig{APIKeyEnv: "OPENAI_API_KEY", BaseURL: "https://api.openai.com/v1"},
		},
		Coach: CoachConfig{HintThrottleMS: 8000},
	}
}

func Dirs() (configDir, dataDir string, err error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", "", err
	}
	if p := os.Getenv("HJKL_CONFIG_DIR"); p != "" {
		configDir = p
	} else if x := os.Getenv("XDG_CONFIG_HOME"); x != "" {
		configDir = filepath.Join(x, "hjkl")
	} else {
		configDir = filepath.Join(home, ".config", "hjkl")
	}
	if p := os.Getenv("HJKL_DATA_DIR"); p != "" {
		dataDir = p
	} else if x := os.Getenv("XDG_DATA_HOME"); x != "" {
		dataDir = filepath.Join(x, "hjkl")
	} else {
		dataDir = filepath.Join(home, ".local", "share", "hjkl")
	}
	return configDir, dataDir, nil
}

func Path() (string, error) {
	c, _, err := Dirs()
	if err != nil {
		return "", err
	}
	return filepath.Join(c, "config.toml"), nil
}

func DBPath() (string, error) {
	_, d, err := Dirs()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "hjkl.db"), nil
}

func CoachEventsPath() (string, error) {
	_, d, err := Dirs()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "coach-events.jsonl"), nil
}

func Load() (Config, error) {
	cfg := Defaults()
	p, err := Path()
	if err != nil {
		return cfg, err
	}
	raw, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, err
	}
	if err := toml.Unmarshal(raw, &cfg); err != nil {
		return cfg, fmt.Errorf("config: %w", err)
	}
	if cfg.Nvim == "" {
		cfg.Nvim = "nvim"
	}
	if cfg.Coach.HintThrottleMS == 0 {
		cfg.Coach.HintThrottleMS = 8000
	}
	return cfg, nil
}

func Save(cfg Config) error {
	p, err := Path()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	raw, err := toml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(p, raw, 0o600)
}

func (c Config) AIEnabled() bool {
	if strings.TrimSpace(c.AI.Provider) != "" {
		return true
	}
	if os.Getenv("ANTHROPIC_API_KEY") != "" {
		return true
	}
	if os.Getenv("OPENAI_API_KEY") != "" {
		return true
	}
	return false
}

func (c Config) ResolvedProvider() string {
	if p := strings.TrimSpace(c.AI.Provider); p != "" {
		return p
	}
	if os.Getenv("ANTHROPIC_API_KEY") != "" {
		return "anthropic"
	}
	if os.Getenv("OPENAI_API_KEY") != "" {
		return "openai"
	}
	if c.AI.CLIAgent.Command != "" {
		return "cli-agent"
	}
	return ""
}
