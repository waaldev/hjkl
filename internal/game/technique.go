package game

import (
	"fmt"
	"regexp"
)

// CheckTechnique reports whether the command keys satisfy a drill's
// require/forbid rules. msg explains the first rule that failed.
func CheckTechnique(cmdKeys string, require, forbid []string) (ok bool, msg string, err error) {
	for _, pat := range require {
		re, err := regexp.Compile(pat)
		if err != nil {
			return false, "", fmt.Errorf("require %q: %w", pat, err)
		}
		if !re.MatchString(cmdKeys) {
			return false, fmt.Sprintf("solved, but this drill is about %s - try it that way", pat), nil
		}
	}
	for _, pat := range forbid {
		re, err := regexp.Compile(pat)
		if err != nil {
			return false, "", fmt.Errorf("forbid %q: %w", pat, err)
		}
		if loc := re.FindStringIndex(cmdKeys); loc != nil {
			return false, fmt.Sprintf("solved, but %q is the habit this drill replaces", cmdKeys[loc[0]:loc[1]]), nil
		}
	}
	return true, "", nil
}
