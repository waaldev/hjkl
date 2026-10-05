package embeds

import "embed"

// FS holds curriculum YAML and the Neovim harness / coach plugin.
//
//go:embed curriculum nvim
var FS embed.FS
