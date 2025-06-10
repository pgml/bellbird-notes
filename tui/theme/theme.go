package theme

import (
	"os"

	"bellbird-notes/app"

	"github.com/charmbracelet/lipgloss"
	bl "github.com/winder/bubblelayout"
	"golang.org/x/term"
)

// @todo make this a theme.conf or whatever
// colors
var (
	ColourBorder        = lipgloss.AdaptiveColor{Dark: "#424B5D"}
	ColourBorderFocused = lipgloss.AdaptiveColor{Dark: "#69c8dc"}
	ColourFg            = lipgloss.NoColor{}
	ColourBgSelected    = lipgloss.AdaptiveColor{Light: "#333", Dark: "#424B5D"}
	ColourDirty         = lipgloss.AdaptiveColor{Light: "#333", Dark: "#c05d5f"}

	BorderStyle = lipgloss.RoundedBorder()
)

type icon struct {
	Nerd string
	Alt  string
}

var (
	IconPen       = icon{Nerd: "", Alt: ">"}
	IconNote      = icon{Nerd: "󰎞", Alt: ""}
	IconDirOpen   = icon{Nerd: "", Alt: "▼"}
	IconDirClosed = icon{Nerd: "󰉋", Alt: "▶"}
	IconDot       = icon{Nerd: "", Alt: "*"}
)

func Icon(icon icon) string {
	icn := icon.Nerd
	if *app.NoNerdFonts {
		icn = icon.Alt
	}
	return icn
}

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
