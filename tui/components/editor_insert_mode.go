package components

import tea "github.com/charmbracelet/bubbletea/v2"

func (e *Editor) handleInsertMode(msg tea.KeyMsg) tea.Cmd {
	if msg.String() == "esc" {
		e.EnterNormalMode()
		return nil
	}

	var cmd tea.Cmd

	// only allow input when this flag is true.
	// See tui.updateComponents() for further explanation
	if e.CanInsert {
		e.Textarea, cmd = e.Textarea.Update(msg)
	}

	return cmd
}
