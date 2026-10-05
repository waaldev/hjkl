package coach

import (
	"strings"
	"unicode/utf8"
)

// Hit is a detected anti-pattern in a Normal-mode key stream.
type Hit struct {
	Pattern string
	Skill   string
	Hint    string
	Count   int
	Keys    string
}

var detectors = []func(string) *Hit{
	repeatMotion("j", 5, "repeat-j", "hjkl", "try a count (5j) or /search instead of jjjjj"),
	repeatMotion("k", 5, "repeat-k", "hjkl", "try a count (5k) or ?search instead of kkkkk"),
	repeatMotion("h", 5, "repeat-h", "line-ends", "try 0 ^ b or ge instead of hhhhh"),
	repeatMotion("l", 5, "repeat-l", "line-ends", "try $ e w or f{char} instead of lllll"),
	repeatMotion("x", 4, "repeat-x", "operators", "try dw / d<motion> / diw instead of xxxx"),
	repeatMotion("w", 5, "repeat-w", "search", "try / or f{char} or } instead of wwwww"),
	arrows,
	visualInnerWordDelete,
	insertArrowHabit,
	countedFriendly,
}

// Detect scans a key sequence (Vim notation: j, <Left>, viwd, …).
func Detect(keys string) []Hit {
	if keys == "" {
		return nil
	}
	var hits []Hit
	for _, d := range detectors {
		if h := d(keys); h != nil {
			hits = append(hits, *h)
		}
	}
	return hits
}

func repeatMotion(key string, n int, pattern, skill, hint string) func(string) *Hit {
	return func(keys string) *Hit {
		toks := tokenize(keys)
		best, run := 0, 0
		var runKeys []string
		var bestKeys []string
		for _, t := range toks {
			if t == key {
				run++
				runKeys = append(runKeys, t)
				if run > best {
					best = run
					bestKeys = append([]string{}, runKeys...)
				}
			} else {
				run = 0
				runKeys = nil
			}
		}
		if best >= n {
			return &Hit{Pattern: pattern, Skill: skill, Hint: hint, Count: best, Keys: strings.Join(bestKeys, "")}
		}
		return nil
	}
}

func arrows(keys string) *Hit {
	toks := tokenize(keys)
	n := 0
	for _, t := range toks {
		switch t {
		case "<Left>", "<Right>", "<Up>", "<Down>":
			n++
		}
	}
	if n >= 3 {
		return &Hit{
			Pattern: "arrow-keys",
			Skill:   "hjkl",
			Hint:    "hjkl (and motions) beat arrow keys - hands stay on the home row",
			Count:   n,
			Keys:    "arrows",
		}
	}
	return nil
}

func visualInnerWordDelete(keys string) *Hit {
	compact := strings.ReplaceAll(keys, "<Esc>", "")
	if strings.Contains(compact, "viwd") || strings.Contains(compact, "viw<d") {
		return &Hit{
			Pattern: "visual-iw-delete",
			Skill:   "text-objects",
			Hint:    "diw repeats with . - viwd does not",
			Count:   1,
			Keys:    "viwd",
		}
	}
	if strings.Contains(compact, "viwy") {
		return &Hit{
			Pattern: "visual-iw-yank",
			Skill:   "text-objects",
			Hint:    "yiw is the operator form",
			Count:   1,
			Keys:    "viwy",
		}
	}
	return nil
}

func insertArrowHabit(keys string) *Hit {
	// i … arrows … - moving around in Insert instead of Esc + motion.
	if strings.Contains(keys, "i<Left>") || strings.Contains(keys, "i<Right>") ||
		strings.Contains(keys, "i<Up>") || strings.Contains(keys, "i<Down>") {
		return &Hit{
			Pattern: "insert-arrows",
			Skill:   "modes",
			Hint:    "Esc to Normal, move, then i/a. <C-o> runs one Normal command without breaking the undo chunk",
			Count:   1,
			Keys:    "i+arrow",
		}
	}
	return nil
}

func countedFriendly(keys string) *Hit {
	toks := tokenize(keys)
	// d3w is fine; this flags a long count of j used as a scroll stand-in (already covered).
	// Flag: using Visual line + jjj instead of a count or operator.
	if strings.Contains(keys, "Vjjj") || strings.Contains(keys, "Vjjjj") {
		return &Hit{
			Pattern: "visual-line-walk",
			Skill:   "operators",
			Hint:    "2dd or d<count>j beats Vjjjd, and . can replay it",
			Count:   1,
			Keys:    "Vjjj",
		}
	}
	_ = toks
	return nil
}

func tokenize(s string) []string {
	var out []string
	for i := 0; i < len(s); {
		if s[i] == '<' {
			if j := strings.IndexByte(s[i:], '>'); j > 1 {
				out = append(out, s[i:i+j+1])
				i += j + 1
				continue
			}
		}
		r, size := utf8.DecodeRuneInString(s[i:])
		out = append(out, string(r))
		i += size
	}
	return out
}
