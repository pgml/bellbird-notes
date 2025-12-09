package editor

import (
	tea "github.com/charmbracelet/bubbletea/v2"
)

func (editor *Editor) handleNormalMode(msg tea.KeyMsg) tea.Cmd {
	editor.saveCursorPosToConf()

	isSearching := editor.Textarea.Search.Query != ""
	if msg.String() == "esc" && isSearching {
		editor.EnterNormalMode(true)
		editor.Textarea.ResetMultiSelection()
		return nil
	}

	return nil
}
