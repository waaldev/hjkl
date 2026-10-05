package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/waaldev/hjkl/internal/ai"
	"github.com/waaldev/hjkl/internal/app"
	"github.com/waaldev/hjkl/internal/config"
	"github.com/waaldev/hjkl/internal/curriculum"
	"github.com/waaldev/hjkl/internal/keys"
	"github.com/waaldev/hjkl/internal/progress"
	"github.com/waaldev/hjkl/internal/runner"
)

// suggestResult is the JSON the coach plugin reads.
type suggestResult struct {
	OK           bool   `json:"ok"`
	Error        string `json:"error,omitempty"`
	Keys         string `json:"keys,omitempty"`
	KeyCount     int    `json:"key_count,omitempty"`
	YourKeyCount int    `json:"your_key_count,omitempty"`
	Explanation  string `json:"explanation,omitempty"`
	Principle    string `json:"principle,omitempty"`
	Drill        string `json:"drill,omitempty"` // saved challenge, for practice / save
}

const suggestDisabled = "suggestions are off: set suggest = true under [ai] in ~/.config/hjkl/config.toml (only the lines of the edit are sent)"

func newSuggestCmd() *cobra.Command {
	var input, practice, save string
	var status bool
	cmd := &cobra.Command{
		Use:   "suggest",
		Short: "Find a verified shorter way to make a real edit (used by :HjklWhy)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return withApp(func(a *app.App) error {
				switch {
				case status:
					name := a.Cfg.ResolvedProvider()
					return printJSON(map[string]any{"enabled": a.Cfg.AI.Suggest && name != "", "provider": name})
				case practice != "":
					ch, err := readChallenge(practice)
					if err != nil {
						return err
					}
					return play(a, ch, false)
				case save != "":
					return printJSON(saveDrill(save))
				case input != "":
					return printJSON(suggest(a, input))
				}
				return cmd.Help()
			})
		},
	}
	cmd.Flags().StringVar(&input, "input", "", "edit JSON written by the coach plugin")
	cmd.Flags().BoolVar(&status, "status", false, "print whether suggestions are enabled")
	cmd.Flags().StringVar(&practice, "practice", "", "play a suggested drill")
	cmd.Flags().StringVar(&save, "save", "", "save a suggested drill for spaced repetition")
	return cmd
}

func suggest(a *app.App, inputPath string) suggestResult {
	if !a.Cfg.AI.Suggest {
		return suggestResult{Error: suggestDisabled}
	}
	raw, err := os.ReadFile(inputPath)
	if err != nil {
		return suggestResult{Error: err.Error()}
	}
	var in ai.EditInput
	if err := json.Unmarshal(raw, &in); err != nil {
		return suggestResult{Error: "bad input: " + err.Error()}
	}
	p, err := ai.FromConfig(a.Cfg)
	if err != nil {
		return suggestResult{Error: err.Error()}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	stars, _ := a.Store.Stars(ctx)
	verify := func(ctx context.Context, ch curriculum.Challenge) error {
		_, err := runner.Verify(ctx, ch, a.Cfg.Nvim)
		return err
	}
	s, ch, err := ai.SuggestEdit(ctx, p, in, progress.UnlockedSkills(a.Cat, stars), verify)
	if err != nil {
		return suggestResult{Error: err.Error()}
	}
	path, err := writeSuggestion(ch)
	if err != nil {
		return suggestResult{Error: err.Error()}
	}
	return suggestResult{
		OK: true, Keys: s.Keys, KeyCount: keys.Count(s.Keys), YourKeyCount: keys.Count(in.Keys),
		Explanation: s.Explanation, Principle: s.Principle, Drill: path,
	}
}

func writeSuggestion(ch curriculum.Challenge) (string, error) {
	dir, err := config.SuggestionsDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	path := filepath.Join(dir, ch.ID+".json")
	raw, err := json.MarshalIndent(ch, "", "  ")
	if err != nil {
		return "", err
	}
	return path, os.WriteFile(path, raw, 0o600)
}

// saveDrill copies a suggested drill into the personal drills, where
// reviews for its skills can pick it.
func saveDrill(path string) map[string]any {
	ch, err := readChallenge(path)
	if err != nil {
		return map[string]any{"ok": false, "error": err.Error()}
	}
	dest, err := savePersonal(ch)
	if err != nil {
		return map[string]any{"ok": false, "error": err.Error()}
	}
	return map[string]any{"ok": true, "path": dest, "skills": ch.Skills}
}

// savePersonal stores a verified drill in the personal drills folder.
// Reviews for its skills can pick it from then on, offline.
func savePersonal(ch curriculum.Challenge) (string, error) {
	dir, err := config.DrillsDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	ch.Belt = curriculum.PersonalBelt
	dest := filepath.Join(dir, ch.ID+".json")
	raw, err := json.MarshalIndent(ch, "", "  ")
	if err != nil {
		return "", err
	}
	return dest, os.WriteFile(dest, raw, 0o600)
}

func readChallenge(path string) (curriculum.Challenge, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return curriculum.Challenge{}, err
	}
	var ch curriculum.Challenge
	if err := json.Unmarshal(raw, &ch); err != nil {
		return curriculum.Challenge{}, fmt.Errorf("%s: %w", path, err)
	}
	return ch, nil
}

func printJSON(v any) error {
	return json.NewEncoder(os.Stdout).Encode(v)
}
