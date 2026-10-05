package ai

import (
	"path/filepath"
	"strings"
)

// secretGlobs match files that must never be sent to a provider. The coach
// plugin keeps the same list for :HjklWhy.
var secretGlobs = []string{"*.env", ".env*", "*.pem", "*.key", "*secret*", "*credential*", "*password*"}

// IsSecretPath reports whether a file looks like it holds secrets. Name
// globs match the file name; *word* globs match anywhere in the path, so
// secrets/config.go counts too.
func IsSecretPath(path string) bool {
	full := strings.ToLower(filepath.ToSlash(path))
	base := filepath.Base(full)
	for _, g := range secretGlobs {
		if ok, _ := filepath.Match(g, base); ok {
			return true
		}
		if word := strings.Trim(g, "*"); strings.HasPrefix(g, "*") && strings.HasSuffix(g, "*") && strings.Contains(full, word) {
			return true
		}
	}
	return false
}
