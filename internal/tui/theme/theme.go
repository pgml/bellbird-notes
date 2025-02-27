package theme

import (
	"os"

	"github.com/charmbracelet/lipgloss"
	bl "github.com/winder/bubblelayout"
	"golang.org/x/term"
)

func BaseColumnLayout(size bl.Size, focused bool) lipgloss.Style {
	var borderColour lipgloss.TerminalColor = lipgloss.Color("#424B5D")
	if focused {
		borderColour = lipgloss.Color("#69c8dc")
	}

	_, termHeight := GetTerminalSize()

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColour).
		Foreground(lipgloss.NoColor{}).
		Width(size.Width).
		Height(termHeight - 3)
}

func GetTerminalSize() (int, int) {
	width, height, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		width = 80 // Default width if terminal size can't be detected
	}
	return width, height
}
