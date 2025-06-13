package components

import (
	tea "github.com/charmbracelet/bubbletea/v2"
)

func (e *Editor) handleVisualMode(msg tea.KeyMsg) tea.Cmd {
	if msg.String() == "esc" {
		e.Textarea.ResetSelection()
		e.EnterNormalMode()
		return nil
	}

	return nil
}
