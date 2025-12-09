package editor

import (
	tea "github.com/charmbracelet/bubbletea/v2"
)

func (editor *Editor) handleVisualMode(msg tea.KeyMsg) tea.Cmd {
	if msg.String() == "esc" {
		editor.Textarea.ResetSelection()
		editor.EnterNormalMode(true)
		return nil
	}

	return nil
}
