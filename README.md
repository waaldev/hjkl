# hjkl

Just for fun: a dojo for learning Vim in the AI era. You press the keys
yourself, in real Neovim.

hjkl teaches in two places: short drills in a belt-based dojo, and a coach
plugin that watches how you edit your own files. Lessons follow *Practical
Vim*: operator + motion, the dot formula, text objects, Ex ranges, macros.

## Install

Needs Neovim (`nvim` on your `PATH`).

```bash
go install github.com/waaldev/hjkl/cmd/hjkl@latest
```

Or grab a binary from [Releases](https://github.com/waaldev/hjkl/releases).

## Use

```bash
hjkl              # the dojo
hjkl daily        # spaced-repetition reviews
hjkl stats        # progress and the Grammar Grid
hjkl --help       # everything else
```

In a drill: **F1** hint (costs a star), **F10** quit. After a miss, **s**
shows the solution and **r** retries.

## Coach

```bash
hjkl coach install
```

This copies the plugin and prints a lazy.nvim spec. Add it to your config.
For kickstart.nvim, save it as `lua/custom/plugins/hjkl.lua`.

The coach hints when it spots slow habits (`jjjjj`, `xxxx`, `viwd`) and
notices the good ones. `:HjklDrill` practices the last habit it flagged.
It stores counts only, never file contents.

## AI (optional)

Drills, the dojo and the coach all work offline. With a provider set up
(`hjkl ai setup`: Anthropic, any OpenAI-compatible endpoint including
Ollama, or a CLI agent like opencode), hjkl can explain your runs, answer
questions (`hjkl ask`, followed by a drill to practice the answer) and
generate drills. Generated drills are replayed in Neovim and dropped if
they don't work; the ones that pass are saved for your reviews.

- `hjkl coach review` writes a weekly review from your coach counts and
  sets quests; `hjkl coach quest` tracks them.
- `hjkl boss <file>` builds a multi-step boss fight from a block of your
  own code. It asks before sending anything and refuses secret files.

**`:HjklWhy`** finds a shorter way to make your last edit in your own
project. The suggestion is replayed in Neovim and shown only if it
produces the same result in fewer keys. You can practice it or save it
as a drill.

This is the only feature that sends code, so it is off until you enable it:

```toml
# ~/.config/hjkl/config.toml
[ai]
suggest = true
```

It sends only the lines of that one edit plus a little context, never the
whole file, and never files that look like secrets (`.env`, keys,
credentials). Use a local model to keep everything on your machine.

## Contributing

```bash
go test ./...                  # nvim-backed tests skip without nvim
go run ./cmd/hjkl dev verify   # replays every drill's solution
```

Drills are YAML in `embeds/curriculum/`. [CONTRIBUTING.md](CONTRIBUTING.md)
covers the format.

## License

MIT
