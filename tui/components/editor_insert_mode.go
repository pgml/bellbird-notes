package components

import tea "github.com/charmbracelet/bubbletea"

func (e *Editor) handleInsertMode(msg tea.KeyMsg) tea.Cmd {
	if msg.String() == "esc" {
		e.enterNormalMode()
		return nil
	}

	var cmd tea.Cmd

	e.Textarea, cmd = e.Textarea.Update(msg)
	e.checkDirty(e.CurrentBuffer.Content)

	return cmd
}
