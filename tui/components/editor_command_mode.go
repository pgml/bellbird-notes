package components

import tea "github.com/charmbracelet/bubbletea/v2"

func (e *Editor) handleCommandMode(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "esc", "enter":
		e.EnterNormalMode(true)
		return nil
	}
	return nil
}
