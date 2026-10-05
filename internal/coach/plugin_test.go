package coach

import (
	"bufio"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// The Lua plugin is the only detector. This drives it in a headless nvim
// with typed keys and checks the events and hints it produces.
func runPlugin(t *testing.T, dir, keys string) (events []fileEvent, hints []string) {
	t.Helper()
	if _, err := exec.LookPath("nvim"); err != nil {
		t.Skip("nvim not installed")
	}
	root, err := filepath.Abs("../../embeds/nvim")
	if err != nil {
		t.Fatal(err)
	}
	evPath := filepath.Join(dir, "ev.jsonl")
	hintPath := filepath.Join(dir, "hints.txt")
	script := filepath.Join(dir, "t.lua")
	lua := `
vim.opt.rtp:prepend(` + luaStr(root) + `)
local hints = io.open(` + luaStr(hintPath) + `, "w")
vim.notify = function(msg) hints:write(msg .. "\n") end
local coach = require("hjkl.coach")
coach.setup({ events_path = ` + luaStr(evPath) + `, hint_throttle_ms = 0 })
vim.api.nvim_buf_set_lines(0, 0, -1, false, vim.fn["repeat"]({ 'foo "bar" baz qux' }, 40))
vim.api.nvim_feedkeys(vim.api.nvim_replace_termcodes(` + luaStr(keys) + `, true, false, true), "tx", false)
coach.flush()
hints:close()
vim.cmd("qa!")
`
	if err := os.WriteFile(script, []byte(lua), 0o600); err != nil {
		t.Fatal(err)
	}
	// The trailing +qa! makes nvim exit even if the script errors.
	out, err := exec.Command("nvim", "--clean", "--headless", "-c", "luafile "+script, "+qa!").CombinedOutput()
	if err != nil {
		t.Fatalf("nvim: %v\n%s", err, out)
	}
	if f, err := os.Open(evPath); err == nil {
		sc := bufio.NewScanner(f)
		for sc.Scan() {
			var ev fileEvent
			if err := json.Unmarshal(sc.Bytes(), &ev); err != nil {
				t.Fatal(err)
			}
			events = append(events, ev)
		}
		f.Close()
	}
	if b, err := os.ReadFile(hintPath); err == nil && len(b) > 0 {
		hints = strings.Split(strings.TrimSpace(string(b)), "\n")
	}
	return events, hints
}

// luaStr quotes s as a Lua long string, which needs no escaping.
func luaStr(s string) string {
	return "[==[" + s + "]==]"
}

func count(events []fileEvent, pattern, keys string) int {
	n := 0
	for _, e := range events {
		if e.Pattern == pattern && (keys == "" || e.Keys == keys) {
			n += e.Count
		}
	}
	return n
}

func TestPluginEvents(t *testing.T) {
	events, _ := runPlugin(t, t.TempDir(), `jjjjjjjjjjgg0viwdj0ciwnew<Esc>j.jd2wjfqxjddjci"z<Esc>`)
	if n := len(filterPattern(events, "repeat-j")); n != 1 || count(events, "repeat-j", "") != 10 {
		t.Fatalf("a 10-j run is one event of 10, got %d events: %+v", n, events)
	}
	if count(events, "visual-iw-delete", "") != 1 {
		t.Fatalf("viwd not detected: %+v", events)
	}
	for _, keys := range []string{"ciw", "d2w", "dd", ".", `ci"`} {
		if count(events, "used", keys) != 1 {
			t.Errorf("used %s not recorded: %+v", keys, events)
		}
	}
	if count(events, "used", "x") != 0 {
		t.Error("the x in fqx is an argument, not a command")
	}
}

func TestPluginHintsFade(t *testing.T) {
	dir := t.TempDir()
	run := strings.Repeat("jjjjjk", 8) // eight separate 5-j runs
	_, hints := runPlugin(t, dir, run)
	if len(hints) != 5 {
		t.Fatalf("want hint_limit (5) hints, got %d: %q", len(hints), hints)
	}
	if !strings.Contains(hints[4], ":HjklDrill") {
		t.Fatalf("the last reminder should point at :HjklDrill: %q", hints[4])
	}
	// The count persists: a new session stays quiet.
	if _, again := runPlugin(t, dir, run); len(again) != 0 {
		t.Fatalf("faded hints came back: %q", again)
	}
}

func filterPattern(events []fileEvent, pattern string) []fileEvent {
	var out []fileEvent
	for _, e := range events {
		if e.Pattern == pattern {
			out = append(out, e)
		}
	}
	return out
}

func TestPluginCelebratesFirstRealUse(t *testing.T) {
	// Day one only records what you already know.
	dir := t.TempDir()
	if _, hints := runPlugin(t, dir, "ciwx<Esc>"); len(hints) != 0 {
		t.Fatalf("no celebrations on the first day: %q", hints)
	}
	// After the baseline, a command never seen before gets one message.
	old := filepath.Join(dir, "coach-state.json")
	if err := os.WriteFile(old, []byte(`{"since":1,"hints":{},"snoozed":{},"seen":{"ciw":true}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	_, hints := runPlugin(t, dir, "ciwx<Esc>jdaw")
	if len(hints) != 1 || !strings.Contains(hints[0], "first real daw") {
		t.Fatalf("want one celebration for daw, got %q", hints)
	}
	if _, again := runPlugin(t, dir, "jdaw"); len(again) != 0 {
		t.Fatalf("celebrate once, got %q", again)
	}
}
