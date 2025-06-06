package components

import (
	tea "github.com/charmbracelet/bubbletea"

	"bellbird-notes/tui/mode"
)

func (e *Editor) handleNormalMode(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "i":
		e.enterInsertMode()

	case "I":
		e.insertLineStart()

	case "a":
		e.inserAfter()

	case "A":
		e.insertLineEnd()

	case "r":
		e.Vim.Mode.Current = mode.Replace

	//case "v":
	//	e.Vim.Mode.Current = app.VisualMode

	case "h":
		e.moveCharacterLeft()

	case "l":
		e.moveCharacterRight()

	case "j":
		e.lineDown()

	case "k":
		e.lineUp()

	case "u":
		e.undo()

	case "ctrl+r":
		e.redo()

	case "w":
		e.wordRightStart()
		e.saveCursorPos()

	case "e":
		e.wordRightEnd()
		e.saveCursorPos()

	case "b":
		e.Textarea.WordLeft()
		e.saveCursorPos()

	case "^", "_":
		e.goToInputStart()

	case "0":
		e.goToLineStart()

	case "$":
		e.goToLineEnd()

	case "o":
		e.insertLineBelow()

	case "O":
		e.insertLineAbove()

	case "d":
		e.operator("d")

	case "D":
		e.Textarea.DeleteAfterCursor()

	case "g":
		e.operator("g")

	case "G":
		e.goToBottom()

	case "ctrl+d":
		e.Textarea.DownHalfPage()

	case "ctrl+u":
		e.Textarea.UpHalfPage()
	case ":":
		e.Vim.Mode.Current = mode.Command
	}

	e.Vim.Pending.ResetKeysDown()
	return nil
}
