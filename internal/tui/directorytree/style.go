package directorytree

import "github.com/charmbracelet/lipgloss"

type styles struct {
	base,
	enumerator,
	dir,
	toggle lipgloss.Style
}

func defaultStyles() styles {
	var s styles
	s.base = lipgloss.NewStyle().
		Foreground(lipgloss.NoColor{}).
		MarginLeft(0).
		PaddingLeft(1)
	s.dir = s.base.
		MarginRight(0).
		PaddingRight(0).
		Foreground(lipgloss.AdaptiveColor{Light: "#333", Dark: "#eee"})
	return s
}
