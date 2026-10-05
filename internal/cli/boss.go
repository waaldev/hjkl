package cli

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/waaldev/hjkl/internal/ai"
	"github.com/waaldev/hjkl/internal/app"
	"github.com/waaldev/hjkl/internal/progress"
)

func newBossCmd() *cobra.Command {
	var lines string
	var yes bool
	cmd := &cobra.Command{
		Use:   "boss <file>",
		Short: "A boss fight built from a block of your own code (optional AI; asks before sending)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := args[0]
			if ai.IsSecretPath(path) {
				return failf("%s looks like it holds secrets; it is never sent", path)
			}
			raw, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			all := strings.Split(strings.ReplaceAll(string(raw), "\r\n", "\n"), "\n")
			lo, hi, err := bossRange(all, lines, rand.Intn)
			if err != nil {
				return err
			}
			code := strings.Join(all[lo-1:hi], "\n")
			return withApp(func(a *app.App) error {
				p, err := ai.FromConfig(a.Cfg)
				if err != nil {
					return err
				}
				// The one place code leaves the machine on request: say what.
				if !yes {
					if !isTTY() {
						return failf("pass --yes to send lines %d-%d of %s without a prompt", lo, hi, path)
					}
					fmt.Printf("Send lines %d-%d of %s to %s to build a boss fight? [y/N] ", lo, hi, filepath.Base(path), p.Name())
					if r := strings.ToLower(strings.TrimSpace(readLine())); r != "y" && r != "yes" {
						return nil
					}
				}
				stars, _ := a.Store.Stars(context.Background())
				fmt.Println("inventing a refactor and checking it in nvim…")
				ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
				defer cancel()
				req := ai.BossRequest{Code: code, Language: languageFor(path), Unlocked: progress.UnlockedSkills(a.Cat, stars)}
				ch, err := ai.GenerateBoss(ctx, p, req, ai.NvimVerifier(a.Cfg.Nvim))
				if err != nil {
					return err
				}
				if _, err := savePersonal(ch); err == nil {
					a.Cat.AddPersonal(ch)
					fmt.Println("checked in nvim and saved - it will come back in your reviews")
				}
				return playOne(a, ch)
			})
		},
	}
	cmd.Flags().StringVar(&lines, "lines", "", "line range to use, e.g. 12-30 (default: a block chosen for you)")
	cmd.Flags().BoolVar(&yes, "yes", false, "send without asking")
	return cmd
}

// bossRange picks the lines for a boss: the --lines range if given,
// otherwise a random paragraph-like block (lines between blank lines) of a
// workable size, or the start of the file.
func bossRange(lines []string, spec string, intn func(int) int) (lo, hi int, err error) {
	if spec != "" {
		a, b, ok := strings.Cut(spec, "-")
		lo, err1 := strconv.Atoi(strings.TrimSpace(a))
		hi, err2 := strconv.Atoi(strings.TrimSpace(b))
		switch {
		case !ok || err1 != nil || err2 != nil || lo < 1 || hi < lo || hi > len(lines):
			return 0, 0, fmt.Errorf("--lines %q: want a range like 12-30 within the file's %d lines", spec, len(lines))
		case hi-lo+1 > ai.MaxBossLines:
			return 0, 0, fmt.Errorf("--lines %q is %d lines; at most %d", spec, hi-lo+1, ai.MaxBossLines)
		}
		return lo, hi, nil
	}
	type block struct{ lo, hi int }
	var blocks []block
	start := 0
	for i := 0; i <= len(lines); i++ {
		if i == len(lines) || strings.TrimSpace(lines[i]) == "" {
			if n := i - start; n >= 4 && n <= ai.MaxBossLines {
				blocks = append(blocks, block{start + 1, i})
			}
			start = i + 1
		}
	}
	if len(blocks) > 0 {
		b := blocks[intn(len(blocks))]
		return b.lo, b.hi, nil
	}
	hi = len(lines)
	for hi > 0 && strings.TrimSpace(lines[hi-1]) == "" {
		hi--
	}
	if hi > ai.MaxBossLines {
		hi = ai.MaxBossLines
	}
	if hi == 0 {
		return 0, 0, fmt.Errorf("the file is empty")
	}
	return 1, hi, nil
}

// languageFor maps a file extension to an nvim filetype for highlighting.
func languageFor(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".go":
		return "go"
	case ".py":
		return "python"
	case ".js", ".mjs", ".cjs":
		return "javascript"
	case ".ts":
		return "typescript"
	case ".tsx":
		return "typescriptreact"
	case ".rs":
		return "rust"
	case ".lua":
		return "lua"
	case ".md":
		return "markdown"
	case ".rb":
		return "ruby"
	case ".java":
		return "java"
	case ".c", ".h":
		return "c"
	case ".sh":
		return "sh"
	case ".yaml", ".yml":
		return "yaml"
	case ".json":
		return "json"
	}
	return ""
}
