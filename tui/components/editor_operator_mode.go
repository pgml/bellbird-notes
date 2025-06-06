package components

import (
	tea "github.com/charmbracelet/bubbletea"

	"bellbird-notes/tui/mode"
)

func (e *Editor) handleOperatorMode(msg tea.KeyMsg) tea.Cmd {
	//origCnt = e.CurrentBuffer.Content
	if e.Vim.Pending.operator == "d" {
		switch msg.String() {
		case "d":
			e.Textarea.DeleteLine()

		case "j":
			e.Textarea.DeleteLines(2, false)

		case "k":
			e.Textarea.DeleteLines(2, true)

		case "w":
			e.Textarea.DeleteWordRight()
		}

		e.CurrentBuffer.History.NewEntry(e.Textarea.CursorPos())
	}

	if e.Vim.Pending.operator == "g" {
		switch msg.String() {
		case "g":
			e.goToTop()
		}
	}

	e.Vim.Pending.ResetKeysDown()
	e.Vim.Mode.Current = mode.Normal
	e.Vim.Pending.operator = ""

	return nil
}
