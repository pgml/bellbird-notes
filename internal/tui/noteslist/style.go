package noteslist

import "github.com/charmbracelet/lipgloss"

type styles struct {
	base,
	enumerator,
	note,
	toggle lipgloss.Style
}

func defaultStyles() styles {
	var s styles
	s.base = lipgloss.NewStyle().
		Foreground(lipgloss.NoColor{}).
		MarginLeft(0).
		PaddingLeft(1)
	s.note = s.base.
		MarginRight(0).
		PaddingLeft(1).
		PaddingRight(0).
		Foreground(lipgloss.AdaptiveColor{Light: "#333", Dark: "#eee"})
	return s
}
