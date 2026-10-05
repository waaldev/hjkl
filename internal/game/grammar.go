package game

// Grammar operators and motions shown on the grid.
var (
	GrammarOps     = []string{"d", "c", "y"}
	GrammarMotions = []string{"w", "b", "e", "$", "0", "G", "gg", "f", "t", "iw", "aw", `i"`, "a(", "ip", "it", "%"}
)

// LightFromKeys finds operator+motion pairs in a keystroke sequence.
func LightFromKeys(keys string) [][2]string {
	if keys == "" {
		return nil
	}
	var out [][2]string
	seen := map[string]struct{}{}
	add := func(op, mo string) {
		id := op + "\t" + mo
		if _, ok := seen[id]; ok {
			return
		}
		seen[id] = struct{}{}
		out = append(out, [2]string{op, mo})
	}
	runes := []rune(keys)
	motions := longestFirst(GrammarMotions)
	for i := 0; i < len(runes); i++ {
		op := string(runes[i])
		if op != "d" && op != "c" && op != "y" {
			continue
		}
		if i+1 >= len(runes) {
			continue
		}
		rest := string(runes[i+1:])
		for _, mo := range motions {
			if hasPrefixRunes(rest, mo) {
				add(op, mo)
				break
			}
		}
	}
	return out
}

func hasPrefixRunes(s, prefix string) bool {
	sr, pr := []rune(s), []rune(prefix)
	if len(sr) < len(pr) {
		return false
	}
	for i := range pr {
		if sr[i] != pr[i] {
			return false
		}
	}
	return true
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
