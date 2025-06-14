package components

import tea "github.com/charmbracelet/bubbletea/v2"

func (e *Editor) handleReplaceMode(msg tea.KeyMsg) tea.Cmd {
	if msg.String() == "esc" {
		e.EnterNormalMode()
		return nil
	}

	// only allow input when this flag is true.
	// See tui.updateComponents() for further explanation
	if e.CanInsert {
		// replace current charater in simple replace mode
		// convert string character to rune
		rune := []rune(msg.String())[0]
		e.checkDirty(func() {
			e.Textarea.ReplaceRune(rune)
		})
		e.EnterNormalMode()
	}

	return nil
}
