package shared

import (
	"github.com/charmbracelet/lipgloss/v2"

	"bellbird-notes/tui/theme"
)

type Styles struct {
	Base,
	Indent,
	Icon,
	Selected,
	IconSelected,
	Toggle lipgloss.Style

	IconWidth,
	ToggleWidth int
}

func NotesListStyle() Styles {
	var s Styles
	s.IconWidth = 3

	s.Base = lipgloss.NewStyle().
		Foreground(lipgloss.NoColor{}).
		Width(25)
		//MarginLeft(0)
		//PaddingLeft(1)

	s.Icon = s.Base.
		Width(s.IconWidth)
		//Foreground(theme.ColourBorder)

	s.Selected = s.Base.
		Background(theme.ColourBgSelected).
		Bold(true)
	return s
}

func DirTreeStyle() Styles {
	var s Styles
	s.IconWidth = 2
	s.ToggleWidth = 2

	s.Base = lipgloss.NewStyle().
		Foreground(lipgloss.NoColor{})

	s.Indent = s.Base.Foreground(theme.ColourBorder)

	s.Icon = lipgloss.NewStyle().
		Width(s.IconWidth)
	//Foreground(theme.ColourBorder)
	s.Toggle = s.Icon.
		Width(s.ToggleWidth).
		Foreground(theme.ColourBorder)

	s.Selected = s.Base.
		Background(theme.ColourBgSelected).
		Bold(true)
	return s
}
