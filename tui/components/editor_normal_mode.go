package components

import (
	tea "github.com/charmbracelet/bubbletea/v2"
)

func (e *Editor) handleNormalMode(msg tea.KeyMsg) tea.Cmd {
	e.saveCursorPosToConf()

	isSearching := e.Textarea.Search.Query != ""
	if msg.String() == "esc" && isSearching {
		e.EnterNormalMode(true)
		e.Textarea.ResetMultiSelection()
		return nil
	}

	return nil
}
