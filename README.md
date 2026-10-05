# hjkl

Just for fun.

This is a little dojo for people who want to learn Vim in the AI era,
not because they have to, and not because an agent should type the keys
for them. You press them yourself, in real Neovim, because that is the
whole joke and the whole point.

Learning happens in two places: short, playful drills, and the files you
already edit. Practice turns into habit instead of staying a separate
tutorial. Drills run in **real Neovim**. Nothing here is a simulator.

```
  ██╗  ██╗     ██╗██╗  ██╗██╗
  ██║  ██║     ██║██║ ██╔╝██║
  ███████║     ██║█████╔╝ ██║
  ██╔══██║██   ██║██╔═██╗ ██║
  ██║  ██║╚█████╔╝██║  ██╗███████╗
  ╚═╝  ╚═╝ ╚════╝ ╚═╝  ╚═╝╚══════╝
  from zero to vim hero
```

## Requirements

- **Neovim** (`nvim` on your `PATH`). Classic Vim is not supported.
- A terminal. Drills take over the terminal and open a real `nvim`.

## Install

Pick one.

**Go** (needs Go 1.26+; put `$(go env GOPATH)/bin` on your `PATH`):

```bash
go install github.com/waaldev/hjkl/cmd/hjkl@latest
```

**Release binary** (no Go required): download `hjkl` for your OS from
[Releases](https://github.com/waaldev/hjkl/releases), unpack it, and move it
onto your `PATH`.

**From source:**

```bash
git clone https://github.com/waaldev/hjkl.git
cd hjkl
go build -o hjkl ./cmd/hjkl
./hjkl
```

Check it: `hjkl version`. Then `hjkl` to open the dojo.

## Quick start

```bash
hjkl                 # dojo TUI
hjkl learn           # same
hjkl learn white     # next White-belt drill (no TUI: --no-tui)
hjkl daily           # spaced reviews (no lesson shown - recall it)
hjkl stats           # XP, streak, Grammar Grid
hjkl cheat dw        # unlocked cheatsheet
hjkl coach install   # Neovim plugin snippet
hjkl dev verify      # every challenge solution, headless
```

Inside a drill: **F1** hint, **F10** abort. Matching the target (or landing on
the highlighted cell) in Normal mode writes a result and returns you to the
results card.

F1 climbs a hint ladder: the skill family, then a nudge, then the answer.
Each rung costs a star. When a run is short of three stars, **s** plays the
par solution in nvim, **r** retries, and **?** asks the AI sensei for a
debrief (if configured).

## The dojo

Belts unlock at 80% of the previous belt (at least one star per drill).

| Belt | Title | You leave able to |
|------|--------|-------------------|
| ⚪ White | Survive | modes, `i a o`, `:w :q :q!`, `hjkl`, `u` |
| 🟡 Yellow | Words and lines | `w b e`, `0 ^ $`, `dd yy p`, counts |
| 🟠 Orange | The grammar | operator + motion, `.` |
| 🟢 Green | Precision | `f t ; ,`, text objects, `%` |
| 🔵 Blue | Navigation | `/ ? n N * #`, marks, jumplist |
| 🟣 Purple | Power | Visual/block, registers, macros, `:s` |
| 🟤 Brown | Workspace | `:g`, `:norm`, ranges, files |
| ⚫ Black | Mastery | combinations, golf, your own config |

Challenge types: **Transform** (buffer matches the target split), **Navigate**
(cursor on the highlight), **Dot** (repeat with `.`), **Golf**, **Boss**.

Stars: at or under par = 3, within 2× par = 2, completed = 1, minus one per
hint. Drills about a technique (`.`, `;`, `cgn`, `g&`…) check that you used
it: the right result by another route earns one star.

Spaced repetition schedules each skill. Only the first attempt per skill per
day counts, speed counts (three stars *and* fluent is the top grade), and
`hjkl learn` mixes a due review in after every three new drills.

Pedagogy is Practical Vim, not a key catalog:

- Don't count, repeat. `dw..` vs `d3w` by how undo-friendly you want the result.
- Operator + motion = action. The Grammar Grid in `hjkl stats` lights up cells
  as you use them.
- Chunk your undos. Leave Insert at natural pauses.
- Prefer operators to Visual mode so `.` can replay the edit.
- Registers (`"0`, `"_`), macros (normalize / strike / abort), `cgn` + `.`,
  Ex ranges, jumplist - taught when the belt needs them.

## Coach (learning in daily work)

```bash
hjkl coach install
```

Paste the printed `lazy.nvim` snippet into your Neovim config. The plugin
watches command keys (Normal, operator-pending and Visual mode) in **your**
files, never the file contents or the text you type.

- It hints (throttled) on habits like `jjjjjjj`, `xxxxx`, arrow keys, and
  `viwd` instead of `diw`. Each hint is shown a few times (`hint_limit`,
  default 5), then fades.
- `:HjklDrill` opens a short drill for the last habit it flagged (or
  `:HjklDrill text-objects`). `:HjklSnooze` mutes that hint for a week.
- It also counts what you use well (`ciw`, `d2w`, `.`, `cgn`…). Real-world
  use lights the Grammar Grid and counts as a review for skills the dojo
  taught. After the first day, the first real use of a new command gets a
  short "nice".

Events land in `~/.local/share/hjkl/coach-events.jsonl` as counts.
`hjkl coach report` and `hjkl coach quest` read the aggregates. Quiet mode:
`opts = { quiet = true }`.

## AI sensei (optional)

The core game is offline and you still press every key. The sensei is
optional: it explains, writes practice, and plans - it never types for you.

Turn it on with an Anthropic key (`ANTHROPIC_API_KEY` or `hjkl ai setup`),
an OpenAI-compatible endpoint (including Ollama), or a CLI agent like
`opencode`.

```bash
hjkl ai list
hjkl ai setup --provider anthropic
hjkl ai test
hjkl ask "delete inside quotes"
hjkl drill --ai operators
```

Every generated challenge is checked the same way as `hjkl dev verify`.
If the solution misses the target or blows par, it is rejected. Debrief
stays inside skills you have unlocked, plus at most one next step. Only
aggregated coach stats would ever be sent, never file contents.

## Data

| | |
|--|--|
| config | `~/.config/hjkl/config.toml` |
| db | `~/.local/share/hjkl/hjkl.db` |
| coach events | `~/.local/share/hjkl/coach-events.jsonl` |

Override with `HJKL_CONFIG_DIR` and `HJKL_DATA_DIR`.

## Develop

```bash
go test ./...
go run ./cmd/hjkl dev verify
```

See [CONTRIBUTING.md](CONTRIBUTING.md) for writing lessons.

## License

MIT
