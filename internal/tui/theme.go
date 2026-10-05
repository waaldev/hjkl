package tui

import (
	"github.com/charmbracelet/lipgloss"
)

var (
	moss      = lipgloss.Color("#8FBF5F")
	ink       = lipgloss.Color("#E8E4D0")
	muted     = lipgloss.Color("#7A7A6A")
	gold      = lipgloss.Color("#D4A017")
	crimson   = lipgloss.Color("#C23B22")
	paper     = lipgloss.Color("#1A1D17")
	highlight = lipgloss.Color("#C4E38A")
)

var beltColors = map[string]lipgloss.Color{
	"white":  lipgloss.Color("#E8E4D0"),
	"yellow": lipgloss.Color("#E6C229"),
	"orange": lipgloss.Color("#E07A3D"),
	"green":  lipgloss.Color("#3D8B5F"),
	"blue":   lipgloss.Color("#3A6EA5"),
	"purple": lipgloss.Color("#7A4E9D"),
	"brown":  lipgloss.Color("#8B5A2B"),
	"black":  lipgloss.Color("#9AA0A6"),
}

var (
	titleStyle = lipgloss.NewStyle().Foreground(moss).Bold(true)
	inkStyle   = lipgloss.NewStyle().Foreground(ink)
	mutedStyle = lipgloss.NewStyle().Foreground(muted)
	goldStyle  = lipgloss.NewStyle().Foreground(gold)
	badStyle   = lipgloss.NewStyle().Foreground(crimson)
	selStyle   = lipgloss.NewStyle().Foreground(paper).Background(highlight).Padding(0, 1)
	itemStyle  = lipgloss.NewStyle().Foreground(ink).Padding(0, 1)
	boxStyle   = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(moss).Padding(1, 2)
)

const logo = `  ██╗  ██╗     ██╗██╗  ██╗██╗
  ██║  ██║     ██║██║ ██╔╝██║
  ███████║     ██║█████╔╝ ██║
  ██╔══██║██   ██║██╔═██╗ ██║
  ██║  ██║╚█████╔╝██║  ██╗███████╗
  ╚═╝  ╚═╝ ╚════╝ ╚═╝  ╚═╝╚══════╝`

func starsBar(n int) string {
	s := ""
	for i := 1; i <= 3; i++ {
		if i <= n {
			s += "★"
		} else {
			s += "☆"
		}
	}
	if n > 0 {
		return goldStyle.Render(s)
	}
	return mutedStyle.Render(s)
}
