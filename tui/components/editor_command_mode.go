package components

import tea "github.com/charmbracelet/bubbletea"

func (e *Editor) handleCommandMode(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "esc", "enter":
		e.EnterNormalMode()
		return nil
	}
	return nil
}
