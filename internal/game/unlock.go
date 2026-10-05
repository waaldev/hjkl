package game

// BeltUnlockThreshold is the fraction of a belt that must be cleared
// (at least one star) before the next belt unlocks.
const BeltUnlockThreshold = 0.8

// Unlocked returns whether each belt is playable given per-challenge best stars.
func Unlocked(beltIDs []string, challengeBelts map[string]string, stars map[string]int) map[string]bool {
	out := make(map[string]bool, len(beltIDs))
	if len(beltIDs) == 0 {
		return out
	}
	out[beltIDs[0]] = true

	cleared := make(map[string]int)
	total := make(map[string]int)
	for id, belt := range challengeBelts {
		total[belt]++
		if stars[id] > 0 {
			cleared[belt]++
		}
	}

	for i := 1; i < len(beltIDs); i++ {
		prev := beltIDs[i-1]
		n := total[prev]
		if n == 0 {
			out[beltIDs[i]] = out[prev]
			continue
		}
		if float64(cleared[prev])/float64(n) >= BeltUnlockThreshold {
			out[beltIDs[i]] = true
		}
	}
	return out
}

func BeltComplete(beltID string, challengeBelts map[string]string, stars map[string]int) float64 {
	var n, ok int
	for id, b := range challengeBelts {
		if b != beltID {
			continue
		}
		n++
		if stars[id] > 0 {
			ok++
		}
	}
	if n == 0 {
		return 0
	}
	return float64(ok) / float64(n)
}
