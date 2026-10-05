package keys

import (
	"strings"
	"unicode/utf8"
)

// Parse splits Vim key notation into one token per keystroke.
// Angle-bracket groups such as <Esc>, <CR> and <C-d> count as a single key.
func Parse(notation string) []string {
	if notation == "" {
		return nil
	}
	var out []string
	for i := 0; i < len(notation); {
		if notation[i] == '<' {
			if j := strings.IndexByte(notation[i:], '>'); j > 1 {
				tok := notation[i : i+j+1]
				if isAngleKey(tok) {
					out = append(out, tok)
					i += j + 1
					continue
				}
			}
		}
		r, size := utf8.DecodeRuneInString(notation[i:])
		out = append(out, string(r))
		i += size
	}
	return out
}

// Count returns the number of keystrokes in Vim key notation.
func Count(notation string) int {
	return len(Parse(notation))
}

func isAngleKey(tok string) bool {
	inner := tok[1 : len(tok)-1]
	if inner == "" {
		return false
	}
	switch inner {
	case "Esc", "CR", "NL", "Tab", "BS", "Del", "Space", "Up", "Down", "Left", "Right",
		"Enter", "Return", "lt", "Bar", "Bslash":
		return true
	}
	if strings.HasPrefix(inner, "C-") || strings.HasPrefix(inner, "A-") ||
		strings.HasPrefix(inner, "M-") || strings.HasPrefix(inner, "S-") ||
		strings.HasPrefix(inner, "D-") {
		return true
	}
	if strings.HasPrefix(inner, "F") && len(inner) <= 3 {
		return true
	}
	return false
}
