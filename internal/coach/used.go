package coach

import (
	"strings"

	"github.com/waaldev/hjkl/internal/game"
)

// motionSkills maps a grid motion to the skill it exercises.
var motionSkills = map[string]string{
	"w": "words", "b": "words", "e": "words",
	"$": "line-ends", "0": "line-ends",
	"f": "find-char", "t": "find-char",
	"iw": "text-objects", "aw": "text-objects", `i"`: "text-objects",
	"a(": "text-objects", "ip": "text-objects", "it": "text-objects",
	"%": "percent",
}

var singleSkills = map[string]string{
	".": "dot", ";": "find-char", ",": "find-char",
	"n": "search", "N": "search", "*": "search", "#": "search",
	"%": "percent", "@": "macros",
}

// UsedSkills returns the skills a real-world command exercises
// ("ciw" → operators, text-objects).
func UsedSkills(keys string) []string {
	if s, ok := singleSkills[keys]; ok {
		return []string{s}
	}
	if keys == "dd" || keys == "yy" {
		return []string{"dd-yy-p"}
	}
	out := []string{}
	for _, pair := range game.LightFromKeys(keys) {
		out = append(out, "operators")
		if s, ok := motionSkills[pair[1]]; ok {
			out = append(out, s)
		}
	}
	if strings.ContainsAny(keys, "123456789") && len(out) > 0 {
		out = append(out, "counts")
	}
	return out
}
