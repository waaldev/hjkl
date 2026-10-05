package game

import "strings"

// Stars awards 1–3 stars from keystrokes versus par.
func Stars(ok bool, keyCount, par int) int {
	if !ok {
		return 0
	}
	if par <= 0 {
		return 1
	}
	if keyCount <= par {
		return 3
	}
	if keyCount <= par*2 {
		return 2
	}
	return 1
}

// HintCap removes one star per hint used, never below one for a clear.
func HintCap(stars, hintsUsed int) int {
	if stars <= 0 || hintsUsed <= 0 {
		return stars
	}
	stars -= hintsUsed
	if stars < 1 {
		stars = 1
	}
	return stars
}

// HintQuality lowers SRS quality for help taken: any hint makes it at best
// a hesitant recall (3); seeing the answer means it was not recalled (2).
func HintQuality(quality, hintsUsed int, sawAnswer bool) int {
	switch {
	case sawAnswer && quality > 2:
		return 2
	case hintsUsed > 0 && quality > 3:
		return 3
	}
	return quality
}

// XP for a single attempt. firstClear adds a one-time bonus.
func XP(beltRank, stars int, firstClear bool) int {
	if stars <= 0 {
		return 0
	}
	xp := 10 * (beltRank + 1)
	if stars == 3 {
		xp += 5
	}
	if firstClear {
		xp += 10
	}
	return xp
}

// Quality maps an attempt onto the SM-2 0–5 scale.
func Quality(ok bool, stars, keyCount, par int) int {
	if !ok {
		return 1
	}
	switch stars {
	case 3:
		if par > 0 && keyCount < par {
			return 5
		}
		return 5
	case 2:
		return 3
	default:
		return 2
	}
}

// DotScore describes whether the run used Vim's repeat command.
func DotScore(keys, principle string) string {
	used := strings.Contains(keys, ".")
	switch {
	case used && principle == "dot":
		return "Dot Score: you let . do the repeating. That's the Vim way."
	case used:
		return "Dot Score: the dot command showed up - keep leaning on it."
	case principle == "dot":
		return "Dot Score: this edit was designed to be repeated with . Try it next time."
	default:
		return ""
	}
}
