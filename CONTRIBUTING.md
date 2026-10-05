# Contributing

hjkl is a dojo, not a wiki. A lesson is a challenge whose **solution actually
works in Neovim**. `hjkl dev verify` is the gate.

## Add a drill

1. Pick a belt YAML under `embeds/curriculum/`.
2. Copy an existing challenge. Required fields:

```yaml
- id: orange-17-something   # unique, belt-prefixed
  title: Short verb phrase
  type: transform           # transform | navigate | dot | golf | speedrun | boss
  skills: [operators, words]
  principle: grammar        # see internal/curriculum/principles.go
  brief: One line. Shown in the nvim winbar neighborhood.
  teach: |
    What to understand before the fight. Practical Vim tone.
  hint: "The keys, if they are stuck."
  start: |
    before
  target: |
    after
  start_cursor: [1, 1]      # 1-indexed line, column
  par: 4                    # keystroke count of solution (<Esc> = 1)
  solution: "cwfoo<Esc>"
  language: go              # optional filetype
```

Navigate drills need `target_cursor` and usually omit a distinct `target`
(it defaults to `start`).

3. Set `par` to the keystroke count of `solution`. Angle-bracket keys
   (`<Esc>`, `<CR>`, `<C-d>`) count as one.

4. Run:

```bash
go test ./internal/curriculum
go run ./cmd/hjkl dev verify orange-17-something
```

5. If you add a skill id, add a cheatsheet line in `internal/curriculum/skills.go`.

## Principles

Tag the Practical Vim idea, not a key name. Prefer:

`modes`, `dot`, `grammar`, `chunk-undo`, `operators-over-visual`,
`text-objects`, `gn`, `ex-ranges`, `macros`, `registers`, `jumps`, `files`.

Solutions should demonstrate that idea. A `dot` drill whose solution never
presses `.` will still *work*, and the results card will nag - better to
include `.`.

## Harness

`embeds/nvim/harness.lua` is the only Neovim UI for drills. Keep user config
out (`nvim --clean`). Do not remap `hjkl` or operator keys. F1 is hint, F10
aborts.

## Coach detectors

Lua lives in `embeds/nvim/lua/hjkl/coach/init.lua`. The Go port (and the
table-driven tests) live in `internal/coach`. If you add a pattern, add it
in **both** places and a test on a key sequence. Never record file contents.

## AI

Generated challenges must pass `VerifyGate` (`runner.Verify` + par). Do not
merge a prompt change that lets a wrong solution through - there is a test
that feeds a deliberately bad drill and expects a reject.
