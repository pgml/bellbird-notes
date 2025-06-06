package components

import tea "github.com/charmbracelet/bubbletea"

func (e *Editor) handleReplaceMode(msg tea.KeyMsg) tea.Cmd {
	if msg.String() == "esc" {
		e.enterNormalMode()
		return nil
	}
	// replace current charater in simple replace mode
	// convert string character to rune
	rune := []rune(msg.String())[0]

	oldCnt := e.CurrentBuffer.Content
	e.Textarea.ReplaceRune(rune)
	e.checkDirty(oldCnt)
	e.enterNormalMode()

	return nil
}
