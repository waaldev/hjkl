package game

import (
	"strings"

	"github.com/waaldev/hjkl/internal/keys"
)

// Grammar operators and motions shown on the grid.
var (
	GrammarOps     = []string{"d", "c", "y"}
	GrammarMotions = []string{"w", "b", "e", "$", "0", "G", "gg", "f", "t", "iw", "aw", `i"`, "a(", "ip", "it", "%"}
)

// argKeys take the next key as an argument (f{char}, "{reg}, q{reg}, …),
// so that argument must never be read as a command.
var argKeys = map[string]bool{
	"f": true, "t": true, "F": true, "T": true, "r": true, "m": true,
	"'": true, "`": true, `"`: true, "q": true, "@": true,
}

// LightFromKeys finds operator+motion pairs in a command-key sequence
// (Normal/operator-pending keys only, never Insert-mode text). Counts are
// allowed on either side of the operator: 2dw, d2w.
func LightFromKeys(cmdKeys string) [][2]string {
	toks := keys.Parse(cmdKeys)
	motions := longestFirst(GrammarMotions)
	var out [][2]string
	seen := map[string]struct{}{}
	for i := 0; i < len(toks); {
		tok := toks[i]
		if argKeys[tok] {
			i += 2
			continue
		}
		if tok != "d" && tok != "c" && tok != "y" {
			i++
			continue
		}
		j := i + 1
		for j < len(toks) && (isCount(toks[j]) || (toks[j] == "0" && j > i+1)) {
			j++
		}
		if j < len(toks) && toks[j] == tok { // dd, cc, yy
			i = j + 1
			continue
		}
		rest := strings.Join(toks[j:], "")
		i = j
		for _, mo := range motions {
			if strings.HasPrefix(rest, mo) {
				id := tok + "\t" + mo
				if _, ok := seen[id]; !ok {
					seen[id] = struct{}{}
					out = append(out, [2]string{tok, mo})
				}
				i = j + len(keys.Parse(mo))
				if mo == "f" || mo == "t" {
					i++ // the {char} argument
				}
				break
			}
		}
	}
	return out
}

func isCount(tok string) bool {
	return len(tok) == 1 && tok[0] >= '0' && tok[0] <= '9' && tok != "0"
}

func longestFirst(in []string) []string {
	out := append([]string(nil), in...)
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if len([]rune(out[j])) > len([]rune(out[i])) {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out
}
