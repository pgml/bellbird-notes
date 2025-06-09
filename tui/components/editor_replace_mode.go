package components

import tea "github.com/charmbracelet/bubbletea"

func (e *Editor) handleReplaceMode(msg tea.KeyMsg) tea.Cmd {
	if msg.String() == "esc" {
		e.EnterNormalMode()
		return nil
	}

	//var cmd tea.Cmd

	// only allow input when this flag is true.
	// See tui.updateComponents() for further explanation
	if e.CanInsert {
		// replace current charater in simple replace mode
		// convert string character to rune
		rune := []rune(msg.String())[0]
		oldCnt := e.CurrentBuffer.Content
		e.Textarea.ReplaceRune(rune)
		e.checkDirty(oldCnt)
		e.EnterNormalMode()
	}

	return nil
}
