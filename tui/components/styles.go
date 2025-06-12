package components

import (
	"github.com/charmbracelet/lipgloss/v2"

	"bellbird-notes/tui/theme"
)

type styles struct {
	base,
	enumerator,
	icon,
	selected,
	toggle lipgloss.Style
	iconWidth int
}

func NotesListStyle() styles {
	var s styles
	s.iconWidth = 3

	s.base = lipgloss.NewStyle().
		Foreground(lipgloss.NoColor{}).
		Width(25)
		//MarginLeft(0)
		//PaddingLeft(1)

	s.icon = s.base.
		Width(s.iconWidth)
		//Foreground(theme.ColourBorder)

	s.selected = s.base.
		Background(theme.ColourBgSelected).
		Bold(true)
	return s
}

func DirTreeStyle() styles {
	var s styles
	s.iconWidth = 2

	s.base = lipgloss.NewStyle().
		Foreground(lipgloss.NoColor{})

	s.icon = lipgloss.NewStyle().
		Width(s.iconWidth)
		//Foreground(theme.ColourBorder)

	s.selected = s.base.
		Background(theme.ColourBgSelected).
		Bold(true)
	return s
}
