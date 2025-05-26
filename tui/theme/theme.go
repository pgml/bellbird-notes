package theme

import (
	"os"

	"github.com/charmbracelet/lipgloss"
	bl "github.com/winder/bubblelayout"
	"golang.org/x/term"
)

// @todo make this a theme.conf or whatever
var (
	ColourBorder        = lipgloss.Color("#424B5D")
	ColourBorderFocused = lipgloss.Color("#69c8dc")
	ColourFg            = lipgloss.NoColor{}
	BorderStyle         = lipgloss.RoundedBorder()
)

// BaseColumnLayout provides thae basic layout style for a column
func BaseColumnLayout(size bl.Size, focused bool) lipgloss.Style {
	borderColour := ColourBorder
	if focused {
		borderColour = ColourBorderFocused
	}

	_, termHeight := GetTerminalSize()

	return lipgloss.NewStyle().
		Border(BorderStyle).
		BorderForeground(borderColour).
		Foreground(ColourFg).
		Width(size.Width).
		Height(termHeight)
}

// GetTerminalSize determines the current
// Terminal providing a fallback and subtracting 1 from height
// because otherwise the upper part of the ui gets truncated
func GetTerminalSize() (int, int) {
	width, height, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		// Default width if terminal size can't be detected
		width = 80
	}
	return width, height - 1
}
