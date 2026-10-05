package coach

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// runWhy edits a named file in a headless nvim with the coach loaded and a
// stub `hjkl` binary, then runs :HjklWhy. It returns what the stub was sent,
// the coach's hints, and the float's text.
func runWhy(t *testing.T, fileName, keys string) (sent map[string]any, hints []string, float string) {
	t.Helper()
	if _, err := exec.LookPath("nvim"); err != nil {
		t.Skip("nvim not installed")
	}
	dir := t.TempDir()
	root, _ := filepath.Abs("../../embeds/nvim")
	stub := filepath.Join(dir, "hjkl")
	sentPath := filepath.Join(dir, "sent.json")
	script := `#!/bin/sh
case "$2" in
  --status) echo '{"enabled":true,"provider":"stub"}' ;;
  --input) cp "$3" ` + sentPath + `; echo '{"ok":true,"keys":"ciwbar<Esc>j.j.","key_count":11,"your_key_count":30,"explanation":"ciw then dot.","drill":"/tmp/x.json"}' ;;
esac
`
	if err := os.WriteFile(stub, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "out.txt")
	lua := `
vim.opt.rtp:prepend(` + luaStr(root) + `)
local log = io.open(` + luaStr(out) + `, "w")
vim.notify = function(msg) log:write("HINT " .. msg:gsub("\n", " ") .. "\n") end
require("hjkl.coach").setup({
  events_path = ` + luaStr(filepath.Join(dir, "ev.jsonl")) + `,
  hint_throttle_ms = 0,
  hjkl_cmd = ` + luaStr(stub) + `,
  why = { edit_idle_ms = 50 },
})
local why = require("hjkl.coach.why")
vim.wait(2000, why._enabled)
vim.cmd("edit " .. ` + luaStr(filepath.Join(dir, fileName)) + `)
-- 20 unrelated lines above and below: they must never be sent.
local filler = vim.fn["repeat"]({ "// unrelated private line" }, 20)
local body = { "package x", "", "foo := 1", "foo := 2", "foo := 3", "", "end" }
vim.api.nvim_buf_set_lines(0, 0, -1, false, vim.list_extend(vim.list_extend(vim.list_extend({}, filler), body), filler))
vim.api.nvim_win_set_cursor(0, { 23, 0 })
vim.api.nvim_feedkeys(vim.api.nvim_replace_termcodes(` + luaStr(keys) + `, true, false, true), "tx", false)
vim.wait(500, function() return why._last_edit() ~= nil end)
vim.cmd("HjklWhy")
vim.wait(3000, function() return why._float ~= nil end)
if why._float then log:write("FLOAT " .. table.concat(why._float.lines, " | ") .. "\n") end
log:close()
vim.cmd("qa!")
`
	luaPath := filepath.Join(dir, "t.lua")
	if err := os.WriteFile(luaPath, []byte(lua), 0o600); err != nil {
		t.Fatal(err)
	}
	if b, err := exec.Command("nvim", "--clean", "--headless", "-c", "luafile "+luaPath, "+qa!").CombinedOutput(); err != nil {
		t.Fatalf("nvim: %v\n%s", err, b)
	}
	if raw, err := os.ReadFile(sentPath); err == nil {
		if err := json.Unmarshal(raw, &sent); err != nil {
			t.Fatal(err)
		}
	}
	raw, _ := os.ReadFile(out)
	for _, l := range strings.Split(strings.TrimSpace(string(raw)), "\n") {
		switch {
		case strings.HasPrefix(l, "HINT "):
			hints = append(hints, strings.TrimPrefix(l, "HINT "))
		case strings.HasPrefix(l, "FLOAT "):
			float = strings.TrimPrefix(l, "FLOAT ")
		}
	}
	return sent, hints, float
}

// The wasteful rename: delete each foo letter by letter and retype it.
const wastefulRename = "xxxibar<Esc>jhhxxxibar<Esc>jhhxxxibar<Esc>"

func TestWhySendsOnlyTheEditAndShowsTheAnswer(t *testing.T) {
	sent, hints, float := runWhy(t, "main.go", wastefulRename)
	if sent == nil {
		t.Fatalf("nothing sent; hints=%q", hints)
	}
	// Only the changed lines plus 2 lines of context - none of the 40
	// unrelated lines around them.
	if got := sent["before"]; got != "package x\n\nfoo := 1\nfoo := 2\nfoo := 3\n\nend" {
		t.Fatalf("before=%q", got)
	}
	if got := sent["after"]; got != "package x\n\nbar := 1\nbar := 2\nbar := 3\n\nend" {
		t.Fatalf("after=%q", got)
	}
	if b, _ := json.Marshal(sent); strings.Contains(string(b), "unrelated") {
		t.Fatalf("unrelated lines were sent: %s", b)
	}
	if got := sent["keys"]; got != wastefulRename {
		t.Fatalf("keys=%q", got)
	}
	if c, _ := sent["cursor"].([]any); len(c) != 2 || c[0].(float64) != 3 || c[1].(float64) != 1 {
		t.Fatalf("cursor=%v", sent["cursor"])
	}
	if sent["filetype"] != "go" {
		t.Fatalf("filetype=%v", sent["filetype"])
	}
	offered := false
	for _, h := range hints {
		if strings.Contains(h, ":HjklWhy for a shorter way") {
			offered = true
		}
	}
	if !offered {
		t.Fatalf("a 30-key rename should be offered :HjklWhy; hints=%q", hints)
	}
	if !strings.Contains(float, "30 keys  →  11 keys") || !strings.Contains(float, "ciwbar<Esc>j.j.") {
		t.Fatalf("float=%q", float)
	}
}

func TestWhyNeverSendsExcludedFiles(t *testing.T) {
	sent, hints, _ := runWhy(t, "prod.env", wastefulRename)
	if sent != nil {
		t.Fatalf("an excluded file was sent: %v", sent)
	}
	for _, h := range hints {
		if strings.Contains(h, ":HjklWhy for a shorter way") {
			t.Fatalf("no offer for an excluded file: %q", hints)
		}
	}
}

func TestWhyNoOfferForEfficientEdits(t *testing.T) {
	_, hints, _ := runWhy(t, "main.go", "ciwbar<Esc>j.j.")
	for _, h := range hints {
		if strings.Contains(h, ":HjklWhy for a shorter way") {
			t.Fatalf("an efficient edit should not get an offer: %q", hints)
		}
	}
}
