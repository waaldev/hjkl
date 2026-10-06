// Command hjkl is a Neovim dojo: learn Vim from zero to hero by pressing
// the keys yourself, in real Neovim.
//
// It teaches in two places: short drills in a belt-based dojo, and a coach
// plugin that watches how you edit your own files. Run hjkl --help for the
// commands.
//
// Install:
//
//	go install github.com/waaldev/hjkl/cmd/hjkl@latest
package main

import "github.com/waaldev/hjkl/internal/cli"

func main() {
	cli.Execute()
}
