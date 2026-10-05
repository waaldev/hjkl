package runner

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/waaldev/hjkl/embeds"
	"github.com/waaldev/hjkl/internal/curriculum"
)

const harnessName = "harness.lua"

// Result is what harness.lua writes after a run.
type Result struct {
	OK         bool   `json:"ok"`
	Keys       string `json:"keys"`
	KeyCount   int    `json:"key_count"`
	CmdKeys    string `json:"cmd_keys"` // keys typed in Normal/operator-pending/Visual mode
	Par        int    `json:"par"`
	DurationMS int    `json:"duration_ms"`
	Buffer     string `json:"buffer"`
	Cursor     []int  `json:"cursor"`
	Reason     string `json:"reason"`
	Aborted    bool   `json:"aborted"`
}

// Options control how Neovim is launched.
type Options struct {
	Nvim     string
	Mode     string // play | verify
	Timeout  time.Duration
	Headless bool
}

func (o Options) withDefaults() Options {
	if o.Nvim == "" {
		o.Nvim = "nvim"
	}
	if o.Mode == "" {
		o.Mode = "play"
	}
	if o.Timeout == 0 {
		if o.Mode == "verify" {
			o.Timeout = 15 * time.Second
		} else {
			o.Timeout = 0
		}
	}
	if o.Mode == "verify" {
		o.Headless = true
	}
	return o
}

// Play launches an interactive Neovim session for the challenge.
func Play(ctx context.Context, ch curriculum.Challenge, nvim string) (Result, error) {
	return Run(ctx, ch, Options{Nvim: nvim, Mode: "play"})
}

// Verify feeds the canonical solution headlessly and checks the target.
func Verify(ctx context.Context, ch curriculum.Challenge, nvim string) (Result, error) {
	return Run(ctx, ch, Options{Nvim: nvim, Mode: "verify", Headless: true, Timeout: 15 * time.Second})
}

// Run writes challenge JSON, extracts the harness, and execs nvim.
func Run(ctx context.Context, ch curriculum.Challenge, opt Options) (Result, error) {
	opt = opt.withDefaults()
	dir, err := os.MkdirTemp("", "hjkl-*")
	if err != nil {
		return Result{}, err
	}
	defer os.RemoveAll(dir)

	challengePath := filepath.Join(dir, "challenge.json")
	resultPath := filepath.Join(dir, "result.json")
	harnessPath := filepath.Join(dir, harnessName)

	raw, err := json.Marshal(ch)
	if err != nil {
		return Result{}, err
	}
	if err := os.WriteFile(challengePath, raw, 0o600); err != nil {
		return Result{}, err
	}
	script, err := embeds.FS.ReadFile("nvim/" + harnessName)
	if err != nil {
		return Result{}, fmt.Errorf("harness: %w", err)
	}
	if err := os.WriteFile(harnessPath, script, 0o600); err != nil {
		return Result{}, err
	}

	args := []string{"--clean", "-n", "-i", "NONE"}
	if opt.Headless {
		args = append(args, "--headless")
	}
	args = append(args, "-c", "luafile "+harnessPath)

	if opt.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, opt.Timeout)
		defer cancel()
	}

	cmd := exec.CommandContext(ctx, opt.Nvim, args...)
	cmd.Env = append(os.Environ(),
		"HJKL_CHALLENGE_PATH="+challengePath,
		"HJKL_RESULT_PATH="+resultPath,
		"HJKL_MODE="+opt.Mode,
	)
	var stderr bytes.Buffer
	if opt.Mode == "play" {
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	} else {
		cmd.Stderr = &stderr
	}

	runErr := cmd.Run()
	data, readErr := os.ReadFile(resultPath)
	if readErr != nil {
		if runErr != nil {
			return Result{}, fmt.Errorf("nvim: %w\n%s", runErr, stderr.String())
		}
		return Result{}, fmt.Errorf("no result.json: %w", readErr)
	}
	var res Result
	if err := json.Unmarshal(data, &res); err != nil {
		return Result{}, fmt.Errorf("result.json: %w", err)
	}
	if !res.OK && opt.Mode == "verify" {
		return res, fmt.Errorf("verify failed (%s): got %q", res.Reason, res.Buffer)
	}
	return res, nil
}

// Command returns the nvim *exec.Cmd for an interactive play session without
// waiting. Used by the TUI via tea.ExecProcess.
func Command(ch curriculum.Challenge, nvim string, dir string) (*exec.Cmd, string, error) {
	if nvim == "" {
		nvim = "nvim"
	}
	challengePath := filepath.Join(dir, "challenge.json")
	resultPath := filepath.Join(dir, "result.json")
	harnessPath := filepath.Join(dir, harnessName)

	raw, err := json.Marshal(ch)
	if err != nil {
		return nil, "", err
	}
	if err := os.WriteFile(challengePath, raw, 0o600); err != nil {
		return nil, "", err
	}
	script, err := embeds.FS.ReadFile("nvim/" + harnessName)
	if err != nil {
		return nil, "", err
	}
	if err := os.WriteFile(harnessPath, script, 0o600); err != nil {
		return nil, "", err
	}

	cmd := exec.Command(nvim, "--clean", "-n", "-i", "NONE", "-c", "luafile "+harnessPath)
	cmd.Env = append(os.Environ(),
		"HJKL_CHALLENGE_PATH="+challengePath,
		"HJKL_RESULT_PATH="+resultPath,
		"HJKL_MODE=play",
	)
	return cmd, resultPath, nil
}

func ReadResult(path string) (Result, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Result{}, err
	}
	var res Result
	if err := json.Unmarshal(data, &res); err != nil {
		return Result{}, err
	}
	return res, nil
}
